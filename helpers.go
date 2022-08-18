package docxplate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
)

func readerBytes(rdr io.ReadCloser) []byte {
	buf := new(bytes.Buffer)

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
func structToXMLBytes(v interface{}) []byte {
	// buf, err := xml.MarshalIndent(v, "", "  ")
	buf, err := xml.Marshal(v)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil
	}

	// This is fixing `xmlns` attribute representation after marshal
	buf = bytes.ReplaceAll(buf, []byte(` xmlns:_xmlns="xmlns"`), []byte(""))
	buf = bytes.ReplaceAll(buf, []byte(`_xmlns:`), []byte("xmlns:"))

	// xml decoder doesnt support <w:t so using placeholder with "w-" (<w-t)
	// Or you have solution?
	buf = bytes.ReplaceAll(buf, []byte("<w-"), []byte("<w:"))
	buf = bytes.ReplaceAll(buf, []byte("</w-"), []byte("</w:"))

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
