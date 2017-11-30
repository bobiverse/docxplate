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

// Expand some placeholders to enable row replacer replace them
// Users: []User{ User{Name:AAA}, User{Name:BBB} }
// {{Users.Name}} -->
//      {{Users.1.Name}}
//      {{Users.2.Name}}
func (t *Template) expandPlaceholders(xnode *xmlNode) {

	t.params.Walk(func(p *Param) {
		if p.Depth() < 3 {
			return
		}

		// placeholder := p.ToCompact(p.Placeholder())
		placeholderPrefix := "{{" + p.parent.CompactKey + "."

		// walk xml nodes find what to clone and change
		xnode.Walk(func(nrow *xmlNode) {
			if nrow.isNew {
				return
			}
			if !nrow.isRowElement() || !nrow.HaveParams() {
				return
			}
			if p.Params == nil {
				return
			}
			if !nrow.AnyChildContains([]byte(placeholderPrefix)) {
				return
			}

			// Current "p" is child of slice
			// we need to get this child neighbors by
			// Parent -> childrens-slice -> Children
			// parent     .parent           .Params
			params := p.parent.parent.Params

			// color.Cyan("%-30s (%d) %s", placeholderPrefix, len(p.Params), nrow.Contents())

			for _, p2 := range params {
				// Clone for every
				nnew := nrow.cloneAndAppend()
				color.HiCyan("\tCLONE: %-10v %s", nrow.isNew, nrow.Contents())

				for _, p3 := range p2.Params {
					old := p3.ToCompact(p3.Placeholder())
					new := p3.Placeholder()

					color.HiCyan("\t\tREPLACE: %v %v", old, new)
					nnew.Walk(func(nnew *xmlNode) {
						nnew.Content = bytes.Replace(nnew.Content, []byte(old), []byte(new), -1)
					})
				}
			}
			nrow.delete()

			// for _, p2 := range p.Params {
			// color.Blue("%s %d", p.Placeholder(), len(p.Params))
			// nrow.cloneAndAppend()
			// nnew.Walk(func(nnew *xmlNode) {
			// 	nnew.Content = bytes.Replace(nnew.Content, []byte(placeholder), []byte(newPlaceholder), -1)
			// })
			// nrow.delete()
			// }
		})

	})

	// xnode.Walk(func(nrow *xmlNode) {
	// 	if !nrow.isRowElement() || !nrow.HaveParams() {
	// 		return
	// 	}
	//
	// 	// // Loop all params and try to replace
	// 	// t.params.Walk(func(p *Param) {
	// 	// 	if p.Depth() < 3 {
	// 	// 		return
	// 	// 	}
	//     //
	// 	// 	placeholder := p.ToComplex(p.Placeholder())
	// 	// 	isValidKey := nrow.AnyChildContains([]byte(placeholder))
	// 	// 	if !isValidKey {
	// 	// 		return
	// 	// 	}
	// 	// 	params := p.parent.parent.Params
	// 	// 	color.Blue("CX: %30s = %v", placeholder, params)
	//     //
	//     //
	// 	// 	for _, p2 := range params {
	// 	// 		newPlaceholder := strings.Replace(placeholder, p.ComplexKey, p2.AbsoluteKey+"."+p.Key, 1)
	// 	// 		color.HiBlue("CX:\t %30s ---> %s", placeholder, newPlaceholder)
	// 	// 		nrow.cloneAndAppend()
	// 	// 		// nnew.Walk(func(nnew *xmlNode) {
	// 	// 		// 	nnew.Content = bytes.Replace(nnew.Content, []byte(placeholder), []byte(newPlaceholder), -1)
	// 	// 		// })
	// 	// 	}
	// 	// 	nrow.delete()
	//     //
	// 	// 	// Add new xml nodes for every param sub-param
	// 	// 	// nnew := nrow.cloneAndAppend()
	// 	// 	// nnew.Walk(func(nnew *xmlNode) {
	// 	// 	// 	// for _, p2 := range params {
	// 	// 	// 	// 	color.HiBlue("CX:\t %30s = %v --> %v", placeholder, p.ComplexKey, p2.AbsoluteKey+"."+p.Key)
	// 	// 	// 	// 	newPlaceholder := strings.Replace(placeholder, p.ComplexKey, p2.AbsoluteKey+"."+p.Key, 1)
	// 	// 	// 	// 	nnew.Content = bytes.Replace(nnew.Content, []byte(placeholder), []byte(newPlaceholder), -1)
	// 	// 	// 	// }
	// 	// 	// })
	// 	// })
	//
	// })
}

