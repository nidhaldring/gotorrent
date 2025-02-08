// For more details see :
// https://www.rasterbar.com/products/libtorrent/udp_tracker_protocol.html
// https://wiki.theory.org/BitTorrentSpecification#peer_id
// https://www.bittorrent.org/beps/bep_0015.html
package trackerclient

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"gotorrent/decoder"
	"gotorrent/encoder"
	"gotorrent/utils"
	"io"
	"math/rand"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// @TODO: maybe reuse udp connections ??
// @TODO: implement "Multitracker Metadata Extension" https://bittorrent.org/beps/bep_0012.html
type TrackerClient struct {
	announceUrl  *url.URL
	torrentFile  decoder.TorrentFile
	infoHash     string
	numPeersWant int32

	downloaded int64
	left       int64
	uploaded   int64
	status     int32

	// this is used for udp and it can expire
	// @TODO: check if connectionId expired before sending udp req
	connectionId int64
	peerId       []byte
}

// This will identify the protocol.
const udpTrackerProtocolMagicNumber int64 = 0x41727101980

// ip + port size for one peer returned in udp
const peerStructureSize = 6

// List of actions sent to tracker
const (
	udpConnect int32 = iota
	udpAnnounce
	udpScrape
	udpError
)

// List of possible status the clients can have
const (
	none int32 = iota
	completed
	started
	stopped
)

func NewTrackerClient(torrentFile decoder.TorrentFile) (*TrackerClient, error) {
	info, err := encoder.Encode(torrentFile.Info)
	if err != nil {
		return nil, err
	}

	h := sha1.New()
	if _, err := io.WriteString(h, info); err != nil {
		return nil, err
	}
	infoHash := string(h.Sum(nil))

	u, err := url.Parse(torrentFile.Announce)
	if err != nil {
		return nil, err
	} else if strings.HasPrefix(u.Scheme, "http") && strings.HasPrefix(u.Scheme, "udp") {
		return nil, fmt.Errorf("Unsupported protofol for %s, we only support HTTP/UDP", u.Scheme)
	}

	return &TrackerClient{
		announceUrl:  u,
		torrentFile:  torrentFile,
		infoHash:     infoHash,
		numPeersWant: 5,

		downloaded: 0,
		left:       int64(torrentFile.Info.Length),
		status:     none,

		peerId: generateRandomPeerId(),
	}, nil
}

type udpPeer struct {
	Ip   netip.Addr
	Port uint16
}

type trackerResponse struct {
	// This will only be populated if response failed
	FailureReason string

	// The number of seconds you should wait until re-announcing yourself.
	Interval int32
	Leechers int32
	Seeders  int32

	Peers []udpPeer
}

func (tc *TrackerClient) Announce() (*trackerResponse, error) {
	if strings.Index(tc.announceUrl.Scheme, "http") == 0 {
		return tc.sendHTTPAnnounceRequest()
	}
	return tc.sendUDPAnnounceRequest()
}

// this does not yet work and it's badly tested
// @TODO: test this
func (tc *TrackerClient) sendHTTPAnnounceRequest() (*trackerResponse, error) {
	u, err := tc.getHttpTrackerUrl()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP[%d] while calling tracker server %s\n %s", resp.StatusCode, u, string(b))
	}

	body, err := decoder.Decode(string(b))
	if err != nil {
		return nil, err
	}

	failureReason, _ := body["failure reason"].(string)
	interval, _ := body["interval"].(int)
	leechers, _ := body["leechers"].(int)

	peers := make([]udpPeer, 0)
	anyPeers, _ := body["peers"].([]any)
	for _, p := range anyPeers {
		v, ok := p.(map[string]any)
		if ok {
			port, _ := v["port"].(int)

			ip, _ := v["ip"].(int)
			buff := new(bytes.Buffer)
			// @TODO: use system specific "endianness"
			if err := binary.Write(buff, binary.LittleEndian, ip); err != nil {
				return nil, err
			}

			peers = append(peers, udpPeer{
				Ip:   netip.AddrFrom4([4]byte(buff.Bytes())),
				Port: uint16(port),
			})
		} else {
			// it should be a byte string as in udp
			v, ok := p.(string)
			if !ok {
				return nil, errors.New("Expected peer to either be dict or byte arr")
			}

			r := bytes.NewBuffer([]byte(v))
			var (
				ip   [4]byte
				port uint16
			)

			if err := binary.Read(r, binary.BigEndian, &ip); err != nil {
				return nil, err
			}

			if err := binary.Read(r, binary.BigEndian, &port); err != nil {
				return nil, err
			}

			peers = append(peers, udpPeer{
				Ip:   netip.AddrFrom4(ip),
				Port: uint16(port),
			})
		}
	}

	return &trackerResponse{
		FailureReason: failureReason,
		Interval:      int32(interval),
		Leechers:      int32(leechers),
		Peers:         peers,
	}, nil
}

