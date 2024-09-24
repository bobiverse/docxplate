package docxplate

import (
	"context"
	"crypto/md5" // #nosec  G501 - allowed weak hash here
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
)

// DownloadClient to use instead of default http.Client
type DownloadClient struct {
}

// Downloader ..
type Downloader interface {
	DownloadFile(ctx context.Context, urlStr string) (tmpFile string, err error)
}

// DefaultDownloader to use as default client
var DefaultDownloader Downloader = &DownloadClient{}

// DownloadFile (satisfy interface) Download url file
func (DownloadClient) DownloadFile(_ context.Context, urlStr string) (tmpFile string, err error) {
	resp, err := http.Get(urlStr) // #nosec  G107 - allowed url variable here
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("download: remove: %s", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", http.ErrMissingFile
	}
	// Create file
	tmpFile = fmt.Sprintf("%x%s", md5.Sum([]byte(urlStr)), path.Ext(urlStr)) // #nosec  G401 - allowed weak hash here
	out, err := os.Create(tmpFile)                                           // #nosec  G304 - allowed filename variable here
	if err != nil {
		return
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("download: close: %s", err)
		}
	}()

	// Write body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return
	}
	return tmpFile, nil
}
