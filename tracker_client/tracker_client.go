package trackerclient

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"gotorrent/decoder"
	"gotorrent/encoder"
	"gotorrent/utils"
	"io"
	"math/rand"
	"net/http"
	"net/url"
)

type TrackerClient struct {
	serverUrl  string
	info       decoder.TorrentInfo
	peerId     string
	downloaded int
	left       int
	status     trackerClientStatus
}

type trackerClientStatus string

const (
	started   trackerClientStatus = "started"
	completed                     = "completed"
	stopped                       = "stopped"
)

type TrackerResponse struct {
  // @TODO: FailureReason this is optional
	FailureReason string
	Interval      int
	Peers         []struct {
		Id   string
		Ip   string
		Port string
	}
}

func NewTrackerClient(torrentFile decoder.TorrentFile) *TrackerClient {
	return &TrackerClient{
		serverUrl:  torrentFile.Announce,
		info:       torrentFile.Info,
		peerId:     generateRandomPeerId(),
		downloaded: 0,
		left:       torrentFile.Info.Length,
		status:     started,
	}
}

// @TODO: support UDP too (https://www.bittorrent.org/beps/bep_0015.html)
func (trackerClient *TrackerClient) AnnounceRequest() (*TrackerResponse, error) {
	trackerUrl, err := trackerClient.prepareTrackerUrl()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(trackerUrl)
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
			resp.StatusCode, trackerUrl, string(b)))
	}

	fmt.Println(string(b))
	body, err := decoder.Decode(string(b))
	if err != nil {
		return nil, err
	}

	var response TrackerResponse
	utils.MapToStruct(body, &response)

	return &response, nil
}

func (trackerClient *TrackerClient) prepareTrackerUrl() (string, error) {

	params := url.Values{}

	info, err := encoder.Encode(trackerClient.info)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	io.WriteString(h, info)
	info_hash := string(h.Sum(nil))
	params.Add("info_hash", string(info_hash))

	params.Add("peer_id", trackerClient.peerId)
	params.Add("event", string(trackerClient.status))

	ip, err := getCurrentIp()
	if err != nil {
		return "", err
	}
	params.Add("ip", ip)

	trackerUrl := fmt.Sprintf("%s?%s", trackerClient.serverUrl, params.Encode())
	return trackerUrl, nil
}

func generateRandomPeerId() string {
	// @TODO: this is bad; change it
	randomPeerId := ""
	for i := 0; i < 20; i++ {
		randomPeerId += string(rune('a' + rand.Intn('z'-'a')))
	}
	return randomPeerId
}

type getIpApiResponse struct {
	Status string
	Query  string
}

func getCurrentIp() (string, error) {
	resp, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return "", errors.New("Cannot get current ip!")
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var body getIpApiResponse
	err = json.Unmarshal(b, &body)
	if err != nil {
		return "", errors.New("Cannot get current ip!")
	}

	if body.Status != "success" {
		return "", errors.New("Cannot get current ip!")
	}
	return body.Query, nil
}
