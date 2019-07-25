package internal

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// NewDownload create a downloader and setup the file chan from the links
func NewDownloader(links []string, workersCount int) (*Downloader, error) {
	// todo: looks like i should move all this args into Opts struct, cause it
	//  too many for func
	d := &Downloader{}
	// todo: crete folder for download
	err := d.makeDir("")
	if err != nil {
		log.Printf("[ERROR] can't create folder: %v", err)
		return nil, err
	}
	d.setQueueFromLinks(links)
	d.setWorkersCount(workersCount)
	return d, nil
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

func (d *Downloader) makeDir(path string) error {
	if path == "" {
		baseDir, _ := os.Getwd()
		// todo: replace it by generic name or name from page for downloading
		path = filepath.Join(baseDir, "sucker_downloads/")
		log.Printf("current path: %v", path)
	}
	return os.MkdirAll(path, os.ModePerm)
}

func (d *Downloader) spawnWorkers(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < d.workersCount; i++ {
		wg.Add(1)
		go d.startWorker(ctx, wg)
	}
}

func (d *Downloader) setQueueFromLinks(links []string) {
	// todo: pass exts from outside
	//  there i can got number of files links and use it for
	//  UI representation (kind of progress bar I guess)
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
