package docxplate_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bobiverse/docxplate"
	"github.com/logrusorgru/aurora"
)

func TestIssues(t *testing.T) {

	filenames := map[string]map[string]any{
		"issue.31.docx": {
			"ISSUE":   31,
			"STREET":  "Brīvības iela",
			"CITY":    "Rīga",
			"ZIPCODE": "LV-1001",
			"Letters": []string{"A", "B", "C"},
		},
		"issue.48.docx": {
			"ISSUE":                        48,
			"Customer Firstname":           "John",
			"Customer Surname":             "Wick",
			"Footer note with many spaces": "A man has to look his best when it's time to get married. Or buried.",
			"Letters":                      []string{"A", "B", "C"},
			"Empfänger Vorname":            "John2",
			"Empfänger Nachname":           "Wick2",
		},
	}

	for fname, params := range filenames {
		tdoc, _ := docxplate.OpenTemplate("test-data/" + fname)
		tdoc.Params(params)

		// placeholder leftovers
		if strings.Contains(tdoc.Plaintext(), "{{") {
			fmt.Printf("\n---\n%s\n---\n", aurora.Yellow(tdoc.Plaintext()))
			t.Fatalf("[%s] Placeholders: %s", fname, "Template still contains unfilled placeholders. Please specify values for them.")
		}

		if err := tdoc.ExportDocx("test-data/~test-" + fname); err != nil {
			t.Fatalf("[%s] ExportDocx: %s", fname, err)
		}

		// success: just needs to be parsed without errors
	}
}
