package docxplate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

func readerBytes(rdr io.ReadCloser) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(rdr)
	rdr.Close()
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

	// Fix xmlns representation after marshal
	buf = bytes.Replace(buf, []byte(` xmlns:_xmlns="xmlns"`), []byte(""), -1)
	buf = bytes.Replace(buf, []byte(`_xmlns:`), []byte("xmlns:"), -1)

	// xml decoder doesnt support <w:t so using placeholder with "w-" (<w-t)
	// Or you have solution?
	buf = bytes.Replace(buf, []byte("<w-"), []byte("<w:"), -1)
	buf = bytes.Replace(buf, []byte("</w-"), []byte("</w:"), -1)

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