// Row placeholders - clone row, append to existing structure and replace values
// Numbers: []int{1,3,5}
// {{Numbers}}
func (t *Template) replaceRowParams(xnode *xmlNode) {
	xnode.Walk(func(nrow *xmlNode) {

		if !nrow.isRowElement() || !nrow.HaveParams() {
			return
		}

		// Loop all params and try to replace
		t.params.Walk(func(p *Param) {
			if p.Params == nil {
				// Allow only slice params here
				return
			}
			// Do not check in nrow.Contents()
			// because it's checks merged nodes plaintext
			// But replacer works on every node separately
			isValidKey := nrow.AnyChildContains([]byte(p.Placeholder()))
			isValidKey = isValidKey || nrow.AnyChildContains([]byte(p.PlaceholderKey()))

			if !isValidKey {
				// specific placeholder not found
				return
			}
			// Add new xml nodes for every param sub-param
			for _, p2 := range p.Params {
				color.Blue("%30s = %v", p.Placeholder(), p2.Value)
				nnew := nrow.cloneAndAppend()
				nnew.Walk(func(nnew *xmlNode) {
					nnew.Content = bytes.Replace(nnew.Content, []byte(p.Placeholder()), []byte(p2.Value), -1)
					nnew.Content = bytes.Replace(nnew.Content, []byte(p.PlaceholderKey()), []byte(p2.Key), -1)
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
func (t *Template) replaceInlineParams(xnode *xmlNode) {
	xnode.Walk(func(n *xmlNode) {
		if !n.HaveParams() {
			return
		}
		contents := n.Contents()

		t.params.Walk(func(p *Param) {
			if p.Params == nil {
				return
			}

			placeholders := []string{
				p.PlaceholderInline(),    // "{{Key " - one side brackets
				p.PlaceholderKeyInline(), // "{{#Key "
			}

			for _, pholder := range placeholders {
				if !n.AnyChildContains([]byte(pholder)) {
					// specific placeholder not found
					continue
				}

				// Separator is last part of placeholder after space
				// {{Numbers ,}} --> ","
				// {{Numbers  , }} --> " , " // spaces around
				var sep string
				arr := bytes.SplitN(contents, []byte(pholder), 2) // aaaa {{Numbers ,}} bbb
				if len(arr) == 2 {
					arr = bytes.SplitN(arr[1], []byte("}}"), 2) // ,}} bbb
					sep = string(arr[0])                        //,
				}

				// Contructed full placeholder with both side brackets - {{Key , }}
				placeholder := fmt.Sprintf("{{%s %s}}", p.Key, sep) // {{Placeholder sep}}

				for _, p2 := range p.Params {
					n.Walk(func(n *xmlNode) {
						// Replace with new value and add same placeholder at the end
						// so we can replace next param
						n.Content = bytes.Replace(n.Content, []byte(placeholder), []byte(p2.Value+sep+placeholder), -1)
					})
				}
				// Remove placeholder so nobody replaces again this
				n.Walk(func(n *xmlNode) {
					n.Content = bytes.Replace(n.Content, []byte(sep+placeholder), nil, -1)
				})

			}

		})
	})
}
func (t *Template) replaceSingleParams(xnode *xmlNode) {
	xnode.Walk(func(n *xmlNode) {
		if bytes.Index(n.Content, []byte("{{")) >= 0 {
			// Try to replace on node that contains possible placeholder
			t.params.Walk(func(p *Param) {
				// color.Blue("%30s --> %+v", p.Placeholder(), p.Value)
				n.Content = bytes.Replace(n.Content, []byte(p.Placeholder()), []byte(p.Value), -1)
				n.Content = bytes.Replace(n.Content, []byte(p.PlaceholderKey()), []byte(p.Key), -1)
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
	t.expandPlaceholders(xnode)

	t.replaceRowParams(xnode)
	t.replaceInlineParams(xnode)
	t.replaceSingleParams(xnode)

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
		if !xnode.HaveParams() {
			return
		}

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
