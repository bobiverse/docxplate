package docxplate

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
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

// Convert given bytes to struct of xml nodes
func (t *Template) bytesToXMLStruct(buf []byte) *xmlNode {
	// Do not strip <w: entiraly, but keep reference as w-t
	// So any string without w: would stay same, but all w- will be replaced again
	buf = bytes.Replace(buf, []byte("<w:"), []byte("<w-"), -1)
	buf = bytes.Replace(buf, []byte("</w:"), []byte("</w-"), -1)

	xdocNode := &xmlNode{}
	if err := xml.Unmarshal(buf, &xdocNode); err != nil {
		log.Printf("fileToXMLStruct: %v", err)
	}

	// Assign parent nodes to all nodes
	xdocNode.Walk(func(xnode *xmlNode) {
		if xnode.Tag() == "w-body" {
			xnode.parent = xdocNode
		}

		for _, n := range xnode.Nodes {
			n.parent = xnode
		}
	})

	// log.Printf("%s", structToXMLBytes(n))
	return xdocNode
}

// Convert given file (from template.Files) to struct of xml nodes
func (t *Template) fileToXMLStruct(fname string) *xmlNode {
	f, ok := t.files[fname]
	if !ok {
		return nil
	}

	fr, _ := f.Open()
	buf := readerBytes(fr)

	return t.bytesToXMLStruct(buf)
}

// Expand some placeholders to enable row replacer replace them
// Users: []User{ User{Name:AAA}, User{Name:BBB} }
// {{Users.Name}} -->
//      {{Users.1.Name}}
//      {{Users.2.Name}}

