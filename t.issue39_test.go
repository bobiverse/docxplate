package docxplate_test

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bobiverse/docxplate"
)

// TestIssue39ClearPlaceholderSharedRun - an empty `:clear:placeholder` (or
// `:remove:placeholder`) trigger must only blank its own placeholder text, even
// when Word merged it into the same XML run as a sibling placeholder. Clearing
// the whole run must not wipe the sibling's already-rendered value.
// https://github.com/bobiverse/docxplate/issues/39
//
// NB: the issue's own example uses `first_name`/`second_name` (underscored
// keys), but those never match docxplate's placeholder regex at all (a
// separate, pre-existing limitation - `_` is excluded from key chars in
// `reParamExtract`, Param.go) - so they'd stay as unprocessed literal text,
// not reproduce the "value went missing" symptom. Using non-underscored keys
// here reproduces the actual reported bug.
func TestIssue39ClearPlaceholderSharedRun(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{
			name: "clear",
			text: `User: {{FirstName}} {{SecondName :empty:clear:placeholder}}`,
		},
		{
			name: "remove",
			text: `User: {{FirstName}} {{SecondName :empty:remove:placeholder}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docxBytes := issue39TemplateDocx(t, c.text)

			tdoc, err := docxplate.OpenTemplateWithBytes(docxBytes)
			if err != nil {
				t.Fatalf("OpenTemplateWithBytes: %s", err)
			}
			tdoc.Params(map[string]any{"FirstName": "john"})

			out, err := tdoc.Bytes()
			if err != nil {
				t.Fatalf("Bytes: %s", err)
			}

			rdoc, err := docxplate.OpenTemplateWithBytes(out)
			if err != nil {
				t.Fatalf("re-open rendered docx: %s", err)
			}
			plaintext := rdoc.Plaintext()

			if !strings.Contains(plaintext, "john") {
				t.Fatalf("FirstName value must survive, got: %q", plaintext)
			}
			if strings.Contains(plaintext, "SecondName") {
				t.Fatalf("SecondName placeholder must be gone, got: %q", plaintext)
			}
		})
	}
}

// issue39TemplateDocx - test-data/depth.docx with word/document.xml's body
// replaced by a single paragraph holding `text` as one run, so any placeholders
// in it share one XML text node (mirrors how Word merges adjacent placeholders
// with no formatting boundary between them)
func issue39TemplateDocx(t *testing.T, text string) []byte {
	t.Helper()

	raw, err := os.ReadFile("test-data/depth.docx")
	if err != nil {
		t.Fatalf("ReadFile: %s", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("zip.NewReader: %s", err)
	}

	paragraph := `<w:p><w:r><w:t>` + text + `</w:t></w:r></w:p>`

	out := new(bytes.Buffer)
	zw := zip.NewWriter(out)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %s", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %s", f.Name, err)
		}
		if f.Name == "word/document.xml" {
			doc := string(b)
			bodyStart := strings.Index(doc, "<w:body>") + len("<w:body>")
			sectPrStart := strings.Index(doc, "<w:sectPr")
			if bodyStart < len("<w:body>") || sectPrStart < 0 {
				t.Fatalf("unexpected document.xml shape in test-data/depth.docx")
			}
			doc = doc[:bodyStart] + paragraph + doc[sectPrStart:]
			b = []byte(doc)
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			t.Fatalf("create %s: %s", f.Name, err)
		}
		if _, err := w.Write(b); err != nil {
			t.Fatalf("write %s: %s", f.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %s", err)
	}

	return out.Bytes()
}
