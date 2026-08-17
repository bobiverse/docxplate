package docxplate_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"regexp"
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

// TestIssue51ReopenGeneratedDocAsTemplate - a docx produced by ExportDocx() must be
// re-openable as a template and, once an image placeholder is filled, must not end up
// with duplicate `[Content_Types].xml` entries (invalid OOXML that Word refuses to open).
func TestIssue51ReopenGeneratedDocAsTemplate(t *testing.T) {
	type withBody struct{ Body string }
	type withImages struct{ Images []*docxplate.Image }

	tdoc, err := docxplate.OpenTemplate("test-data/issue.51.docx")
	if err != nil {
		t.Fatalf("OpenTemplate: %s", err)
	}
	tdoc.Params(withBody{Body: "Example Body"})
	step1Path := "test-data/~test-issue.51.step1.docx"
	if err := tdoc.ExportDocx(step1Path); err != nil {
		t.Fatalf("step1 ExportDocx: %s", err)
	}

	// Re-open step1 output as a new template and fill the image placeholder
	tdoc2, err := docxplate.OpenTemplate(step1Path)
	if err != nil {
		t.Fatalf("re-open generated doc: %s", err)
	}
	tdoc2.Params(withImages{
		Images: []*docxplate.Image{
			{Path: "images/avatar-1.png", Width: 50, Height: 50},
		},
	})
	step2Path := "test-data/~test-issue.51.docx"
	if err := tdoc2.ExportDocx(step2Path); err != nil {
		t.Fatalf("step2 ExportDocx: %s", err)
	}

	contentTypesXML := readContentTypesXML(t, step2Path)

	matches := regexp.MustCompile(`Extension="png"`).FindAllString(contentTypesXML, -1)
	if len(matches) != 1 {
		t.Fatalf("[Content_Types].xml has %d Default Extension=\"png\" entries, want 1 (duplicate entries make Word reject the file): %s", len(matches), contentTypesXML)
	}
}

// TestIssue51MultipleImageExtensions - when a single Params() call embeds images of
// different, not-yet-registered extensions, each processImage() call must see the
// content types added by the previous call in the same run - otherwise only the last
// extension survives and the docx ends up with an image referencing a content type
// that was never registered (invalid OOXML that Word refuses to open).
func TestIssue51MultipleImageExtensions(t *testing.T) {
	tdoc, err := docxplate.OpenTemplate("test-data/issue.51.docx")
	if err != nil {
		t.Fatalf("OpenTemplate: %s", err)
	}
	tdoc.Params(struct{ Images []*docxplate.Image }{
		Images: []*docxplate.Image{
			{Path: "images/avatar-1.jpg", Width: 20, Height: 20},
			{Path: "images/avatar-1.gif", Width: 20, Height: 20},
		},
	})
	outPath := "test-data/~test-issue.51.multi-ext.docx"
	if err := tdoc.ExportDocx(outPath); err != nil {
		t.Fatalf("ExportDocx: %s", err)
	}

	contentTypesXML := readContentTypesXML(t, outPath)
	for _, ext := range []string{"jpg", "gif"} {
		want := fmt.Sprintf(`Extension="%s"`, ext)
		if !strings.Contains(contentTypesXML, want) {
			t.Fatalf("[Content_Types].xml is missing %s (lost when a later image overwrote it): %s", want, contentTypesXML)
		}
	}
}

// readContentTypesXML - open docxPath as a zip and return the contents of its
// [Content_Types].xml. Fails the test if the zip is invalid or the entry isn't found
// exactly once (Word rejects a docx missing or duplicating that entry).
func readContentTypesXML(t *testing.T, docxPath string) string {
	t.Helper()

	raw, err := os.ReadFile(docxPath) // #nosec G304 - test fixture path, not user input
	if err != nil {
		t.Fatalf("read %s: %s", docxPath, err)
	}

	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("%s is not a valid zip: %s", docxPath, err)
	}

	var found int
	var contentTypesXML string
	for _, f := range zr.File {
		if f.Name != "[Content_Types].xml" {
			continue
		}
		found++

		fr, err := f.Open()
		if err != nil {
			t.Fatalf("open [Content_Types].xml: %s", err)
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(fr); err != nil {
			t.Fatalf("read [Content_Types].xml: %s", err)
		}
		_ = fr.Close()
		contentTypesXML = buf.String()
	}
	if found != 1 {
		t.Fatalf("%s has %d [Content_Types].xml entries, want exactly 1", docxPath, found)
	}

	return contentTypesXML
}
