package trackerclient

import (
	"encoding/json"
	"errors"
	"gotorrent/decoder"
	"io"
	"math/rand"
	"net/http"
	"net/url"
)

type TrackerClientStatus string

const (
	started   TrackerClientStatus = "started"
	completed                     = "completed"
	stopped                       = "stopped"
)

type TrackerClient struct {
	serverUrl  string
	info       decoder.TorrentInfo
	peerId     string
	downloaded int
	left       int
	status     TrackerClientStatus
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

func (trackerClient *TrackerClient) AnnounceRequest() error {
	params := url.Values{}
	params.Add("info_hash", "")
	params.Add("peer_id", trackerClient.peerId)
	params.Add("status", string(trackerClient.status))

	ip, err := getCurrentIp()
	if err != nil {
		return err
	}
	params.Add("ip", ip)

	http.Get(trackerClient.serverUrl)

	return nil
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
