package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/vbauerster/mpb/v8"
)

func TestDownloadConcurrentSeenAccess(t *testing.T) {
	t.Parallel()

	body := []byte("video-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	tempDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	seen := make(map[string]struct{})
	seenMu := &sync.Mutex{}
	var seenCnt atomic.Int32
	progress := mpb.New(mpb.WithOutput(io.Discard))
	mainBar := progress.AddBar(10)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			download(server.URL+"/file-"+strconv.Itoa(i)+".mp4", progress, mainBar, seen, seenMu, &seenCnt, false)
		}(i)
	}

	wg.Wait()
	progress.Wait()

	if got := len(seen); got != 10 {
		t.Fatalf("seen entries = %d, want 10", got)
	}

	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, "downloads", "file-"+strconv.Itoa(i)+".mp4")
		if _, err := os.Stat(filePath); err != nil {
			t.Fatalf("expected downloaded file %q: %v", filePath, err)
		}
	}
}
