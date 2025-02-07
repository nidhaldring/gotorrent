package trackerclient

import (
	"encoding/json"
	"errors"
	"gotorrent/decoder"
	"io"
	"math/rand"
	"net/http"
)

func generateRandomPeerId() []byte {
	var randomPeerId = make([]byte, 20)
	for i := 0; i < 20; i++ {
		randomPeerId[i] = byte('a' + rand.Intn('z'-'a'))
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

func getUrls(torrentFile decoder.TorrentFile) []string {
	urls := []string{torrentFile.Announce}
	if len(torrentFile.AnnounceList) != 0 {
		for _, innerList := range torrentFile.AnnounceList {
			for _, elm := range innerList {
				urls = append(urls, elm)
			}
		}
	}
	return urls
}
