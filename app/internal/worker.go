package internal

import (
	"context"
	"github.com/cheggaaa/pb/v3"
	"log"
	"sync"
)

type Worker struct {
	queue chan *File
}

func NewWorker(queue chan *File) *Worker {
	w := new(Worker)
	w.queue = queue
	return w
}

func (w *Worker) start(ctx context.Context, group *sync.WaitGroup, bar *pb.ProgressBar) {
	go func() {
		defer group.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case file, ok := <-w.queue:
				if !ok {
					return
				}
				err := file.download()
				if err != nil {
					log.Printf("[ERROR] can't download or save file %s: %v", file.downloadLink, err)
					w.retry(file)
					continue
				}
			}
			bar.Increment()
		}
	}()
}

func (w *Worker) retry(file *File) {
	if file.retryCount > file.retryLimit {
		log.Printf("[ERROR] max retry %d times for %s, skip", file.retryCount, file.fileName)
	}

	go func() {
		file.delete()
		w.queue <- file
	}()
}
