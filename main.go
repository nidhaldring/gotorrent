package main

import (
	"flag"
	"fmt"
	"gotorrent/decoder"
)

func main() {

	trntFile := flag.String("decode", "", "specify torrent file to decode")
	flag.Parse()

	if *trntFile != "" {
		decoded, err := decoder.DecodeTorrentFile(*trntFile)
		if err != nil {
			fmt.Printf("Failed to decode %s due to [Error]: %s\n", *trntFile, err)
		}

		fmt.Println("---------- Decoded file ----------")
		fmt.Printf("%+v\n", decoded)
		fmt.Println("---------- End decoded file ----------")
	}
}
