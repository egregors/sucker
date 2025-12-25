package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"golang.org/x/net/html"
	"golang.org/x/term"
)

func main() {
	// get HTML from stdin
	rawPage, err := getInputFromPipe()
	if err != nil {
		log.Fatalf("can't get page: %v", err)
	}

	// parse input to get links
	links, err := ParseInput(rawPage)
	if err != nil {
		log.Fatalf("can't parse input: %v", err)
	}

	// load history is possible
	var seen map[string]struct{}
	var seenMutex sync.Mutex
	if fileExists("history.gob") {
		seen, err = loadHistory("history.gob")
		if err != nil {
			log.Fatalf("can't load history: %v", err)
		}
	} else {
		seen = make(map[string]struct{})
	}

	// make a Bar
	wg := &sync.WaitGroup{}
	progress := mpb.New(
		mpb.WithWidth(60),
		mpb.WithRefreshRate(180*time.Millisecond),
		mpb.WithWaitGroup(wg),
	)

	// Use a mutex to protect the main bar total
	var mainBarMutex sync.Mutex
	totalItems := int64(len(links))
	mainBar := progress.Add(totalItems,
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

	// make chan from links list - buffered to allow adding new items
	linksChan := make(chan string, 100)

	go func() {
		for k := range links {
			linksChan <- k
		}
	}()

	// spawn workers
	ctx, cancel := context.WithCancel(context.Background())
	var chanMutex sync.Mutex // Protect channel closure
	chanClosed := false

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutting down gracefully...")
		cancel()
		// Wait a bit for keyboard listener to exit, then close channel
		time.Sleep(100 * time.Millisecond)
		chanMutex.Lock()
		if !chanClosed {
			close(linksChan)
			chanClosed = true
		}
		chanMutex.Unlock()
	}()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go worker(ctx, wg, linksChan, progress, mainBar, seen, &seenMutex)
	}

	// Start keyboard listener for adding items from clipboard
	log.Println("Press 'p' to add downloads from clipboard, Ctrl+C to exit")
	go keyboardListener(ctx, linksChan, &mainBarMutex, mainBar, &totalItems, seen, &seenMutex, &chanMutex, &chanClosed)

	// wait until end (workers exit when context is cancelled or channel is closed)
	progress.Wait()

	// save history
	seenMutex.Lock()
	defer seenMutex.Unlock()
	if seen != nil {
		err = saveHistory("history.gob", seen)
		if err != nil {
			log.Fatalf("can't save history: %v", err)
		}
	}
}

// worker processes download jobs from the queue
func worker(ctx context.Context, wg *sync.WaitGroup, ls <-chan string, p *mpb.Progress, mBar *mpb.Bar, seen map[string]struct{}, seenMutex *sync.Mutex) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case l, ok := <-ls:
			if !ok {
				return
			}
			download(l, p, mBar, seen, seenMutex)
		}
	}
}

// ParseInput parses the input text (HTML or plain text with links) and returns a map of valid links
func ParseInput(text string) (map[string]bool, error) {
	links := make(map[string]bool)

	// Try to parse as HTML
	doc, err := html.Parse(strings.NewReader(text))
	if err == nil {
		findLinks(doc, links)
		if len(links) > 0 {
			return links, nil
		}
	}

	// If no HTML links found, try to parse as plain text URLs
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if it looks like a URL and has valid extension
		if (strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")) && isValidExt(line) {
			links["_a"+line] = true
		}
	}

	return links, nil
}

// ReadClipboard reads the current clipboard content
func ReadClipboard() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		// Try xclip first, fallback to wl-paste for Wayland
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		} else if _, err := exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command("wl-paste")
		} else {
			return "", fmt.Errorf("no clipboard tool found (install xclip or wl-paste)")
		}
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}

	return string(output), nil
}

// keyboardListener listens for keyboard input and handles adding items from clipboard
func keyboardListener(ctx context.Context, linksChan chan<- string, mainBarMutex *sync.Mutex, mainBar *mpb.Bar, totalItems *int64, seen map[string]struct{}, seenMutex *sync.Mutex, chanMutex *sync.Mutex, chanClosed *bool) {
	// Open /dev/tty to read keyboard input even when stdin is piped
	tty, err := os.Open("/dev/tty")
	if err != nil {
		log.Printf("Warning: failed to open /dev/tty for keyboard input: %v", err)
		return
	}
	defer tty.Close()

	// Set terminal to raw mode to read single keypresses
	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		log.Printf("Warning: failed to enable keyboard listener: %v", err)
		return
	}
	defer term.Restore(int(tty.Fd()), oldState)

	buf := make([]byte, 1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set a read timeout to allow checking ctx.Done()
			tty.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := tty.Read(buf)
			if err != nil {
				// Check for timeout specifically
				if os.IsTimeout(err) {
					continue
				}
				// Log other errors but continue
				log.Printf("Warning: keyboard read error: %v", err)
				continue
			}

			if n > 0 && buf[0] == 'p' {
				handleClipboardAdd(linksChan, mainBarMutex, mainBar, totalItems, seen, seenMutex, chanMutex, chanClosed)
			}
		}
	}
}

