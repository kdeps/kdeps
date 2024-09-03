package download

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/spf13/afero"
)

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 50))
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

func DownloadFile(fs afero.Fs, url string, filepath string) error {
	// Create the .tmp file using afero
	out, err := fs.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	// Perform the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create a WriteCounter to show download progress
	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	fmt.Println()

	// Rename the file from .tmp to the desired filepath
	err = fs.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}
