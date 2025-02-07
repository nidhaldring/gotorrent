package trackerclient

import (
	"gotorrent/decoder"
	"testing"
)

func TestAnnounceRequest(t *testing.T) {
	torrent, err := decoder.DecodeTorrentFile("../decoder/files/test.torrent")
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewTrackerClient(*torrent)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Announce()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", resp)
}
