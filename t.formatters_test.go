package docxplate_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bobiverse/docxplate"
)

func TestFormatters(t *testing.T) {

	filenames := []string{
		"formatters.docx",
	}

	for _, fname := range filenames {
		tdoc, _ := docxplate.OpenTemplate("test-data/" + fname)
		tdoc.Params(map[string]any{"Name": "Lorem ipsum"})
		if err := tdoc.ExportDocx("test-data/~test-" + fname); err != nil {
			t.Fatalf("[%s] ExportDocx: %s", fname, err)
		}
		plaintext := tdoc.Plaintext()

		// Default
		if !strings.Contains(plaintext, "This is Lorem ipsum.") {
			t.Fatalf("[%s] ExportDocx: %s", fname, "no formatter failed")
		}

		// Lowercase
		if !strings.Contains(plaintext, "Lowercase: `lorem ipsum`") {
			fmt.Println(tdoc.Plaintext())
			t.Fatalf("[%s] ExportDocx: %s", fname, "Lowercase failed")
		}

		// Capitalize
		if !strings.Contains(plaintext, "Capitalize: `Lorem ipsum`") {
			fmt.Println(tdoc.Plaintext())
			t.Fatalf("[%s] ExportDocx: %s", fname, "Capitalize failed")
		}

		// Title: `Lorem Ipsum`
		if !strings.Contains(plaintext, "Title: `Lorem Ipsum`") {
			fmt.Println(tdoc.Plaintext())
			t.Fatalf("[%s] ExportDocx: %s", fname, "Title: `Lorem Ipsum` failed")
		}

		// success: just needs to be parsed without errors
	}
}
