package docxplate

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
)

const mainDocFname = "word/document.xml"

// filename allowed to check/modify for params
// which have keyword in it
var modFileNamesLike = []string{
	"word/footer",
	mainDocFname,
	"word/header",
}
var t *Template

// Template ..
type Template struct {
	path string
	// file *os.File
	// zipw *zip.Writer // zip writer
	zipr *zip.Reader // zip reader

	// save all zip files here so we can build it again
	files map[string]*zip.File
	// content type document file
	documentContentTypes *zip.File
	// document relations
	documentRels map[string]*zip.File
	// only added files (converted to []byte) save here
	added map[string][]byte
	// only modified files (converted to []byte) save here
	modified map[string][]byte

	// hold all parsed params:values here
	params ParamList
}

// OpenTemplate - docpath local file
func OpenTemplate(docpath string) (*Template, error) {
	var err error
	docBytes, err := os.ReadFile(docpath) // #nosec G304 - allowed filename variable here
	if err != nil {
		return nil, err
	}

	t, err := OpenTemplateWithBytes(docBytes)
	if err != nil {
		return nil, err
	}

	t.path = docpath
	return t, nil

}

// OpenTemplateWithBytes - template from bytes
// Credits to @dreamph for implementing this function
func OpenTemplateWithBytes(docBytes []byte) (*Template, error) {
	var err error

	// Init doc template
	t = &Template{
		files:        map[string]*zip.File{},
		documentRels: map[string]*zip.File{},
		added:        map[string][]byte{},
		modified:     map[string][]byte{},
	}

	// Unzip
	if t.zipr, err = zip.NewReader(bytes.NewReader(docBytes), int64(len(docBytes))); err != nil {
		return nil, err
	}

	// Get main document
	for _, f := range t.zipr.File {
		t.files[f.Name] = f

		if f.Name == "[Content_Types].xml" {
			t.documentContentTypes = f
			continue
		}

		if path.Ext(f.Name) == ".rels" {
			t.documentRels[f.Name] = f
			continue
		}

	}

	if t.files[mainDocFname] == nil {
		return nil, fmt.Errorf("mandatory [ %s ] not found", mainDocFname)
	}

	return t, nil
}

// OpenTemplateWithURL .. docpath is remote url
func OpenTemplateWithURL(docurl string) (tpl *Template, err error) {
	docpath, err := DefaultDownloader.DownloadFile(context.Background(), docurl)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := os.Remove(docpath); err != nil {
			log.Printf("open url: remove: %s", err)
		}
	}()

	tpl, err = OpenTemplate(docpath)
	if err != nil {
		return nil, err
	}
	return
}

// Expand some placeholders to enable row replacer replace them
// Note: Currently only struct type support image replacement
// Users: []User{ User{Name:AAA}, User{Name:BBB} }
// {{Users.Name}} -->
//      {{Users.1.Name}}
//      {{Users.2.Name}}

// Params  - replace template placeholders with params
// "Hello {{ Name }}!"" --> "Hello World!""
func (t *Template) Params(v any) {
	// t.params = collectParams("", v)
	switch val := v.(type) {
	case map[string]any:
		t.params = mapToParams(val)
	case string:
		t.params = JSONToParams([]byte(val))
	case []byte:
		t.params = JSONToParams(val)
	default:
		if reflect.ValueOf(v).Kind() == reflect.Struct {
			t.params = StructToParams(val)
		} else {
			// any other type try to convert
			t.params = AnyToParams(val)
		}
	}

	for _, f := range t.files {
		for _, keyword := range modFileNamesLike {
			if !strings.Contains(f.Name, keyword) {
				continue
			}

			xnode := t.fileToXMLStruct(f.Name)

			// Enhance some markup (removed when building XML in the end)
			// so easier to find some element
			t.enhanceMarkup(xnode)

			// While formating docx sometimes same style node is split to
			// multiple same style nodes and different content
			// Merge them so placeholders are in the same node
			t.fixBrokenPlaceholders(xnode)

			// Complex placeholders with more depth needs to be expanded
			// for correct replace
			t.expandPlaceholders(xnode)

			// Replace params
			t.replaceSingleParams(xnode, false)

			// Collect placeholders with trigger but unset in `t.params`
			// Placeholders with trigger `:empty` must be triggered
			// otherwise they are left
			t.triggerMissingParams(xnode)

			// After all done with placeholders, modify contents
			// - new lines to docx new lines
			t.enhanceContent(xnode)

			// Save []bytes
			t.modified[f.Name] = structToXMLBytes(xnode)
		}
	}
}

