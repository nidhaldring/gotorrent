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

	t.Logf("%+v", *torrent)

	client := NewTrackerClient(*torrent)
	r, err := client.AnnounceRequest()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", *r)
}
