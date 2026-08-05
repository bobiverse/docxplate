package docxplate_test

import (
	"archive/zip"
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/bobiverse/docxplate"
)

// TestVMerge - {{Name :vmerge}} should vertically merge cell
// over multiplied table rows: first row cell gets vMerge "restart"
// with replaced value, all the next rows - "continue" with empty cell.
// test-data/vmerge.docx holds "Template" table (rendered) and
// "Expect" table (expected result) to compare with
func TestVMerge(t *testing.T) {

	user := &User{
		Name: "Alice",
		Friends: []*User{
			{Name: "Bob", Age: 28},
			{Name: "Cecilia", Age: 29},
			{Name: "Den", Age: 30},
			{Name: "Edgar", Age: 31},
		},
	}

	tdoc, err := docxplate.OpenTemplate("test-data/vmerge.docx")
	if err != nil {
		t.Fatalf("OpenTemplate: %s", err)
	}
	tdoc.Params(user)

	if err := tdoc.ExportDocx("test-data/~test-vmerge.docx"); err != nil {
		t.Fatalf("ExportDocx: %s", err)
	}

	buf, err := tdoc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %s", err)
	}

	// read rendered word/document.xml
	zr, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		t.Fatalf("zip.NewReader: %s", err)
	}
	var docXML string
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		docXML = string(b)
	}
	if docXML == "" {
		t.Fatal("word/document.xml not found in rendered docx")
	}

	// first table is rendered "Template", second one is untouched "Expect"
	tables := strings.Split(docXML, "<w:tbl>")
	if len(tables) < 3 {
		t.Fatalf("expected 2 tables in rendered document, got %d", len(tables)-1)
	}
	rendered := strings.SplitN(tables[1], "</w:tbl>", 2)[0]
	expect := strings.SplitN(tables[2], "</w:tbl>", 2)[0]

	// rendered table must have same vMerge structure as Expect table
	// (attr prefix varies: parsed nodes marshal as main:val, new ones as w:val)
	reRestart := regexp.MustCompile(`<w:vMerge [^>]*:val="restart"`)
	reContinue := regexp.MustCompile(`<w:vMerge [^>]*:val="continue"`)

	rRestart, eRestart := len(reRestart.FindAllString(rendered, -1)), len(reRestart.FindAllString(expect, -1))
	if rRestart != eRestart {
		t.Errorf("vMerge restart count: rendered %d, expect %d", rRestart, eRestart)
	}
	rContinue, eContinue := len(reContinue.FindAllString(rendered, -1)), len(reContinue.FindAllString(expect, -1))
	if rContinue != eContinue {
		t.Errorf("vMerge continue count: rendered %d, expect %d", rContinue, eContinue)
	}

	// merged value rendered only once (in the restart cell).
	// NB: value is replaced in its own run, so "is friend to " and "Alice"
	// stay in separate w-t nodes (unlike hand-joined Expect table)
	if n := strings.Count(rendered, "<w:t>Alice</w:t>"); n != 1 {
		t.Errorf("'Alice' value count in rendered table: expected 1, got %d", n)
	}

	// per data row checks: first row restart with value, others continue and empty
	rows := strings.Split(rendered, "<w:tr ")
	cases := []struct {
		friend string
		vmerge string
	}{
		{"Bob", "restart"},
		{"Cecilia", "continue"},
		{"Den", "continue"},
		{"Edgar", "continue"},
	}
	for _, c := range cases {
		var row string
		for _, r := range rows {
			if strings.Contains(r, "<w:t>"+c.friend+"</w:t>") {
				row = r
				break
			}
		}
		if row == "" {
			t.Errorf("row with friend %s not found", c.friend)
			continue
		}
		reVMerge := regexp.MustCompile(`<w:vMerge [^>]*:val="` + c.vmerge + `"`)
		if !reVMerge.MatchString(row) {
			t.Errorf("row[%s]: expected vMerge %s", c.friend, c.vmerge)
		}
		if c.vmerge == "restart" && (!strings.Contains(row, "is friend to") || !strings.Contains(row, "<w:t>Alice</w:t>")) {
			t.Errorf("row[%s]: restart cell must hold the replaced value", c.friend)
		}
		if c.vmerge == "continue" && (strings.Contains(row, "is friend to") || strings.Contains(row, "<w:t>Alice</w:t>")) {
			t.Errorf("row[%s]: continuation cell must be empty", c.friend)
		}
	}

	// placeholders must be replaced/rendered
	rdoc, err := docxplate.OpenTemplateWithBytes(buf)
	if err != nil {
		t.Fatalf("OpenTemplateWithBytes: %s", err)
	}
	plaintext := rdoc.Plaintext()
	for _, must := range []string{"Bob", "Cecilia", "Den", "Edgar", "is friend to", "Alice"} {
		if !strings.Contains(plaintext, must) {
			t.Errorf("rendered plaintext must contain %q", must)
		}
	}
	if strings.Contains(plaintext, ":vmerge") {
		t.Errorf("rendered plaintext must not contain :vmerge mark")
	}
}
