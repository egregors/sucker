package main

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"golang.org/x/net/html"
)

func main() {
	// get HTML from stdin
	rawPage, err := getInputFromPipe()
	if err != nil {
		log.Fatalf("can't get page: %v", err)
	}

	// parse HTML page into searchable tree
	doc, err := html.Parse(strings.NewReader(rawPage))
	if err != nil {
		log.Fatalf("can't parse page: %v", err)
	}

	// search for a links with valid exts
	links := make(map[string]bool)
	findLinks(doc, links)

	// make a Bar
	wg := &sync.WaitGroup{}
	progress := mpb.New(
		mpb.WithWidth(60),
		mpb.WithRefreshRate(180*time.Millisecond),
		mpb.WithWaitGroup(wg),
	)
	mainBar := progress.Add(int64(len(links)),
		mpb.NewBarFiller(mpb.BarStyle().Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟")),
		mpb.PrependDecorators(
			decor.Name("Total:", decor.WCSyncSpaceR),
			decor.CountersNoUnit("%d / %d", decor.WCSyncSpaceR),
			decor.Elapsed(decor.ET_STYLE_GO, decor.WCSyncWidthR),
		),
		mpb.AppendDecorators(
			decor.OnComplete(
				decor.Name("downloading...", decor.WCSyncSpaceR), " done!",
			),
		),
	)

	// make chan from links list
	linksChan := make(chan string)
	go func() {
		for k := range links {
			linksChan <- k
		}
		close(linksChan)
	}()

	// spawn workers
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ls <-chan string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case l, ok := <-linksChan:
					if !ok {
						return
					}
					download(l, progress, mainBar)
				}

			}
		}(ctx, wg, linksChan)
	}

	// wait until end
	progress.Wait()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func download(link string, p *mpb.Progress, mBar *mpb.Bar) {
	fileNameParts := strings.Split(link, "/")
	fileName := fileNameParts[len(fileNameParts)-1]
	downloadToPath, _ := os.Getwd()
	filePath := filepath.Join(downloadToPath, "sucker_downloads", fileName)
	if fileExists(filePath) {
		mBar.Increment()
		return
	}

	resp, err := http.Get(link)
	if err != nil {
		log.Printf("[ERROR] can't request %s : %s", link, err)
	}

	bar, proxyReader := func(resp *http.Response, link string) (*mpb.Bar, io.ReadCloser) {
		b := p.AddBar(
			resp.ContentLength,
			mpb.BarFillerClearOnComplete(),
			mpb.BarOptOn(mpb.BarRemoveOnComplete(), func() bool { return true }), // del bar
			mpb.PrependDecorators(
				decor.Name(link, decor.WCSyncSpaceR),
				decor.CountersKibiByte("% .2f / % .2f"),
			),
			mpb.AppendDecorators(
				decor.EwmaETA(decor.ET_STYLE_GO, 90),
				decor.Name(" ] "),
				decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
			),
		)
		pR := b.ProxyReader(resp.Body)
		return b, pR
	}(resp, link)

	_ = os.MkdirAll(filepath.Join(downloadToPath, "sucker_downloads"), os.ModePerm)
	file, _ := os.Create(filePath)

	_, err = io.Copy(file, proxyReader)
	if err != nil {
		bar.Abort(true)
	}

	mBar.Increment()
	_ = proxyReader.Close()
	_ = file.Close()
}

func findLinks(n *html.Node, acc map[string]bool) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" && isValidExt(a.Val) {
				// TODO: hide the domain
				acc["_a"+a.Val] = true
			}
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		findLinks(ch, acc)
	}
}

func isValidExt(l string) bool {
	exts := []string{"webm", "mp4"}
	for _, ext := range exts {
		if strings.Contains(l, ext) {
			return true
		}
	}
	return false
}

func getInputFromPipe() (string, error) {
	nBytes, nChunks := int64(0), int64(0)
	r := bufio.NewReader(os.Stdin)
	buf := make([]byte, 0, 128*1024)
	var res string

	for {
		n, err := r.Read(buf[:cap(buf)])
		buf = buf[:n]

		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			return "", err
		}

		nChunks++
		nBytes += int64(len(buf))

		res += string(buf)

		if err != nil && err != io.EOF {
			return "", nil
		}
	}

	return res, nil
}
