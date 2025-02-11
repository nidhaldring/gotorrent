package trackerclient

import (
	"context"
	"fmt"
	"gotorrent/decoder"
	"testing"
	"time"
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

	d, _ := time.ParseDuration("1m")
	ctx, _ := context.WithTimeout(context.Background(), d)

	ch := make(chan error)
	go client.Start(ctx, ch)

	select {
	case <-ctx.Done():
		fmt.Println("done after 1m")

	case v := <-ch:
		fmt.Printf("got error %s\n", v)
	}
}
