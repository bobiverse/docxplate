package docxplate

import (
	"bytes"
	"crypto/md5" // #nosec  G501 - allowed weak hash
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
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
		// fmt.Printf("error: %v\n", err)
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
func downloadFile(urlStr string) (tmpFile string, err error) {
	// Get file
	resp, err := http.Get(urlStr) // #nosec  G107 - allowed url variable here
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("download: remove: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
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
