package main

import (
	"context"
	"github.com/egregors/sucker/app/downloader"
	"github.com/egregors/sucker/app/internal"
	"log"
	"os"
)

func main() {
	// get all pages licks from CLI args
	pageLinks, err := internal.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("bad args: %v", err)
	}

	ctx, _ := context.WithCancel(context.Background())

	d, err := downloader.NewDownloader(ctx, pageLinks)
	if err != nil {
		panic(err)
	}
	d.Run()
}