func (t *Template) expandPlaceholders(xnode *xmlNode) {
	t.params.Walk(func(p *Param) {
		if !p.IsSlice {
			return
		}

		prefixes := []string{
			p.PlaceholderPrefix(),
			p.ToCompact(p.PlaceholderPrefix()),
		}

		for _, prefix := range prefixes {
			xnode.Walk(func(nrow *xmlNode) {
				if nrow.isNew {
					return
				}
				if !nrow.isRowElement() {
					return
				}
				if !nrow.AnyChildContains([]byte(prefix)) {
					return
				}

				// fmt.Printf("%-30s - %s", prefix, nrow.Contents())
				for _, p2 := range p.Params {
					// fmt.Printf("\tCLONE: %s -- %s -- %s\n", prefix+p2.Key, p2.PlaceholderPrefix(), nrow.Contents())
					nnew := nrow.cloneAndAppend()
					nnew.Walk(func(n *xmlNode) {
						if !inSlice(n.XMLName.Local, []string{"w-t"}) {
							return
						}
						n.Content = bytes.Replace(n.Content, []byte(prefix), []byte(p2.PlaceholderPrefix()), -1) // w-t
						n.haveParam = true
						// n.parent.Content = nil                                                                   // w-r
						//
						// color.Magenta("\t clone: %s", n)
						// color.HiMagenta("\t clone: %s -- %s", n.Content, n.Contents())
					})
				}
				nrow.delete()
			})
		}

		// }
		// fmt.Printf("\n")
	})

	// Cloned nodes are marked as new by default.
	// After expanding mark as old so next operations doesn't ignore them
	xnode.Walk(func(n *xmlNode) {
		n.isNew = false
	})
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
				// fmt.Printf("%30s = %v", p.Placeholder(), p2.Value)
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
		contents := n.AllContents()

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
		if n == nil || n.isDeleted {
			return
		}

		if bytes.Index(n.Content, []byte("{{")) >= 0 {
			// Try to replace on node that contains possible placeholder
			t.params.Walk(func(p *Param) {
				if p.IsSlice {
					// do not replace slice/map values here. Only singles
					return
				}

				// Trigger: does placeholder have trigger
				if p.extractTriggerFrom(n.Content) != nil {
					n.Content = bytes.Replace(n.Content, []byte(p.PlaceholderWithTrigger()), []byte(p.Value), -1)
					n.Content = bytes.Replace(n.Content, []byte(p.PlaceholderKeyWithTrigger()), []byte(p.Key), -1)
					p.RunTrigger(n)
					return
				}

				// fmt.Printf("%30s --> %+v", p.Placeholder(), p.Value)
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
	switch val := v.(type) {
	case string:
		t.params = JSONToParams([]byte(val))
	case []byte:
		t.params = JSONToParams(val)
	default:
		t.params = StructParams(val)
	}

	f := t.MainDocument() // TODO: loop all xml files
	xnode := t.fileToXMLStruct(f.Name)

	// While formating docx sometimes same style node is split to
	// multiple same style nodes and different content
	// Merge them so placeholders are in the same node
	t.fixBrokenPlaceholders(xnode)

	// First try to replace all exact-match placeholders
	// Do it before expand because it may expand unwanted placeholders
	t.replaceSingleParams(xnode)

	// Complex placeholders with more depth needs to be expanded
	// for correct replace
	t.expandPlaceholders(xnode)

	// xnode.Walk(func(xn *xmlNode) {
	// 	if xn.XMLName.Local == "w-tbl" {
	// 		xn.printTree("BEFORE REPLACE")
	// 	}
	// })

	t.replaceRowParams(xnode)
	t.replaceInlineParams(xnode)
	t.replaceSingleParams(xnode)

	// // DEBUG:
	// for _, p := range t.params {
	// 	fmt.Printf("|| %-20s %v", p.Key, p.Value)
	// }

	// Save []bytes
	t.modified[f.Name] = structToXMLBytes(xnode)
}

// Fix broken placeholders by merging "w-t" nodes.
// "w-p" (Record) can hold multiple "w-r". And "w-r" holts "w-t" node
// -
// If these nodes not fixed than params replace can not be done as
// replacer process nodes one by one
func (t *Template) fixBrokenPlaceholders(xnode *xmlNode) {
	xnode.Walk(func(xnode *xmlNode) {
		if !xnode.isRowElement() {
			// broken placeholders are in row elements
			return
		}

		if !xnode.HaveParams() {
			// whole text doesn't hold any params
			return
		}

		xnode.WalkTree(0, func(depth int, wrNode *xmlNode) {
			if wrNode.XMLName.Local != "w-r" {
				return
			}

			// have end but doesn't have beginning in the same node
			// NOTE: parent "w-p" should contain broken placeholder end "}}"
			isBrokenEnd := bytes.Contains(wrNode.AllContents(), []byte("}}"))
			isBrokenEnd = isBrokenEnd && !bytes.Contains(wrNode.AllContents(), []byte("{{"))
			isBrokenEnd = isBrokenEnd && bytes.Contains(wrNode.parent.AllContents(), []byte("}}"))

			if !isBrokenEnd {
				return
			}

			var keepNode *xmlNode
			wrNode.parent.WalkTree(depth, func(depth int, wtNode *xmlNode) {
				if wtNode.XMLName.Local != "w-t" {
					return
				}

				// finished: skip rest
				if keepNode != nil && bytes.Contains(keepNode.Content, []byte("}}")) {
					return
				}

				// Found placeholder start
				if bytes.Contains(wtNode.Content, []byte("{{")) {
					keepNode = wtNode
					return
				}

				if keepNode == nil {
					return
				}

				// Append contens if not completed placeholder
				keepNode.Content = append(keepNode.Content, wtNode.Content...)
				// `w-t` node is under `w-r` so remove parent `w-r`
				wtNode.parent.delete()
			})

			// fmt.Printf("Merged: %s", keepNode)

		})

	})
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
			fw.Write(buf)
			continue
		}

		fw.Write(fbuf.Bytes())
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

	err = ioutil.WriteFile(path, buf, 0644)

	return err
}

// Placeholders - get list of used params placeholders in template
// If you already replaced params with values then you will not get all placeholders.
// Or use it after replace and see how many placeholders left.
func (t *Template) Placeholders() []string {
	var arr []string

	plaintext := t.Plaintext()

	// re := regexp.MustCompile("{{(#|)([a-zA-Z0-9_\\-\\.])+( .|)}}")
	re := regexp.MustCompile("{{(#|)[\\w\\.]+?(| .| )+?}}")
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

	f := t.MainDocument() // TODO: loop all xml files
	xnode := t.bytesToXMLStruct(t.modified[f.Name])

	xnode.Walk(func(n *xmlNode) {
		if n.XMLName.Local != "w-r" {
			return
		}

		s := string(n.AllContents())
		plaintext += s
		if s != "" {
			plaintext += "\n"
		}
	})

	return plaintext
}
