package main

import (
	"github.com/egregors/sucker/app/internal"
	"log"
	"os"
	"strings"
)

const (
	version      = "0.1"
	workersCount = 6
)

func main() {
	log.Printf("<<<< SUCKER ver: %s", version)

	links, err := internal.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("bad args: %v", err)
	}

	log.Printf("Downloading for: %s", strings.Join(links, "\n"))
	dl, err := internal.NewDownloader(links, workersCount)
	if err != nil {
		log.Printf("[FATAL] can't make downloader: %v", err)
		os.Exit(0)
	}
	dl.DownloadAll()
	log.Print("Done")
}