// Bytes - create docx archive but return only bytes of it
// do not save it anywhere
func (t *Template) Bytes() ([]byte, error) {
	var err error

	bufw := new(bytes.Buffer)
	zipw := zip.NewWriter(bufw)

	// Loop existing files to build docx archive again
	for _, f := range t.files {
		// Read contents of single file inside zip
		var fr io.ReadCloser
		if fr, err = f.Open(); err != nil {
			log.Printf("Error reading [ %s ] from archive", f.Name)
			continue
		}
		fbuf := new(bytes.Buffer)
		if _, err := fbuf.ReadFrom(fr); err != nil {
			log.Printf("[%s] read file: %s", f.Name, err)
		}

		if err := fr.Close(); err != nil {
			log.Printf("[%s] file close: %s", f.Name, err)
		}

		// Write contents as single file inside zip
		var fw io.Writer
		if fw, err = zipw.Create(f.Name); err != nil {
			log.Printf("Error writing [ %s ] to archive", f.Name)
			continue
		}

		// Move/Write struct-saved file to docx archive file back
		if buf, isModified := t.modified[f.Name]; isModified {
			if _, err := fw.Write(buf); err != nil {
				log.Printf("[%s] write error: %s", f.Name, err)
			}
			continue
		}

		if _, err := fw.Write(fbuf.Bytes()); err != nil {
			log.Printf("[%s] write error: %s", f.Name, err)
		}
	}
	// Loop new added files to build docx archive
	for fName, buf := range t.added {
		var fw io.Writer
		if fw, err = zipw.Create(fName); err != nil {
			log.Printf("Error writing [ %s ] to archive", fName)
			continue
		}
		_, _ = fw.Write(buf)
	}

	zipErr := zipw.Close()
	return bufw.Bytes(), zipErr
}

// ExportDocx - save new/modified docx based on template
func (t *Template) ExportDocx(path string) error {

	buf, err := t.Bytes()
	if err != nil {
		return err
	}

	err = os.WriteFile(path, buf, 0640) // #nosec G306

	return err
}

// Placeholders - get list of used params placeholders in template
// If you already replaced params with values then you will not get all placeholders.
// Or use it after replace and see how many placeholders left.
func (t *Template) Placeholders() []string {
	var arr []string

	plaintext := t.Plaintext()

	re := regexp.MustCompile(ParamPattern)
	arr = re.FindAllString(plaintext, -1)

	return arr
}

// Plaintext - return as plaintext
func (t *Template) Plaintext() string {

	if len(t.params) == 0 {
		// if params not set yet we init process with empty params
		// and mark content as changed so we can return plaintext with placeholders
		// not replaced yet
		t.Params(nil)
	}

	plaintext := ""

	// for fpath, f := range t.files {
	// 	for _, keyword := range modFileNamesLike {
	// 		if strings.Contains(f.Name, keyword) {
	// 			log.Printf("%-30s %v", fpath, f.Name)
	// 			fmt.Printf("=============== %v", t.modified[f.Name])
	// 		}
	// 	}
	// }

	// header and footer must be printed in plaintext
	for _, f := range t.modified {
		xnode := t.bytesToXMLStruct(f)

		xnode.Walk(func(n *xmlNode) {
			if n.Tag() != "w-p" {
				return
			}

			s := string(n.AllContents())
			plaintext += s
			if s != "" {
				plaintext += "\n"
			}
		})

	}

	return plaintext
}
