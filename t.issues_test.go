package docxplate_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// TestIssue53FooterImage - an image placeholder in a footer must get its
// relationship written to that footer's own rels part. Word resolves r:id
// per part, so a relationship living only in document.xml.rels leaves the
// footer image blank
func TestIssue53FooterImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	originalDownloader := docxplate.DefaultDownloader
	docxplate.DefaultDownloader = &docxplate.DownloadClient{}
	t.Cleanup(func() { docxplate.DefaultDownloader = originalDownloader })

	tdoc, err := docxplate.OpenTemplate("test-data/header-footer.docx")
	if err != nil {
		t.Fatalf("OpenTemplate: %s", err)
	}

	tdoc.Params(struct {
		Name *docxplate.Image
	}{
		Name: &docxplate.Image{
			URL:    server.URL + "/avatar.png?X-Amz-Signature=abc&X-Amz-Algorithm=AWS4",
			Width:  50,
			Height: 50,
		},
	})

	docxBytes, err := tdoc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %s", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("open generated docx: %s", err)
	}

	footerXML := readZipFile(t, zipReader, "word/footer1.xml")
	if !strings.Contains(footerXML, "v:imagedata") {
		t.Fatalf("footer does not contain an image")
	}
	relationshipID := regexp.MustCompile(`r:id="([^"]+)"`).FindStringSubmatch(footerXML)
	if len(relationshipID) != 2 {
		t.Fatalf("footer image relationship ID is missing: %s", footerXML)
	}

	footerRels := readZipFile(t, zipReader, "word/_rels/footer1.xml.rels")
	if !strings.Contains(footerRels, `Id="`+relationshipID[1]+`"`) || !strings.Contains(footerRels, "relationships/image") {
		t.Fatalf("footer image relationship is missing: %s", footerRels)
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
	for _, f := range zr.File {
		if f.Name == "[Content_Types].xml" {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("%s has %d [Content_Types].xml entries, want exactly 1", docxPath, found)
	}

	return readZipFile(t, zr, "[Content_Types].xml")
}

// readZipFile - return the contents of a single entry of an open zip.
// Fails the test if the entry is missing or unreadable
func readZipFile(t *testing.T, zipReader *zip.Reader, filename string) string {
	t.Helper()

	for _, file := range zipReader.File {
		if file.Name != filename {
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %s", filename, err)
		}
		defer func() {
			if err := fileReader.Close(); err != nil {
				t.Errorf("close %s: %s", filename, err)
			}
		}()

		fileBytes := new(bytes.Buffer)
		if _, err := fileBytes.ReadFrom(fileReader); err != nil {
			t.Fatalf("read %s: %s", filename, err)
		}
		return fileBytes.String()
	}

	t.Fatalf("%s not found", filename)
	return ""
}
