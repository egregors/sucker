package internal

import (
	"context"
	"errors"
	"github.com/cheggaaa/pb/v3"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// NewDownload create a downloader and setup the file chan from the links
func NewDownloader(links []string, workersCount int) (*Downloader, error) {
	// todo: looks like i should move all this args into Opts struct, cause it
	//  too many for func
	d := new(Downloader)
	d.retryLimit = 3
	d.setQueueFromLinks(links)

	if d.queueLen == 0 {
		return nil, errors.New("[WARN] nothing to download [0 links was find]")
	}

	d.setWorkersCount(workersCount)
	d.setProgressBar()
	err := d.setDownloadDir()
	if err != nil {
		log.Printf("[ERROR] can't create dir: %v", err)
		return nil, err
	}

	return d, nil
}

type Downloader struct {
	queue                              chan *File
	queueLen, retryLimit, workersCount int
	downloadDir                        string
	workers                            []*Worker
	bar                                *pb.ProgressBar
	errors                             []error
}

// DownloadAll is spawn N workers and downloading all file from file chan
func (d *Downloader) DownloadAll() {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	errorCh := make(chan error)

	d.bar.Start()
	d.createWorkers(wg)
	d.spawnWorkers(ctx, errorCh)
	d.catchErrors(ctx, errorCh)
	wg.Wait()

	d.bar.Finish()
	cancel()
	d.showErrors()
}

func (d *Downloader) makeDir(path string) (string, error) {
	if path == "" {
		baseDir, _ := os.Getwd()
		// todo: replace it by generic name or name from page for downloading
		path = filepath.Join(baseDir, "sucker_downloads/")
		log.Printf("current path: %v", path)
	}
	err := os.MkdirAll(path, os.ModePerm)
	return path, err
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

func (d *Downloader) createWorkers(wg *sync.WaitGroup) {
	for i := 0; i < d.workersCount; i++ {
		w := NewWorker(d.queue, wg, d.bar)
		d.workers = append(d.workers, w)
	}
}

func (d *Downloader) spawnWorkers(ctx context.Context, errorCh chan<- error) {
	for _, w := range d.workers {
		w.wg.Add(1)
		go w.start(ctx, errorCh)
	}
}

func (d *Downloader) setDownloadDir() error {
	// todo: replace path from Opts
	path, err := d.makeDir("")
	if err != nil {
		log.Printf("[ERROR] can't create folder: %v", err)
		return err
	}
	d.downloadDir = path
	return err
}

func (d *Downloader) setQueueFromLinks(links []string) {
	filesLinks := NewHtmlParser(links, nil).GetLinks()
	d.queueLen = len(filesLinks)
	d.queue = make(chan *File)
	go func() {
		for _, l := range filesLinks {
			d.queue <- NewFile(l, d.downloadDir, d.retryLimit)
		}
		close(d.queue)
	}()
}

func (d *Downloader) setProgressBar() {
	d.bar = pb.New(d.queueLen)
}

func (d *Downloader) showErrors() {
	if len(d.errors) > 0 {
		log.Printf("[WARN] errors during downloading: %d", len(d.errors))
		for _, err := range d.errors {
			log.Println(err)
		}
	}
}

func (d *Downloader) catchErrors(ctx context.Context, errorsCh chan error) {
	go func() {
		defer close(errorsCh)
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-errorsCh:
				if !ok {
					return
				}
				d.errors = append(d.errors, e)
			}
		}
	}()
}
