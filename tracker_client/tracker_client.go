// For more details see :
// https://www.rasterbar.com/products/libtorrent/udp_tracker_protocol.html
// https://wiki.theory.org/BitTorrentSpecification#peer_id
// https://www.bittorrent.org/beps/bep_0015.html
package trackerclient

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"gotorrent/decoder"
	"gotorrent/encoder"
	"gotorrent/utils"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

// This will identify the protocol.
const udpTrackerProtocolMagicNumber int64 = 0x41727101980

// Ip + port size for one peer returned in udp
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

// @TODO: maybe reuse udp connections ??
// @TODO: implement "Multitracker Metadata Extension" https://bittorrent.org/beps/bep_0012.html
type TrackerClient struct {
	torrentFile      decoder.TorrentFile
	announceUrl      *url.URL
	announceInterval int32
	infoHash         string
	numPeersWant     int32

	peers      []UdpPeer
	downloaded int64
	left       int64
	uploaded   int64
	status     int32

	// this is used for udp and it can expire
	// @TODO: check if connectionId expired before sending udp req
	connectionId int64
	peerId       []byte

	mu sync.Mutex
}

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
	}

	if strings.HasPrefix(u.Scheme, "http") && strings.HasPrefix(u.Scheme, "udp") {
		return nil, fmt.Errorf("Unsupported protofol for %s, we only support HTTP/UDP", u.Scheme)
	}

	return &TrackerClient{
		torrentFile:  torrentFile,
		announceUrl:  u,
		infoHash:     infoHash,
		numPeersWant: 5,

		downloaded: 0,
		left:       int64(torrentFile.Info.Length),
		status:     none,

		peerId: generateRandomPeerId(),
	}, nil
}

func (tc *TrackerClient) Start(ctx context.Context, chErr chan<- error) {
	for {
		d, err := time.ParseDuration(fmt.Sprintf("%ds", tc.announceInterval))
		if err != nil {
			chErr <- fmt.Errorf("Couldn't parse %ds into duration got err '%s' instead!", tc.announceInterval, err)
			return
		}

		interval := time.After(d)
		log.Printf("Waiting for %s (%d seconds) before sending announce req \n", d, tc.announceInterval)
		select {
		case <-ctx.Done():
			return
		case <-interval:
			log.Printf("Sending announce request\n")
			resp, err := tc.announce()
			if err != nil {
				chErr <- err
				return
			}
			log.Printf("Announce response %+v\n", resp)

			tc.mu.Lock()

			// @TODO: fill the rest of the values
			tc.announceInterval = resp.Interval
			tc.peers = resp.Peers

			tc.mu.Unlock()
		}
	}
}

type UdpPeer struct {
	Ip   netip.Addr
	Port uint16
}

func (tc *TrackerClient) GetPeers() []UdpPeer {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	peers := make([]UdpPeer, len(tc.peers))
	copy(peers, tc.peers)

	return peers
}

type announceResponse struct {
	// The number of seconds you should wait until re-announcing yourself.
	Interval int32

	Leechers int32
	Seeders  int32

	Peers []UdpPeer
}

func (tc *TrackerClient) announce() (*announceResponse, error) {
	if strings.Index(tc.announceUrl.Scheme, "http") == 0 {
		return tc.sendHTTPAnnounceRequest()
	}
	return tc.sendUDPAnnounceRequest()
}