func (tc *TrackerClient) sendUDPAnnounceRequest() (*trackerResponse, error) {
	err := tc.setUpUDPConnectionId()
	if err != nil {
		return nil, err
	}

	conn, err := utils.ConnectToUDPURL(tc.announceUrl)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	announce := new(bytes.Buffer)
	var randomTransactionId int32 = rand.Int31()
	tc.writeAnnounceRequest(announce, randomTransactionId)

	if _, err := conn.Write(announce.Bytes()); err != nil {
		return nil, err
	}

	resp := make([]byte, 20+tc.numPeersWant*peerStructureSize)
	// @TODO: there must be a way to time this out or it will wait forever
	if _, err := conn.Read(resp); err != nil {
		return nil, err
	}

	var (
		action, transactionId, interval, leechers, seeders int32
	)
	r := bytes.NewBuffer(resp)
	binary.Read(r, binary.BigEndian, &action)
	binary.Read(r, binary.BigEndian, &transactionId)
	binary.Read(r, binary.BigEndian, &interval)
	binary.Read(r, binary.BigEndian, &leechers)
	binary.Read(r, binary.BigEndian, &seeders)

	peers := make([]udpPeer, 0)
	for {
		var (
			ip   [4]byte
			port uint16
		)

		if err := binary.Read(r, binary.BigEndian, &ip); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if err := binary.Read(r, binary.BigEndian, &port); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// I'm not sure why but tracker always returns "numPeersWant" even if the tracker does not have
		// that much so i will end up with port=0, ip=0.0.0.0 thus i'm ending reading as soon as i get this case
		if port == 0 {
			break
		}

		peers = append(peers, udpPeer{
			Ip:   netip.AddrFrom4(ip),
			Port: port,
		})
	}

	if transactionId != randomTransactionId {
		return nil, fmt.Errorf("Received different transaction_id, sent %d and got %d", randomTransactionId, transactionId)
	}

	return &trackerResponse{Interval: interval, Leechers: leechers, Seeders: seeders, Peers: peers}, nil
}

func (tc *TrackerClient) writeAnnounceRequest(req *bytes.Buffer, transactionId int32) error {
	write := func(buff *bytes.Buffer, data any) error {
		if err := binary.Write(buff, binary.BigEndian, data); err != nil {
			return err
		}
		return nil
	}

	var (
		downloaded int64  = 0
		left       int64  = int64(tc.torrentFile.Info.Length)
		uploaded   int64  = 0
		ip         uint32 = 0
		key        uint32 = rand.Uint32()
		port       uint16 = 6881
		extension  uint16 = 0
	)

	values := []any{
		tc.connectionId,
		udpAnnounce,
		transactionId,
		[]byte(tc.infoHash),
		tc.peerId,
		downloaded,
		left,
		uploaded,
		started,
		ip,
		key,
		tc.numPeersWant,
		port,
		extension,
	}

	for _, v := range values {
		if err := write(req, v); err != nil {
			return err
		}
	}

	return nil
}

func (tc *TrackerClient) setUpUDPConnectionId() error {
	conn, err := utils.ConnectToUDPURL(tc.announceUrl)
	if err != nil {
		return err
	}
	defer conn.Close()

	connReq := new(bytes.Buffer)

	if err = binary.Write(connReq, binary.BigEndian, udpTrackerProtocolMagicNumber); err != nil {
		return err
	}

	if err := binary.Write(connReq, binary.BigEndian, udpConnect); err != nil {
		return err
	}

	var randomTransactionId int32 = rand.Int31()
	if err := binary.Write(connReq, binary.BigEndian, randomTransactionId); err != nil {
		return err
	}

	if _, err := conn.Write(connReq.Bytes()); err != nil {
		return err
	}

	resp := make([]byte, 16)
	// @TODO: there must be a way to time this out or it will wait forever
	if _, err := conn.Read(resp); err != nil {
		return err
	}

	var (
		action, transactionId int32
		connectionId          int64
	)

	r := bytes.NewBuffer(resp)
	binary.Read(r, binary.BigEndian, &action)
	binary.Read(r, binary.BigEndian, &transactionId)
	binary.Read(r, binary.BigEndian, &connectionId)

	if transactionId != randomTransactionId {
		return fmt.Errorf("Received different transaction_id, sent %d and got %d", randomTransactionId, transactionId)
	}

	if action == udpError {
		return errors.New("Received an error action from tracker server")
	}

	tc.connectionId = connectionId

	return nil
}

func (tc *TrackerClient) getHttpTrackerUrl() (*url.URL, error) {
	params := url.Values{}

	params.Add("info_hash", tc.infoHash)
	params.Add("peer_id", string(tc.peerId))
	params.Add("event", string(tc.status))

	ip, err := getCurrentIp()
	if err != nil {
		return nil, err
	}
	params.Add("ip", ip)

	trackerUrl, err := url.Parse(fmt.Sprintf("%s?%s", tc.announceUrl, params.Encode()))
	if err != nil {
		return nil, err
	}

	return trackerUrl, nil
}
