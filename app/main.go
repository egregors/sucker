package main

import (
	"context"
	"log"
	"os"

	"github.com/egregors/sucker/app/downloader"
	"github.com/egregors/sucker/app/internal"
)

// FIXME: Soooo..... locks like we can't make requests through Cloudflare "check" page.
// 	Instead of page with content we got stub page with JS, and can't jump next.
// 	For now we got HTTP/1.1 401 Unauthorized by Cloudflare validation.
//  Looks like this is the end :\
func main() {
	// get all pages licks from CLI args
	pageLinks, err := internal.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("bad args: %v", err)
	}

	// FIXME: wtf?
	ctx, _ := context.WithCancel(context.Background())

	d, err := downloader.NewDownloader(ctx, pageLinks)
	if err != nil {
		panic(err)
	}
	d.Run()
}
