package docxplate

import (
	"context"
	"crypto/md5" // #nosec  G501 - allowed weak hash here
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	tmpFile = fmt.Sprintf("%x%s", md5.Sum([]byte(urlStr)), urlExt(urlStr)) // #nosec  G401 - allowed weak hash here
	out, err := os.Create(tmpFile)                                         // #nosec  G304 - allowed filename variable here
	if err != nil {
		return "", err
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("download: close: %s", err)
		}
	}()

	// Write body to file
	if _, err = io.Copy(out, resp.Body); err != nil {
		if rmErr := os.Remove(tmpFile); rmErr != nil {
			log.Printf("download: remove: %s", rmErr)
		}
		return "", err
	}
	return tmpFile, nil
}

// File extension of an url path. Query and fragment are dropped, so a
// pre-signed link (../avatar.png?X-Amz-Signature=..) still gives ".png"
// Called only after http.Get(urlStr) succeeded, so urlStr already parses.
func urlExt(urlStr string) string {
	u, _ := url.Parse(urlStr)

	return path.Ext(u.Path)
}
