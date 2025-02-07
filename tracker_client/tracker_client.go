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
	"net/url"
	"strings"
)

// @TODO: maybe reuse udp connections ??
type TrackerClient struct {
	url         *url.URL
	torrentFile decoder.TorrentFile
	infoHash    string

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

// List of actions sent to tracker
const (
	udpConnect int32 = iota
	udpAnnounce
	udpScrape
	udpError
)

// list of possible status the clients can have
const (
	none int32 = iota
	completed
	started
	stopped
)

type TrackerResponse struct {
	// @TODO: FailureReason this is optional
	FailureReason string
	Interval      int32
	Peers         []struct {
		Id   string
		Ip   string
		Port string
	}
}

func NewTrackerClient(torrentFile decoder.TorrentFile) (*TrackerClient, error) {
	info, err := encoder.Encode(torrentFile.Info)
	if err != nil {
		return nil, err
	}

	h := sha1.New()
	io.WriteString(h, info)
	infoHash := string(h.Sum(nil))

	return &TrackerClient{
		peerId:      generateRandomPeerId(),
		downloaded:  0,
		torrentFile: torrentFile,
		infoHash:    infoHash,
		left:        int64(torrentFile.Info.Length),
		status:      none,
	}, nil
}

func (tc *TrackerClient) Announce() (*TrackerResponse, error) {
	serverUrls := getUrls(tc.torrentFile)
	var resp *TrackerResponse = nil
	for _, u := range serverUrls {
		trackerUrl, err := tc.prepareTrackerUrl(u)
		if err != nil {
			return nil, err
		}

		tmpResp, err := tc.sendAnnounceRequest(trackerUrl)
		if err != nil {
			fmt.Printf("[Error]: %s\n", err)
			continue
		}

		if tmpResp != nil {
			resp = tmpResp
			break
		}
	}

	if resp == nil {
		return nil, errors.New("Got no response whatsoever :(")
	}

	return resp, nil
}

func (tc *TrackerClient) sendAnnounceRequest(u *url.URL) (*TrackerResponse, error) {
	if strings.Index(u.Scheme, "http") == 0 {
		return tc.sendHTTPAnnounceRequest(u.String())
	} else if strings.Index(u.Scheme, "udp") == 0 {
		return tc.sendUDPAnnounceRequest(u)
	}

	return nil, errors.New(fmt.Sprintf("Unsupported protofol for url=%s", u))
}

// this does not yet work and it's badly tested
func (tc *TrackerClient) sendHTTPAnnounceRequest(u string) (*TrackerResponse, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("HTTP[%d] while calling tracker server %s\n %s",
			resp.StatusCode, u, string(b)))
	}

	body, err := decoder.Decode(string(b))
	if err != nil {
		return nil, err
	}

	var response TrackerResponse
	utils.MapToStruct(body, &response)

	return &response, nil
}

func (tc *TrackerClient) sendUDPAnnounceRequest(u *url.URL) (*TrackerResponse, error) {
	err := tc.setUpUDPConnectionId(u)
	if err != nil {
		return nil, err
	}

	conn, err := utils.ConnectToUDPURL(u)
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

	resp := make([]byte, 200)
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

	if transactionId != randomTransactionId {
		return nil, errors.New(fmt.Sprintf("Received different transaction_id, sent %d and got %d", randomTransactionId, transactionId))
	}

	return &TrackerResponse{Interval: interval}, nil
}

func (tc *TrackerClient) writeAnnounceRequest(req *bytes.Buffer, transactionId int32) error {
	write := func(buff *bytes.Buffer, data any) error {
		if err := binary.Write(buff, binary.BigEndian, data); err != nil {
			return err
		}
		return nil
	}

	var (
		downloaded   int64  = 0
		left         int64  = int64(tc.torrentFile.Info.Length)
		uploaded     int64  = 0
		ip           uint32 = 0
		key          uint32 = rand.Uint32()
		numPeersWant int32  = -1
		port         uint16 = 6881
		extension    uint16 = 0
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
		numPeersWant,
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

func (tc *TrackerClient) setUpUDPConnectionId(u *url.URL) error {
	conn, err := utils.ConnectToUDPURL(u)
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
		return errors.New(fmt.Sprintf("Received different transaction_id, sent %d and got %d", randomTransactionId, transactionId))
	}

	if action == udpError {
		return errors.New("Received an error action from tracker server")
	}

	tc.connectionId = connectionId

	return nil
}

func (tc *TrackerClient) prepareTrackerUrl(u string) (*url.URL, error) {
	params := url.Values{}

	params.Add("info_hash", tc.infoHash)

	params.Add("peer_id", string(tc.peerId))
	params.Add("event", string(tc.status))

	ip, err := getCurrentIp()
	if err != nil {
		return nil, err
	}
	params.Add("ip", ip)

	trackerUrl, err := url.Parse(fmt.Sprintf("%s?%s", u, params.Encode()))
	if err != nil {
		return nil, err
	}

	return trackerUrl, nil
}

