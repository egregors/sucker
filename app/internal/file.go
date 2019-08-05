package internal

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	retryCount     int
	downloadLink   string
	downloadToPath string
	fileName       string
}

func NewFile(link, path string) *File {
	f := new(File)
	f.downloadLink = link
	f.downloadToPath = path
	f.fileName = getFileNameFromLink(link)
	return f
}

func getFileNameFromLink(l string) string {
	n := strings.Split(l, "/")
	return n[len(n)-1]
}

func (f *File) isExist() bool {
	if _, err := os.Stat(f.getFilePath()); err != nil {
		return false
	}
	return true
}

func (f *File) download() error {
	if !f.isExist() {
		f.retryCount++
		resp, err := http.Get(f.downloadLink)
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

func (f *File) save(resp *http.Response) error {
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

func (f *File) getFilePath() string {
	return filepath.Join(f.downloadToPath, f.fileName)
}
