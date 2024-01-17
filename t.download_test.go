package docxplate

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type errorTransport struct{}

func (et *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("transport error")
}

func TestDownloadFileInvalidCases(t *testing.T) {
	t.Run("invalid URL", func(t *testing.T) {
		tmpFpath, err := downloadFile("::invalid-url")
		if err == nil {
			t.Fatalf("Expected an error, but got nil")
		}
		os.Remove(tmpFpath)
	})

	t.Run("transport error", func(t *testing.T) {
		// Save original transport
		originalTransport := http.DefaultTransport

		// Set custom error transport
		http.DefaultTransport = &errorTransport{}

		// Restore original transport after the test
		defer func() {
			http.DefaultTransport = originalTransport
		}()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer server.Close()

		tmpFpath, err := downloadFile(server.URL)
		fmt.Println("remove tmp file", tmpFpath, server.URL)
		if err == nil {
			t.Fatalf("Expected an error, but got nil")
		}
		os.Remove(tmpFpath)
	})

	t.Run("non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tmpFpath, err := downloadFile(server.URL)
		if !errors.Is(err, http.ErrMissingFile) {
			t.Fatalf("Expected http.ErrMissingFile, but got: %v", err)
		}
		os.Remove(tmpFpath)
	})

	t.Run("server read error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "2")
			io.WriteString(w, "1") // #nosec G104
		}))
		defer server.Close()

		tmpFpath, err := downloadFile(server.URL)
		if err == nil {
			t.Fatalf("Expected an error, but got nil")
		}
		os.Remove(tmpFpath)
	})
	//
	//t.Run("create temp file error", func(t *testing.T) {
	//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//		io.WriteString(w, "test data")
	//	}))
	//	defer server.Close()
	//
	//	// Create a temporary directory with no write permissions
	//	tempDir, err := os.MkdirTemp("", "downloadFileTest")
	//	if err != nil {
	//		t.Fatalf("Failed to create temporary directory: %v", err)
	//	}
	//	defer os.RemoveAll(tempDir)
	//
	//	if err := os.Chmod(tempDir, 0555); err != nil {
	//		t.Fatalf("Failed to set permissions on temporary directory: %v", err)
	//	}
	//
	//	// Temporarily replace os.TempDir function
	//	originalTempDirFunc := os.TempDir
	//	os.TempDir = func() string { return tempDir }
	//	defer func() { os.TempDir = originalTempDirFunc }()
	//
	//	_, err = downloadFile(server.URL)
	//	if err == nil {
	//		t.Fatalf("Expected an error, but got nil")
	//	}
	//})
}
