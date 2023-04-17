package docxplate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
)

func readerBytes(rdr io.ReadCloser) []byte {
	buf := new(bytes.Buffer)

	if rdr == nil {
		log.Printf("can't read bytes from empty reader")
		return nil

	}

	if _, err := buf.ReadFrom(rdr); err != nil {
		log.Printf("can't read bytes: %s", err)
		return nil
	}

	if err := rdr.Close(); err != nil {
		log.Printf("can't close reader: %s", err)
		return nil
	}

	return buf.Bytes()
}

// Encode struct to xml code string
func structToXMLBytes(v any) []byte {
	// buf, err := xml.MarshalIndent(v, "", "  ")
	buf, err := xml.Marshal(v)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil
	}

	// This is fixing `xmlns` attribute representation after marshal
	buf = bytes.ReplaceAll(buf, []byte(` xmlns:_xmlns="xmlns"`), []byte(""))
	buf = bytes.ReplaceAll(buf, []byte(`_xmlns:`), []byte("xmlns:"))
	buf = bytes.ReplaceAll(buf, []byte(` xmlns:r="r"`), []byte(""))
	buf = bytes.ReplaceAll(buf, []byte(` xmlns:o="o"`), []byte(""))

	// xml decoder doesnt support <w:t so using placeholder with "w-" (<w-t)
	// Or you have solution?
	buf = bytes.ReplaceAll(buf, []byte("<w-"), []byte("<w:"))
	buf = bytes.ReplaceAll(buf, []byte("</w-"), []byte("</w:"))
	buf = bytes.ReplaceAll(buf, []byte("<v-"), []byte("<v:"))
	buf = bytes.ReplaceAll(buf, []byte("</v-"), []byte("</v:"))

	// buf = bytes.Replace(buf, []byte("w-item"), []byte("w-p"), -1)

	return buf
}

// Is slice contains item
func inSlice(a string, slice []string) bool {
	for _, b := range slice {
		if a == b {
			return true
		}
	}
	return false
}

// Download url file
// TODO: check for mime-type? if allow to download image, then only white-listed types
func downloadFile(urlStr string) (tmpFile string, err error) {
	// validate url first
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// download
	resp, err := http.Get(parsedURL.String()) // #nosec G107 - expected to download here
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // #nosec G307

	// validate
	if resp.StatusCode != http.StatusOK {
		return "", http.ErrMissingFile
	}

	// create temporary file
	tmpFile = fmt.Sprintf("remotefile-*%s", path.Ext(urlStr))
	out, err := os.CreateTemp("", tmpFile) // #nosec G304
	if err != nil {
		return
	}
	tmpFile = out.Name()

	// write body to temp file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write response body to file: %s", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer to file: %s", err)
	}

	return tmpFile, nil
}
