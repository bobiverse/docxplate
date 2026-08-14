package docxplate_test

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
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

	docXML := documentXMLFromBytes(t, buf)

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

// vmergeVariant - build a template from test-data/vmerge.docx
// with word/document.xml modified by `edit`
func vmergeVariant(t *testing.T, edit func(string) string) *docxplate.Template {
	t.Helper()

	raw, err := os.ReadFile("test-data/vmerge.docx")
	if err != nil {
		t.Fatalf("ReadFile: %s", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("zip.NewReader: %s", err)
	}

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
			b = []byte(edit(string(b)))
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

	tdoc, err := docxplate.OpenTemplateWithBytes(out.Bytes())
	if err != nil {
		t.Fatalf("OpenTemplateWithBytes: %s", err)
	}
	return tdoc
}

// vmergeCell - `{{Name :vmerge}}` cell markup up to the placeholder
func vmergeCell(t *testing.T, doc string) (start int, cell string) {
	t.Helper()

	i := strings.Index(doc, "{{Name :vmerge}}")
	if i < 0 {
		t.Fatal("{{Name :vmerge}} not found in template")
	}
	start = strings.LastIndex(doc[:i], "<w:tc>")
	if start < 0 {
		t.Fatal("<w:tc> of :vmerge placeholder not found")
	}
	return start, doc[start:i]
}

// documentXMLFromBytes - word/document.xml of a rendered docx
func documentXMLFromBytes(t *testing.T, docxBytes []byte) string {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("zip.NewReader: %s", err)
	}
	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %s", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %s", f.Name, err)
		}
		return string(b)
	}
	t.Fatal("word/document.xml not found in rendered docx")
	return ""
}

// renderedTable - first table of the rendered document
func renderedTable(t *testing.T, tdoc *docxplate.Template) string {
	t.Helper()

	buf, err := tdoc.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %s", err)
	}
	docXML := documentXMLFromBytes(t, buf)
	return strings.SplitN(strings.Split(docXML, "<w:tbl>")[1], "</w:tbl>", 2)[0]
}

// vmergeUser - fresh test data for each test, so none of them
// can end up sharing (and accidentally mutating) one User
func vmergeUser() *User {
	return &User{
		Name: "Alice",
		Friends: []*User{
			{Name: "Bob", Age: 28},
			{Name: "Cecilia", Age: 29},
			{Name: "Den", Age: 30},
		},
	}
}

// TestVMergeDoesNotLeak - the same param is reused for every node,
// so `:vmerge` on one placeholder must not stick to a plain
// `{{Name}}` somewhere else in the document
func TestVMergeDoesNotLeak(t *testing.T) {
	tdoc := vmergeVariant(t, func(doc string) string {
		return strings.Replace(doc, "</w:body>",
			"<w:p><w:r><w:t>PLAIN={{Name}}</w:t></w:r></w:p></w:body>", 1)
	})
	tdoc.Params(vmergeUser())

	plaintext := tdoc.Plaintext()
	if !strings.Contains(plaintext, "PLAIN=Alice") {
		t.Errorf("plain {{Name}} must be replaced, got:\n%s", plaintext)
	}
}

// TestVMergeCellWithoutTcPr - `w:tcPr` is optional in a cell,
// the merge must still happen
func TestVMergeCellWithoutTcPr(t *testing.T) {
	tdoc := vmergeVariant(t, func(doc string) string {
		start, cell := vmergeCell(t, doc)
		a := strings.Index(cell, "<w:tcPr>")
		b := strings.Index(cell, "</w:tcPr>")
		if a < 0 || b < 0 {
			t.Fatal("w:tcPr not found in :vmerge cell")
		}
		return doc[:start] + cell[:a] + cell[b+len("</w:tcPr>"):] + doc[start+len(cell):]
	})
	tdoc.Params(vmergeUser())

	tbl := renderedTable(t, tdoc)
	if n := strings.Count(tbl, `<w:vMerge w:val="restart"`); n < 1 {
		t.Errorf("cell without w:tcPr must still get vMerge restart, got %d", n)
	}
	if n := strings.Count(tbl, "<w:t>Alice</w:t>"); n != 1 {
		t.Errorf("merged value count: expected 1, got %d", n)
	}
}

// TestVMergeCellAlreadyMerged - a cell merged in the template already
// has `w:vMerge`, and CT_TcPr allows only one of them
func TestVMergeCellAlreadyMerged(t *testing.T) {
	tdoc := vmergeVariant(t, func(doc string) string {
		start, cell := vmergeCell(t, doc)
		merged := strings.Replace(cell, "</w:tcPr>",
			`<w:vMerge w:val="restart"/></w:tcPr>`, 1)
		return doc[:start] + merged + doc[start+len(cell):]
	})
	tdoc.Params(vmergeUser())

	tbl := renderedTable(t, tdoc)
	for _, tcPr := range regexp.MustCompile(`<w:tcPr>.*?</w:tcPr>`).FindAllString(tbl, -1) {
		if n := strings.Count(tcPr, "<w:vMerge"); n > 1 {
			t.Errorf("w:tcPr holds %d w:vMerge, only one allowed", n)
		}
	}
}
