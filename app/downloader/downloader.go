package downloader

import (
	"context"
	"errors"
	"github.com/egregors/sucker/app/internal"
	log "github.com/go-pkgz/lgr"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"net/url"
	"sync"
)

type Path string

func NewDownloader(ctx context.Context, rawPageURLs []string) (d *Downloader, err error) {
	// get file URLs from list of page URLs
	fileURLs, err := parsePageURLs(rawPageURLs)

	if err != nil {
		return nil, err
	}

	d = new(Downloader)

	d.fileURLs = fileURLs

	// todo: extract it. this is knda config
	d.spawnLimit = 6
	d.retryLimit = 3

	d.q = make(map[string]int)
	d.ctx = ctx

	d.wg = &sync.WaitGroup{}
	d.progress = mpb.New(
		mpb.WithWidth(60),
		mpb.WithRefreshRate(180*time.Millisecond),
		mpb.WithWaitGroup(d.wg),
	)

	return
}

type Downloader struct {
	fileURLs     []*url.URL
	q            map[string]int
	downloadPath Path

	// context
	ctx                    context.Context
	spawnLimit, retryLimit int

	// progress bar
	progress *mpb.Progress
	mainBar  *mpb.Bar
	wg       *sync.WaitGroup
}

func (d *Downloader) Run() {
	// total bar
	d.mainBar = d.makeMainBar()

	// make chan for URLs
	fileURLs := make(chan *url.URL)
	go func() {
		for _, URL := range d.fileURLs {
			fileURLs <- URL
		}
		close(fileURLs)
	}()

	// spawn workers
	d.wg.Add(d.spawnLimit)
	for i := 0; i < d.spawnLimit; i++ {
		go d.spawn(fileURLs)
	}

	// wait for all
	d.progress.Wait()
}

func (d *Downloader) makeMainBar() *mpb.Bar {
	return d.progress.AddBar(int64(len(d.fileURLs)), mpb.BarStyle("╢▌▌░╟"), // bar for proxy reader
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
}

func (d *Downloader) makeDownloadBarAndProxyReader(resp *http.Response, URL *url.URL) (*mpb.Bar, io.ReadCloser) {
	b := d.progress.AddBar(resp.ContentLength, mpb.BarStyle("[=>-|"), // bar for proxy reader
		mpb.BarOptOn(mpb.BarRemoveOnComplete(), func() bool { return true }), // del bar
		mpb.PrependDecorators(
			decor.Name(getFileNameFromURL(URL.String()), decor.WCSyncSpaceR),
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
}

func (d *Downloader) spawn(fileURLs <-chan *url.URL) {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		case URL, ok := <-fileURLs:
			if !ok {
				return
			}

			d.download(URL)
		}
	}
}

func (d *Downloader) download(URL *url.URL) {

	downloadToPath, _ := os.Getwd()
	fileName := getFileNameFromURL(URL.String())

	if fileExists(filepath.Join(downloadToPath, "sucker_downloads", fileName)) {
		return
	}

	resp, err := http.Get(URL.String())

	if err != nil {
		d.mainBar.Increment()
		return
	}

	bar, proxyReader := d.makeDownloadBarAndProxyReader(resp, URL)

	// TODO: refactor it
	os.MkdirAll(filepath.Join(downloadToPath, "sucker_downloads"), os.ModePerm)
	file, _ := os.Create(filepath.Join(downloadToPath, "sucker_downloads", fileName))

	_, err = io.Copy(file, proxyReader)
	if err != nil {
		// todo: try to make retry here
		bar.Abort(true)
	}

	d.mainBar.Increment()
	proxyReader.Close()
	file.Close()
}

func parseRawURLs(rawURLs []string) (urls []*url.URL, err error) {
	for _, rURL := range rawURLs {
		url_, err := url.Parse(rURL)
		if err != nil {
			log.Printf("[ERROR] can't parse page URL: %v, skip", err)
			continue
		}
		urls = append(urls, url_)
	}

	if len(urls) == 0 {
		return nil, errors.New("can't find any URLs")
	}

	return
}

func parsePageURLs(pageURLs []string) (fileURLs []*url.URL, err error) {
	p := internal.NewHtmlParser(pageURLs, nil)
	URLs := p.GetLinks()
	if len(URLs) == 0 {
		return nil, errors.New("URLs list is empty")
	}

	fileURLs, err = parseRawURLs(URLs)
	if err != nil {
		return nil, err
	}

	return
}

func getFileNameFromURL(l string) string {
	n := strings.Split(l, "/")
	return strings.Trim(n[len(n)-1], " ")
}