// handleClipboardAdd reads clipboard, parses it, and adds new items to the queue
func handleClipboardAdd(linksChan chan<- string, mainBarMutex *sync.Mutex, mainBar *mpb.Bar, totalItems *int64, seen map[string]struct{}, seenMutex *sync.Mutex, chanMutex *sync.Mutex, chanClosed *bool) {
	clipboardContent, err := ReadClipboard()
	if err != nil {
		log.Printf("Failed to read clipboard: %v", err)
		return
	}

	if strings.TrimSpace(clipboardContent) == "" {
		log.Println("Clipboard is empty")
		return
	}

	newLinks, err := ParseInput(clipboardContent)
	if err != nil {
		log.Printf("Failed to parse clipboard: %v", err)
		return
	}

	if len(newLinks) == 0 {
		log.Println("No valid links found in clipboard")
		return
	}

	// Filter out duplicates and add new items
	addedCount := 0
	for link := range newLinks {
		// Check if already in seen or being processed
		url := strings.TrimPrefix(link, "_a")
		fileNameParts := strings.Split(url, "/")
		fileName := fileNameParts[len(fileNameParts)-1]
		downloadToPath, _ := os.Getwd()
		filePath := filepath.Join(downloadToPath, "sucker_downloads", fileName)

		seenMutex.Lock()
		_, alreadySeen := seen[filePath]
		seenMutex.Unlock()

		if !alreadySeen && !fileExists(filePath) {
			// Check if channel is still open before sending
			chanMutex.Lock()
			if !*chanClosed {
				linksChan <- link
				addedCount++
				// Update main bar total using tracked totalItems
				mainBarMutex.Lock()
				*totalItems++
				mainBar.SetTotal(*totalItems, false)
				mainBarMutex.Unlock()
			}
			chanMutex.Unlock()
		}
	}

	if addedCount > 0 {
		log.Printf("Added %d new items from clipboard", addedCount)
	} else {
		log.Println("No new items to add (all duplicates)")
	}
}

func loadHistory(path string) (map[string]struct{}, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read history: %w", err)
	}

	buf := bytes.NewBuffer(f)
	decoder := gob.NewDecoder(buf)

	seen := make(map[string]struct{})
	err = decoder.Decode(&seen)
	if err != nil {
		return nil, fmt.Errorf("can't decode history: %w", err)
	}

	return seen, nil
}

func saveHistory(path string, seen map[string]struct{}) error {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	err := encoder.Encode(seen)
	if err != nil {
		return fmt.Errorf("can't encode items: %w", err)
	}
	err = os.WriteFile(path, buf.Bytes(), 0o600)
	if err != nil {
		return fmt.Errorf("can't save history: %w", err)
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func download(link string, p *mpb.Progress, mBar *mpb.Bar, seen map[string]struct{}, seenMutex *sync.Mutex) {
	fileNameParts := strings.Split(link, "/")
	fileName := fileNameParts[len(fileNameParts)-1]
	downloadToPath, _ := os.Getwd()
	filePath := filepath.Join(downloadToPath, "sucker_downloads", fileName)
	if fileExists(filePath) {
		mBar.Increment()
		return
	}

	seenMutex.Lock()
	_, alreadySeen := seen[filePath]
	seenMutex.Unlock()

	if alreadySeen {
		mBar.Increment()
		return
	}

	resp, err := http.Get(link)
	if err != nil {
		log.Printf("[ERROR] can't request %s : %s", link, err)
		mBar.Increment()
		return
	}
	defer resp.Body.Close()

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
	seenMutex.Lock()
	seen[filePath] = struct{}{}
	seenMutex.Unlock()
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
	exts := []string{".webm", ".mp4"}
	// Parse URL to get path component
	if idx := strings.Index(l, "?"); idx != -1 {
		l = l[:idx] // Remove query string
	}
	if idx := strings.Index(l, "#"); idx != -1 {
		l = l[:idx] // Remove fragment
	}

	// Check if URL ends with valid extension
	lowerURL := strings.ToLower(l)
	for _, ext := range exts {
		if strings.HasSuffix(lowerURL, ext) {
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