// this does not yet work and it's badly tested
// @TODO: test this
func (tc *TrackerClient) sendHTTPAnnounceRequest() (*announceResponse, error) {
	u, err := tc.getHttpTrackerUrl()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("Sending an HTTP GET to %s\n", u)

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
	if failureReason != "" {
		return nil, fmt.Errorf("%s", failureReason)
	}

	interval, _ := body["interval"].(int)
	leechers, _ := body["leechers"].(int)

	peers := make([]UdpPeer, 0)
	anyPeers, _ := body["peers"].([]any)
	for _, p := range anyPeers {
		v, ok := p.(map[string]any)
		if ok {
			port, _ := v["port"].(int)

			ip, _ := v["ip"].(int)
			buff := new(bytes.Buffer)
			if err := binary.Write(buff, binary.NativeEndian, ip); err != nil {
				return nil, err
			}

			peers = append(peers, UdpPeer{
				Ip:   netip.AddrFrom4([4]byte(buff.Bytes())),
				Port: uint16(port),
			})
		} else {
			// if it's not a dict then it should be a byte string
			v, ok := p.(string)
			if !ok {
				return nil, errors.New("Expected peer to either be dict or byte arr")
			}

			r := bytes.NewBuffer([]byte(v))
			var (
				ip   [4]byte
				port uint16
			)

			if err := binary.Read(r, binary.NativeEndian, &ip); err != nil {
				return nil, err
			}

			if err := binary.Read(r, binary.NativeEndian, &port); err != nil {
				return nil, err
			}

			peers = append(peers, UdpPeer{
				Ip:   netip.AddrFrom4(ip),
				Port: uint16(port),
			})
		}
	}

	return &announceResponse{
		Interval: int32(interval),
		Leechers: int32(leechers),
		Peers:    peers,
	}, nil
}

func (tc *TrackerClient) sendUDPAnnounceRequest() (*announceResponse, error) {
	err := tc.setUpUDPConnectionId()
	if err != nil {
		return nil, err
	}

	conn, err := utils.ConnectToUDPURL(tc.announceUrl)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	log.Printf("Set up upd connection to %s\n", tc.announceUrl)

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
	binary.Read(r, binary.NativeEndian, &action)
	binary.Read(r, binary.NativeEndian, &transactionId)
	binary.Read(r, binary.NativeEndian, &interval)
	binary.Read(r, binary.NativeEndian, &leechers)
	binary.Read(r, binary.NativeEndian, &seeders)

	peers := make([]UdpPeer, 0)
	for {
		var (
			ip   [4]byte
			port uint16
		)

		if err := binary.Read(r, binary.NativeEndian, &ip); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if err := binary.Read(r, binary.NativeEndian, &port); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// I'm not sure why but tracker always returns "numPeersWant" even if the tracker does not have
		// that much so i will end up with port=0, ip=0.0.0.0 repating till "numPeersWant"
		// thus i'm ending reading as soon as i get this case.
		if port == 0 {
			break
		}

		peers = append(peers, UdpPeer{
			Ip:   netip.AddrFrom4(ip),
			Port: port,
		})
	}

	if transactionId != randomTransactionId {
		return nil, fmt.Errorf("Received different transaction_id, sent %d and got %d", randomTransactionId, transactionId)
	}

	return &announceResponse{Interval: interval, Leechers: leechers, Seeders: seeders, Peers: peers}, nil
}

func (tc *TrackerClient) writeAnnounceRequest(req *bytes.Buffer, transactionId int32) error {
	write := func(buff *bytes.Buffer, data any) error {
		if err := binary.Write(buff, binary.NativeEndian, data); err != nil {
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
	log.Printf("Set up upd connection to %s\n", tc.announceUrl)

	connReq := new(bytes.Buffer)

	if err = binary.Write(connReq, binary.NativeEndian, udpTrackerProtocolMagicNumber); err != nil {
		return err
	}

	if err := binary.Write(connReq, binary.NativeEndian, udpConnect); err != nil {
		return err
	}

	var randomTransactionId int32 = rand.Int31()
	if err := binary.Write(connReq, binary.NativeEndian, randomTransactionId); err != nil {
		return err
	}
	log.Printf("Send UDP init conn packets to %s\n", tc.announceUrl)

	if _, err := conn.Write(connReq.Bytes()); err != nil {
		return err
	}

	timeout, _ := time.ParseDuration("1m")
	conn.SetReadDeadline(time.Now().Add(timeout))
	resp := make([]byte, 16)
	if _, err := conn.Read(resp); err != nil {
		return err
	}

	var (
		action, transactionId int32
		connectionId          int64
	)

	r := bytes.NewBuffer(resp)
	binary.Read(r, binary.NativeEndian, &action)
	binary.Read(r, binary.NativeEndian, &transactionId)
	binary.Read(r, binary.NativeEndian, &connectionId)

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
