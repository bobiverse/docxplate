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

	return buf
}

// interface{} to []string
func toStringSlice(v interface{}) []string {
	var sarr []string

	//TODO: add more slice types
	switch arr := v.(type) {
	case []string:
		sarr = arr

	case []float64:
		for _, val := range arr {
			sarr = append(sarr, fmt.Sprintf("%v", val))
		}

	case []int:
		for _, val := range arr {
			sarr = append(sarr, fmt.Sprintf("%v", val))
		}

	}

	return sarr
}

// interface{} to []string
func toMap(v interface{}) map[string]string {
	m := map[string]string{}

	fn := func(key, val interface{}) {
		k := fmt.Sprintf("%v", key)
		m[k] = fmt.Sprintf("%v", val)
	}

	//TODO: add more slice types
	switch arr := v.(type) {

	// Slices
	case []string:
		for i, val := range arr {
			fn(i+1, val) //use i+1 to avoid starting number 0 as it not useful for user
		}
	case []float64:
		for i, val := range arr {
			fn(i+1, val)
		}
	case []int:
		for i, val := range arr {
			fn(i+1, val)
		}

		// Maps
	case map[string]string:
		m = arr
	case map[string]int:
		for key, val := range arr {
			fn(key, val)
		}
	case map[string]float64:
		for key, val := range arr {
			fn(key, val)
		}

	}

	return m
}
