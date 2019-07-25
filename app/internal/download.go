package internal

import (
	"context"
	"log"
	"sync"
)

// NewDownload create a downloader and setup the file chan from the links
func NewDownloader(links []string, n int) Downloader {
	d := Downloader{}
	d.setQueueFromLinks(links)
	d.setWorkersCount(n)
	return d
}

type file struct {
	url        string
	retryCount int
}

type Downloader struct {
	queue        chan file
	workersCount int
}

// DownloadAll is spawn N workers and downloading all file from file chan
func (d *Downloader) DownloadAll() {
	wg := &sync.WaitGroup{}
	ctx := context.Background()
	d.spawnWorkers(ctx, wg)
	wg.Wait()
}

func (d *Downloader) spawnWorkers(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < d.workersCount; i++ {
		wg.Add(1)
		go d.startWorker(ctx, wg)
	}
}

func (d *Downloader) setQueueFromLinks(links []string) {
	// todo: pass exts from outside
	filesLinks := NewHtmlParser(links, nil).GetLinks()

	d.queue = make(chan file)
	go func() {
		for _, l := range filesLinks {
			d.queue <- file{l, 0}
		}
		close(d.queue)
	}()
}

func (d *Downloader) setWorkersCount(n int) {
	d.workersCount = n
	if d.workersCount == 0 {
		d.setWorkersCountDefault()
	}
}

func (d *Downloader) setWorkersCountDefault() {
	// todo: move it into config
	d.workersCount = 5
}

func (d *Downloader) startWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-d.queue:
			if !ok {
				return
			}
			// download file
			log.Printf("[DEBUG] downloading file %s", f.url)
		}
	}
}
