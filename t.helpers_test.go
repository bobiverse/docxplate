package docxplate

import (
	"bytes"
	"errors"
	"io"
	"log"
	"testing"
)

// Invalid input tests as valid tests are performed mainly in `t_docx_test.go`

type brokenReadCloser struct {
	io.Reader
	shouldReadFail  bool
	shouldCloseFail bool
}

func (brc *brokenReadCloser) Read(p []byte) (n int, err error) {
	if brc.shouldReadFail {
		return 0, errors.New("broken read error")
	}
	if brc.Reader == nil {
		return 0, io.EOF
	}
	return brc.Reader.Read(p)
}

func (brc *brokenReadCloser) Close() error {
	if brc.shouldCloseFail {
		return errors.New("broken close error")
	}
	return nil
}

func TestReaderBytesInvalidCases(t *testing.T) {
	// disable log output for tests
	wr := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(wr)

	t.Run("nil input", func(t *testing.T) {
		var rdr io.ReadCloser = nil
		result := readerBytes(rdr)
		if result != nil {
			t.Fatalf("Expected nil result, got: %v", result)
		}
	})

	t.Run("broken reader", func(t *testing.T) {
		rdr := &brokenReadCloser{shouldReadFail: true}
		result := readerBytes(rdr)
		if result != nil {
			t.Fatalf("Expected nil result, got: %v", result)
		}
	})

	t.Run("broken closer", func(t *testing.T) {
		data := []byte("test data")
		rdr := &brokenReadCloser{Reader: bytes.NewReader(data), shouldCloseFail: true}
		result := readerBytes(rdr)
		if result != nil {
			t.Fatalf("Expected nil result, got: %v", result)
		}
	})
}

type invalidTestXMLStruct struct {
	UnsupportedField complex128
}

func TestStructToXMLBytesError(t *testing.T) {
	t.Run("invalid struct", func(t *testing.T) {
		invalidStruct := invalidTestXMLStruct{UnsupportedField: complex(1, 2)}
		result := structToXMLBytes(invalidStruct)
		if result != nil {
			t.Fatalf("Expected nil result, got: %v", result)
		}
	})
}
