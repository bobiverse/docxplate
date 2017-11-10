package docxplate

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/fatih/color"
)

// Template ..
type Template struct {
	path string
	zipr *zip.ReadCloser // zip reader

	// save all zip files here so we can build it again
	files map[string]*zip.File

	// only modified files (converted to []byte) save here
	modified map[string][]byte

	// hold all parsed params:values here
	params map[string]interface{}
}

// OpenTemplate ..
func OpenTemplate(docpath string) (*Template, error) {
	var err error

	t := &Template{
		path:     docpath,
		modified: map[string][]byte{},
	}

	// Unzip
	if t.zipr, err = zip.OpenReader(t.path); err != nil {
		return nil, err
	}

	// Get main document
	t.files = map[string]*zip.File{}
	for _, f := range t.zipr.File {
		t.files[f.Name] = f
	}
	if t.MainDocument() == nil {
		return nil, fmt.Errorf("mandatory [ word/document.xml ] not found")
	}

	return t, nil
}

// MainDocument ..
func (t *Template) MainDocument() *zip.File {
	fxml := t.files["word/document.xml"]
	return fxml
}

// Convert given file (from template.Files) to struct of xml nodes
func (t *Template) fileToXMLStruct(fname string) *xmlNode {
	f, ok := t.files[fname]
	if !ok {
		return nil
	}

	fr, _ := f.Open()
	buf := readerBytes(fr)

	// Do not strip <w: entiraly, but keep reference as w-t
	// So any string without w: would stay same, but all w- will be replaced again
	buf = bytes.Replace(buf, []byte("<w:"), []byte("<w-"), -1)
	buf = bytes.Replace(buf, []byte("</w:"), []byte("</w-"), -1)

	xnode := &xmlNode{}
	if err := xml.Unmarshal(buf, &xnode); err != nil {
		color.Red("fileToXMLStruct: %v", err)
	}

	// Assign parent nodes to all nodes
	xnode.Walk(func(xnode *xmlNode) {
		for _, n := range xnode.Nodes {
			n.parent = xnode
		}
	})

	// color.Cyan("%s", structToXMLBytes(n))
	return xnode
}

// Row placeholders - clone row, append to existing structure and replace values
// Numbers: []int{1,3,5}
// {{Numbers}}
func (t *Template) replaceRowParams(xnode *xmlNode) {
	xnode.Walk(func(nrow *xmlNode) {

		if !nrow.isRowElement() {
			return
		}

		contents := nrow.Contents()

		if !bytes.Contains(contents, []byte("{{")) {
			// without any params
			return
		}

		color.Cyan("ROW: %s", contents)

		for pKey, pVal := range t.params {
			if fmt.Sprintf("%T", pVal)[:2] != "[]" {
				// only any kind of slices are valid
				continue
			}

			if !bytes.Contains(contents, []byte("{{"+pKey+"}}")) && !bytes.Contains(contents, []byte("{{#"+pKey+"}}")) {
				// specific placeholder not found
				continue
			}

			// interface{} to string slice
			values := toStringSlice(pVal)
			color.HiCyan("\t{{%s}}: %v", pKey, values)

			for i, sVal := range values {
				sindex := fmt.Sprintf("%d", i+1) // because 0 is for programmers first number in a row
				nnew := nrow.cloneAndAppend()
				nnew.Walk(func(nnew *xmlNode) {
					nnew.Content = bytes.Replace(nnew.Content, []byte("{{#"+pKey+"}}"), []byte(sindex), -1)
					nnew.Content = bytes.Replace(nnew.Content, []byte("{{"+pKey+"}}"), []byte(sVal), -1)
				})
			}
			nrow.delete()
		}

	})
}

// Inline placeholders - clone text node, append to existing structure and replace values
// Numbers: []int{1,3,5}
// {{Numbers ,}}
func (t *Template) replaceColumnParams(xnode *xmlNode) {
	xnode.Walk(func(n *xmlNode) {
		if bytes.Index(n.Content, []byte("{{")) >= 0 {
			for pKey, pVal := range t.params {

				pholder := []byte("{{" + pKey + " ") //space at the end

				if fmt.Sprintf("%T", pVal)[:2] != "[]" {
					// only any kind of slices are valid
					continue
				}

				// with space {{Placeholder ,}}, {{Placeholder , }}
				if !bytes.Contains(n.Content, pholder) {
					// specific placeholder not found
					continue
				}

				// Separator is last part of placeholder after space
				// {{Numbers ,}} --> ","
				// {{Numbers  , }} --> " , " // spaces around
				var sep []byte
				arr := bytes.SplitN(n.Content, pholder, 2) // aaaa {{Numbers ,}} bbb
				if len(arr) == 2 {
					arr = bytes.SplitN(arr[1], []byte("}}"), 2) // ,}} bbb
					sep = arr[0]                                // ,
				}
				color.Blue("SEP[%s]", sep)

				placeholder := fmt.Sprintf("{{%s %s}}", pKey, sep) // {{Placeholder}}

				// interface{} to string slice
				values := toStringSlice(pVal)
				color.HiCyan("\t{{%s}}: %v", pKey, values)

				for _, val := range values {
					sval := fmt.Sprintf("%v%s", val, sep) // interface{} to string
					n.Content = bytes.Replace(n.Content, []byte(placeholder), []byte(sval+placeholder), -1)
				}
				n.Content = bytes.Replace(n.Content, []byte(string(sep)+placeholder), nil, 1)
				// n.Content = bytes.Replace(n.Content, []byte(placeholder), nil, 1)

			}
		}
	})
}
func (t *Template) replaceSingleParams(xnode *xmlNode) {
	xnode.Walk(func(n *xmlNode) {
		if bytes.Index(n.Content, []byte("{{")) >= 0 {
			// Try to replace on node that contains possible placeholder
			for pKey, pVal := range t.params {
				placeholder := fmt.Sprintf("{{%s}}", pKey) // {{Placeholder}}
				sval := fmt.Sprintf("%v", pVal)            // interface{} to string
				n.Content = bytes.Replace(n.Content, []byte(placeholder), []byte(sval), -1)
			}
		}
	})
}

