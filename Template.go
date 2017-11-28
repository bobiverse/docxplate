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
	"strings"

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
	params ParamList
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

		// Loop all params and try to replace
		t.params.Walk(func(p *Param) {
			if p.Params == nil {
				return
			}

			// isValidKey := bytes.Contains(contents, []byte(p.Placeholder()))                 // {{Name}} --> John
			// isValidKey = isValidKey || bytes.Contains(contents, []byte(p.PlaceholderKey())) // {{#Name}} --> 0
			// isValidKey = isValidKey || bytes.Contains(contents, []byte(p.PlaceholderMultiple())) // {{Name.FirstLetter}} --> J

			isValidKey := nrow.AnyChildContains([]byte(p.Placeholder()))
			isValidKey = isValidKey || nrow.AnyChildContains([]byte(p.PlaceholderKey()))
			isValidKey = isValidKey || nrow.AnyChildContains([]byte(p.PlaceholderMultiple()))

			if !isValidKey {
				// specific placeholder not found
				return
			}

			// Add new xml nodes for every param sub-param
			for _, p2 := range p.Params {
				color.Blue("%30s = %v", p.Placeholder(), p2.Value)
				nnew := nrow.cloneAndAppend()
				nnew.Walk(func(nnew *xmlNode) {
					color.HiBlue("\t%30s = %s", p.Placeholder(), nnew.Content)

					// oldContent := nnew.Content
					nnew.Content = bytes.Replace(nnew.Content, []byte(p.Placeholder()), []byte(p2.Value), -1)
					nnew.Content = bytes.Replace(nnew.Content, []byte(p.PlaceholderKey()), []byte(p2.Key), -1)
					// if bytes.Equal(oldContent, nnew.Content) {
					// 	nnew.Content = []byte("xxx")
					// }
				})
			}

			// Remove original row which contains placeholder
			nrow.delete()
		})

	})
}

// Inline placeholders - clone text node, append to existing structure and replace values
// Numbers: []int{1,3,5}
// {{Numbers ,}}
func (t *Template) replaceColumnParams(xnode *xmlNode) {
	xnode.Walk(func(n *xmlNode) {
		if bytes.Index(n.Content, []byte("{{")) >= 0 {
			for _, p := range t.params {

				pholder := []byte("{{" + p.Key + " ") //space at the end

				vtype := fmt.Sprintf("%T", p.Value)
				isSlice := strings.HasPrefix(vtype, "[]")
				isMap := strings.HasPrefix(vtype, "map[")
				if !isSlice && !isMap {
					// slices and maps are allowed
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

				placeholder := fmt.Sprintf("{{%s %s}}", p.Key, sep) // {{Placeholder}}

				// interface{} to string slice
				values := toMap(p.Value)
				color.HiCyan("\t{{%s}}: %v", p.Key, values)

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
			t.params.Walk(func(p *Param) {
				// color.Blue("%30s --> %+v", p.Placeholder(), p.Value)
				n.Content = bytes.Replace(n.Content, []byte(p.Placeholder()), []byte(p.Value), -1)
			})
		}
	})
}

// Params  - replace template placeholders with params
// "Hello {{ Name }}!"" --> "Hello World!""
func (t *Template) Params(v interface{}) {
	// t.params = collectParams("", v)
	t.params = StructParams(v)

	f := t.MainDocument() // TODO: loop all xml files
	xnode := t.fileToXMLStruct(f.Name)

	t.mergeSimilarNodes(xnode)

	t.replaceRowParams(xnode)
	// t.replaceColumnParams(xnode)
	// t.replaceSingleParams(xnode)

	for _, p := range t.params {
		color.Green("|| %-20s %v", p.Key, p.Value)
	}

	// Save []bytes
	t.modified[f.Name] = structToXMLBytes(xnode)
}

// Merge similar nodes of same styles.
// Like "w-p" (Record) can hold multiple "w-r" with same styles
// -
// If these nodes not fixed than params replace can not be done as
// replacer process nodes one by one
func (t *Template) mergeSimilarNodes(xnode *xmlNode) {
	xnode.Walk(func(xnode *xmlNode) {
		if !bytes.Contains(xnode.Contents(), []byte("{{")) {
			return
		}
		// parent scope
		// color.Yellow("%v", xnode.XMLName)

		var nprev *xmlNode
		xnode.Walk(func(n *xmlNode) {
			//child scope
			if n.XMLName.Local != "w-r" {
				return
			}

			if nprev != nil {

				// Merge only same parent nodes
				isMergable := nprev.parent == n.parent
				isMergable = isMergable && n.StylesString() == nprev.StylesString()

				// color.Magenta("\n\n\nM0: %v / %v", nprev.parent == n.parent, nprev.StylesString() == n.StylesString())
				// color.HiMagenta("S1: %s", nprev.StylesString())
				// color.HiMagenta("S2: %s", n.StylesString())
				// color.Cyan("\tM1: Parent:%p %s", nprev.parent, nprev.Contents())
				// color.HiCyan("\tM2: Parent:%p %s", n.parent, n.Contents())

				if isMergable {
					color.Yellow("\tMERGE: %s%s", nprev.Contents(), color.HiYellowString("%s", n.Contents()))
					bufMerged := append(nprev.Contents(), n.Contents()...)
					nprev.ReplaceInContents(nprev.Contents(), bufMerged)
					// n.ReplaceInContents(n.Contents(), nil)
					n.delete()
					return
				}
			}

			nprev = n
		})

	})
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
