package docxplate_test

import (
	"testing"

	"github.com/bobiverse/docxplate"
)

func TestIssues(t *testing.T) {

	filenames := []string{
		"issue-31.docx",
	}

	for _, fname := range filenames {
		tdoc, _ := docxplate.OpenTemplate("test-data/" + fname)
		tdoc.Params(map[string]any{"ISSUE": 31})
		if err := tdoc.ExportDocx("test-data/~test-" + fname); err != nil {
			t.Fatalf("[%s] ExportDocx: %s", fname, err)
		}

		// success: just needs to be parsed without errors
	}
}
