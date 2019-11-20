package internal

import (
	"context"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"log"
	"sync"
)

type Worker struct {
	queue chan *File
	wg    *sync.WaitGroup
	bar   *pb.ProgressBar

	errorCh chan<- error
	mux     *sync.Mutex
}

func NewWorker(queue chan *File, wg *sync.WaitGroup, bar *pb.ProgressBar) *Worker {
	w := &Worker{
		queue,
		wg,
		bar,
		nil,
		new(sync.Mutex),
	}
	return w
}

func (w *Worker) start(ctx context.Context, errorCh chan<- error) {
	defer w.wg.Done()
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
				errorCh <- errors.New(
					fmt.Sprintf("[ERROR] can't download or save file %s retry: %d: %v",
						file.downloadLink,
						file.retryCount,
						err),
				)
				w.retry(file)
				continue
			}
		}
		w.barIncrement()
	}
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

func (w *Worker) barIncrement() {
	w.mux.Lock()
	w.bar.Increment()
	w.mux.Unlock()
}
