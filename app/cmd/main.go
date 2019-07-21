package main

import (
	"github.com/egregors/sucker/app/internal"
	"log"
	"os"
	"strings"
)

const version = "0.1"

func main() {
	log.Printf("<<<< SUCKER ver: %s", version)

	links, err := internal.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("bad args: %v", err)
	}

	log.Printf("Downloading for: %s", strings.Join(links, ", "))
	// todo: pass urls into downloader interface
	//	dl := Downloader(links)
	//	err := dl.start()
}
