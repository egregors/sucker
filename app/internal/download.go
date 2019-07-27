package internal

import (
	"context"
	"github.com/cheggaaa/pb/v3"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// NewDownload create a downloader and setup the file chan from the links
func NewDownloader(links []string, workersCount int) (*Downloader, error) {
	// todo: looks like i should move all this args into Opts struct, cause it
	//  too many for func
	d := &Downloader{}
	// todo: crete folder for download
	err := d.setDownloadDir()
	if err != nil {
		log.Printf("[ERROR] can't create dir: %v", err)
		return nil, err
	}
	d.setQueueFromLinks(links)
	d.setWorkersCount(workersCount)
	d.setProgressBar()
	return d, nil
}

type file struct {
	url, path  string
	retryCount int
}

func (f *file) isExist() bool {
	if _, err := os.Stat(f.getFilePath()); err != nil {
		return false
	}
	return true
}

func (f *file) download() error {
	if !f.isExist() {
		f.retryCount++
		resp, err := http.Get(f.url)
		if err != nil {
			return err
		}

		err = f.save(resp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *file) save(resp *http.Response) error {
	filePath := f.getFilePath()
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}

func (f *file) getFilePath() string {
	fileName := strings.Split(f.url, "/")
	return filepath.Join(f.path, fileName[len(fileName)-1])
}

type Downloader struct {
	queue                              chan file
	queueLen, retryLimit, workersCount int
	downloadDir                        string
	bar                                *pb.ProgressBar
}

// DownloadAll is spawn N workers and downloading all file from file chan
func (d *Downloader) DownloadAll() {
	d.bar.Start()
	wg := &sync.WaitGroup{}
	ctx := context.Background()
	d.spawnWorkers(ctx, wg)
	wg.Wait()
	d.bar.Finish()
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

func (d *Downloader) spawnWorkers(ctx context.Context, wg *sync.WaitGroup) {
	for i := 0; i < d.workersCount; i++ {
		wg.Add(1)
		go d.startWorker(ctx, wg)
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
	// todo: pass exts from outside
	//  there i can got number of files links and use it for
	//  UI representation (kind of progress bar I guess)
	filesLinks := NewHtmlParser(links, nil).GetLinks()
	d.queueLen = len(filesLinks)
	d.queue = make(chan file)
	go func() {
		for _, l := range filesLinks {
			d.queue <- file{l, d.downloadDir, 0}
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
			err := f.download()
			if err != nil {
				log.Printf("[ERROR] can't download or save file %s: %v", f.url, err)
				if f.retryCount < d.retryLimit {
					f.retryCount++
					go func() {
						d.queue <- f
					}()
				}
				continue
			}
			d.bar.Increment()
		}
	}
}

func (d *Downloader) setProgressBar() {
	d.bar = pb.New(d.queueLen)
}
