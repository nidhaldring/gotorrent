package trackerclient

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"gotorrent/decoder"
	"gotorrent/encoder"
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
  Interval int
  Peers []struct{
    Id string
    Ip string
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
func (trackerClient *TrackerClient) AnnounceRequest() ( *TrackerResponse, error ) {
	params := url.Values{}

  info, err := encoder.Encode(trackerClient.info)
  if err != nil {
    return nil, err
  }

  h := sha1.New()
  info_hash := string(h.Sum([]byte(info)))
	params.Add("info_hash", info_hash)

	params.Add("peer_id", trackerClient.peerId)
	params.Add("event", string(trackerClient.status))

	ip, err := getCurrentIp()
	if err != nil {
		return nil, err
	}
	params.Add("ip", ip)

  resp, err := http.Get(fmt.Sprintf("%s?%s", trackerClient.serverUrl, params.Encode()))
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()

  b, err := io.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }

  body, err := decoder.Decode(string(b))
  if err != nil {
    return nil, err
  }

	return body, nil
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
