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
	"strings"
)

type TrackerClient struct {
	serverUrls []string
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
	urls := []string{torrentFile.Announce}
	if len(torrentFile.AnnounceList) != 0 {
		for _, innerList := range torrentFile.AnnounceList {
			for _, elm := range innerList {
				urls = append(urls, elm)
			}
		}
	}

	return &TrackerClient{
		serverUrls: urls,
		info:       torrentFile.Info,
		peerId:     generateRandomPeerId(),
		downloaded: 0,
		left:       torrentFile.Info.Length,
		status:     started,
	}
}

// @TODO: support UDP too (https://www.bittorrent.org/beps/bep_0015.html)
func (trackerClient *TrackerClient) Start() (*TrackerResponse, error) {
	var resp *TrackerResponse = nil
	for _, u := range trackerClient.serverUrls {
		trackerUrl, err := trackerClient.prepareTrackerUrl(u)
		if err != nil {
			return nil, err
		}

		tmpResp, err := trackerClient.sendRequestToTracker(trackerUrl)
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

func (trackerClient *TrackerClient) sendRequestToTracker(u string) (*TrackerResponse, error) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	if strings.Index(parsedUrl.Scheme, "http") == 0 {
		return trackerClient.sendHttpRequestToTracker(u)
	}

	// @TODO: support udp

	return nil, nil
}

// this does not yet work and it's badly tested
func (trackerClient *TrackerClient) sendHttpRequestToTracker(u string) (*TrackerResponse, error) {
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

	fmt.Println(string(b))
	body, err := decoder.Decode(string(b))
	if err != nil {
		return nil, err
	}

	var response TrackerResponse
	utils.MapToStruct(body, &response)

	return &response, nil
}

func (trackerClient *TrackerClient) prepareTrackerUrl(u string) (string, error) {

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

	trackerUrl := fmt.Sprintf("%s?%s", u, params.Encode())
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