// Params  - replace template placeholders with params
// "Hello {{ Name }}!"" --> "Hello World!""
func (t *Template) Params(v interface{}) {
	t.params = collectParams("", v)

	f := t.MainDocument() // TODO: loop all xml files
	xnode := t.fileToXMLStruct(f.Name)

	t.replaceRowParams(xnode)
	t.replaceColumnParams(xnode)
	t.replaceSingleParams(xnode)

	// Save []bytes
	t.modified[f.Name] = structToXMLBytes(xnode)

	return

	xnode.Walk(func(xnode *xmlNode) {
		// Any param
		rxpattern := `{{( |row\.|col\.)(\w|\d|\.|_)+}}`
		isMatch, err := regexp.Match(rxpattern, xnode.Content)
		if !isMatch || err != nil {
			// placeholders not found, skip
			return
		}

		// Get parent ROW element to multiply
		// p, tblRow
		nrow := xnode.parent
		for {
			if nrow == nil {
				break
			}

			if !nrow.isRowElement() {
				nrow = nrow.parent
				continue
			}

			markForRemoval := false

			// Loop params and find slices for row cloning
			for pKey, pVal := range t.params {
				var svalues []string

				// convert all slice values to string values
				switch arr := pVal.(type) {
				case []string:
					color.Red("STRING: %v", arr)
				// 	svalues = arr
				case []int:
					for _, val := range arr {
						// prepend item because cloning row adds new lines in reverse
						sval := fmt.Sprintf("%v", val)
						svalues = append([]string{sval}, svalues...)
					}
				default:
					continue
				}

				// Row clone and replace
				if bytes.Contains(nrow.Contents(), []byte("row.")) {
					markForRemoval = true
					for _, s := range svalues {
						// Row clone and replace
						nnew := nrow.cloneAndAppend()
						nnew.Walk(func(nnew *xmlNode) {
							// Replace in every sub-node as it there could be multiple plain text nodes
							nnew.Content = bytes.Replace(nnew.Content, []byte("{{row."+pKey+"}}"), []byte(s), -1)
						})
					}
				}

				// Col clone and replace
				// color.Red("{{col.%s}}", pKey)
				if bytes.Contains(nrow.Contents(), []byte("col.")) {
					for i, s := range svalues {
						nrow.Walk(func(n *xmlNode) {
							// Replace in every sub-node as it there could be multiple plain text nodes
							if bytes.Contains(n.Content, []byte("{{col.")) {
								nnew := n.cloneAndAppend()
								nnew.Content = bytes.Replace(nnew.Content, []byte("{{col."+pKey+"}}"), []byte(s), -1)
								if i == 0 {
									color.Red("i==0 -- %v", pKey)
									// Last item trim. Index=0 because of cloning last cloned item is first in list
									nnew.Content = bytes.TrimRight(nnew.Content, ",;- \t\n|/+")
								}
								if i == len(svalues)-1 {
									color.Red("i==last -- %v", pKey)
									// After last item cloned remove clone original woith placeholder param
									n.delete()
								}
							}
						})
					}
				}

			}

			if markForRemoval {
				nrow.delete() //delete original placeholder row
			}

			nrow = nrow.parent

		}
		color.Yellow("%-50s (%v)", string(xnode.Content), xnode.parentString(6))
	})

	// When replace massive simple params: single int, string or single Struct.string
	// fr, _ := f.Open()
	// buf := readerBytes(fr)
	buf := structToXMLBytes(xnode)

	for k, v := range t.params {
		pkey := fmt.Sprintf("{{%s}}", k)
		pval := fmt.Sprintf("%v", v)
		color.Green("%-20s %-10T %v", k, v, v)
		buf = bytes.Replace(buf, []byte(pkey), []byte(pval), -1)
	}

	t.modified[f.Name] = buf
}

// ExportDocx - save new/modified docx based on template
func (t *Template) ExportDocx(path string) {
	fDocx, err := os.Create(path)
	if err != nil {
		return
	}
	defer fDocx.Close()

	zipw := zip.NewWriter(fDocx)
	defer zipw.Close()

	// Loop existing files to build docx archive again
	for _, f := range t.files {
		var err error

		// Read contents of single file inside zip
		var fr io.ReadCloser
		if fr, err = f.Open(); err != nil {
			log.Printf("Error reading [ %s ] from archive", f.Name)
			continue
		}
		fbuf := new(bytes.Buffer)
		fbuf.ReadFrom(fr)
		fr.Close()

		// Write contents as single file inside zip
		var fw io.Writer
		if fw, err = zipw.Create(f.Name); err != nil {
			log.Printf("Error writing [ %s ] to archive", f.Name)
			continue
		}

		// Move/Write struct-saved file to docx archive file back
		if buf, isModified := t.modified[f.Name]; isModified {

			// // // file to XML nodes struct
			// xmlNodes := t.fileToXMLStruct(f.Name)
			// buf := structToXMLBytes(xmlNodes)

			// head := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
			// buf = append(head, buf...)
			fw.Write(buf)

			ioutil.WriteFile("XXX.xml", buf, 0666) // DEBUG
			continue
		}

		fw.Write(fbuf.Bytes())
	}

	return
}
