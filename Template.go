package docxplate

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
)

var t *Template

// Template ..
type Template struct {
	path string
	file *os.File
	zipw *zip.Writer     // zip writer
	zipr *zip.ReadCloser // zip reader

	// save all zip files here so we can build it again
	files map[string]*zip.File
	// content type document file
	documentContentTypes *zip.File
	// main document file
	documentMain *zip.File
	// document relations
	documentRels map[string]*zip.File
	// only added files (converted to []byte) save here
	added map[string][]byte
	// only modified files (converted to []byte) save here
	modified map[string][]byte

	// hold all parsed params:values here
	params ParamList
}

// OpenTemplate .. docpath local file
func OpenTemplate(docpath string) (*Template, error) {
	var err error

	// Init doc template
	t = &Template{
		path:         docpath,
		files:        map[string]*zip.File{},
		documentRels: map[string]*zip.File{},
		added:        map[string][]byte{},
		modified:     map[string][]byte{},
	}

	// Unzip
	if t.zipr, err = zip.OpenReader(t.path); err != nil {
		return nil, err
	}

	// Get main document
	for _, f := range t.zipr.File {
		t.files[f.Name] = f
		if f.Name == "[Content_Types].xml" {
			t.documentContentTypes = f
		}
		if f.Name == "word/document.xml" {
			t.documentMain = f
		}
		if path.Ext(f.Name) == ".rels" {
			t.documentRels[f.Name] = f
		}

	}

	if t.documentMain == nil {
		return nil, fmt.Errorf("mandatory [ word/document.xml ] not found")
	}

	return t, nil
}

// OpenTemplateWithURL .. docpath is remote url
func OpenTemplateWithURL(docurl string) (tpl *Template, err error) {
	docpath, err := downloadFile(docurl)
	if err != nil {
		return nil, err
	}
	defer os.Remove(docpath)
	tpl, err = OpenTemplate(docpath)
	if err != nil {
		return nil, err
	}
	return
}

// Convert given bytes to struct of xml nodes
func (t *Template) bytesToXMLStruct(buf []byte) *xmlNode {
	// Do not strip <w: entiraly, but keep reference as w-t
	// So any string without w: would stay same, but all w- will be replaced again
	buf = bytes.ReplaceAll(buf, []byte("<w:"), []byte("<w-"))
	buf = bytes.ReplaceAll(buf, []byte("</w:"), []byte("</w-"))
	buf = bytes.ReplaceAll(buf, []byte("<v:"), []byte("<v-"))
	buf = bytes.ReplaceAll(buf, []byte("</v:"), []byte("</v-"))

	xdocNode := &xmlNode{}
	if err := xml.Unmarshal(buf, &xdocNode); err != nil {
		log.Printf("fileToXMLStruct: %v", err)
	}

	xdocNode.FixNamespaceDuplication()
	// Assign parent nodes to all nodes
	xdocNode.Walk(func(xnode *xmlNode) {
		xnode.FixNamespaceDuplication()

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
// Note: Currently only struct type support image replacement
// Users: []User{ User{Name:AAA}, User{Name:BBB} }
// {{Users.Name}} -->
//      {{Users.1.Name}}
//      {{Users.2.Name}}

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
		if reflect.ValueOf(v).Kind() == reflect.Struct {
			t.params = StructToParams(val)
		} else {
			t.params = StructParams(val)
		}
	}

	f := t.documentMain // TODO: loop all xml files
	xnode := t.fileToXMLStruct(f.Name)

	// Enchance some markup (removed when building XML in the end)
	// so easier to find some element
	t.enchanceMarkup(xnode)

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

	// xnode.Walk(func(n *xmlNode) {
	// 	if is, _ := n.IsListItem(); is {
	// 		n.Walk(func(wt *xmlNode) {
	// 			if wt.Tag() == "w-t" {
	// 				color.Yellow("%s", wt)
	// 			}
	// 		})
	// 	}
	// })

	// Save []bytes
	t.modified[f.Name] = structToXMLBytes(xnode)
}

// Collect and trigger placeholders with trigger but unset in `t.params`
// Placeholders with trigger `:empty` must be triggered
// otherwise they are left
func (t *Template) triggerMissingParams(xnode *xmlNode) {
	if t.params == nil {
		return
	}

	var triggerParams ParamList

	xnode.Walk(func(n *xmlNode) {
		if !n.isRowElement() || !n.HaveParams() {
			return
		}

		p := NewParamFromRaw(n.AllContents())
		if p != nil && p.Trigger != nil {
			triggerParams = append(triggerParams, p)
		}
	})

	if triggerParams == nil {
		return
	}

	// make sure not to "tint" original t.params
	_params := t.params
	t.params = triggerParams

	// do stuff only with filtered params
	t.replaceSingleParams(xnode, true)

	// back to original
	t.params = _params
}

type placeholderType int8

const (
	singlePlaceholder placeholderType = iota
	inlinePlaceholder
	rowPlaceholder
)

type placeholder struct {
	Type         placeholderType
	Placeholders []string
	Separator    string
}

// Expand complex placeholders
func (t *Template) expandPlaceholders(xnode *xmlNode) {
	t.params.Walk(func(p *Param) {
		if p.Type != SliceParam {
			return
		}

		prefixes := []string{
			p.PlaceholderPrefix(),
			p.ToCompact(p.PlaceholderPrefix()),
		}

		var max int
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

				contents := nrow.AllContents()
				rowParams := rowParams(contents)
				rowPlaceholders := make(map[string]*placeholder)
				// Collect placeholder that for expansion
				for _, rowParam := range rowParams {
					var placeholderType placeholderType
					if len(rowParam.Separator) > 0 {
						placeholderType = inlinePlaceholder
					} else {
						placeholderType = rowPlaceholder
					}

					var trigger string
					if rowParam.Trigger != nil {
						trigger = " " + rowParam.Trigger.String()
					}

					var isMatch bool
					var index = -1
					currentLevel := p.Level
					placeholders := make([]string, 0, len(p.Params))
					p.WalkFunc(func(p *Param) {
						if p.Level == currentLevel+1 {
							index++
						}
						if rowParam.AbsoluteKey == p.CompactKey {
							isMatch = true
							placeholders = append(placeholders, "{{"+p.AbsoluteKey+trigger+"}}")
						}
					})

					if isMatch {
						rowPlaceholders[rowParam.RowPlaceholder] = &placeholder{
							Type:         placeholderType,
							Placeholders: placeholders,
							Separator:    strings.TrimLeft(rowParam.Separator, " "),
						}

						if max < len(placeholders) {
							max = len(placeholders)
						}
					}
				}
				// Expand placeholder exactly
				nnews := make([]*xmlNode, max, max)
				for oldPlaceholder, newPlaceholder := range rowPlaceholders {
					switch newPlaceholder.Type {
					case inlinePlaceholder:
						nrow.Walk(func(n *xmlNode) {
							if !inSlice(n.XMLName.Local, []string{"w-t"}) || len(n.Content) == 0 {
								return
							}
							n.Content = bytes.ReplaceAll(n.Content, []byte(oldPlaceholder), []byte(strings.Join(newPlaceholder.Placeholders, newPlaceholder.Separator)))
						})
					case rowPlaceholder:
						defer func() {
							nrow.delete()
						}()
						for i, placeholder := range newPlaceholder.Placeholders {
							if nnews[i] == nil {
								nnews[i] = nrow.cloneAndAppend()
							}
							nnews[i].Walk(func(n *xmlNode) {
								if !inSlice(n.XMLName.Local, []string{"w-t"}) || len(n.Content) == 0 {
									return
								}
								n.Content = bytes.ReplaceAll(n.Content, []byte(oldPlaceholder), []byte(placeholder))
							})
						}
					}
				}
			})
		}
	})

	// Cloned nodes are marked as new by default.
	// After expanding mark as old so next operations doesn't ignore them
	xnode.Walk(func(n *xmlNode) {
		n.isNew = false
	})
}

// Replace single params by type
func (t *Template) replaceSingleParams(xnode *xmlNode, triggerParamOnly bool) {
	xnode.Walk(func(n *xmlNode) {
		if n == nil || n.isDeleted {
			return
		}

		if bytes.Contains(n.Content, []byte("{{")) {
			// Try to replace on node that contains possible placeholder
			t.params.Walk(func(p *Param) {
				// Only string and image param to replace
				if p.Type != StringParam && p.Type != ImageParam {
					return
				}
				// Prefix check
				if !bytes.Contains(n.Content, []byte(p.PlaceholderPrefix())) {
					return
				}
				// Trigger: does placeholder have trigger
				if p.Trigger = p.extractTriggerFrom(n.Content); p.Trigger != nil {
					defer func() {
						p.RunTrigger(n)
					}()
				}
				if triggerParamOnly {
					return
				}
				// Repalce by type
				switch p.Type {
				case StringParam:
					t.replaceStringParams(n, p)
				case ImageParam:
					t.replaceImageParams(n, p)
				}
			})
		}
	})
}

// String placeholder replace
func (t *Template) replaceStringParams(xnode *xmlNode, param *Param) {
	xnode.Content = bytes.ReplaceAll(xnode.Content, []byte(param.Placeholder()), []byte(param.Value))
	xnode.Content = bytes.ReplaceAll(xnode.Content, []byte(param.PlaceholderKey()), []byte(param.Key))
	return
}

// Image placeholder replace
func (t *Template) replaceImageParams(xnode *xmlNode, param *Param) {
	// Sometime the placeholder is in the before or middle of the text, but node is appended in the last.
	// So, we have to split the text and image into different nodes to achieve cross-display.
	contentSlice := bytes.Split(xnode.Content, []byte(param.Placeholder()))
	for i, content := range contentSlice {
		// text node
		if len(content) != 0 {
			contentNode := &xmlNode{
				XMLName: xml.Name{Space: "", Local: "w-t"},
				Content: content,
				parent:  xnode.parent,
				isNew:   true,
			}
			xnode.parent.Nodes = append(xnode.parent.Nodes, contentNode)
		}
		// image node
		if len(contentSlice)-i > 1 {
			imgNode := t.bytesToXMLStruct([]byte(param.Value))
			imgNode.parent = xnode.parent
			xnode.parent.Nodes = append(xnode.parent.Nodes, imgNode)
		}
	}
	// Empty the content before deleting to prevent reprocessing when params walk
	xnode.Content = []byte("")
	xnode.delete()
	return
}

// Enchance some markup (removed when building XML in the end)
// so easier to find some element
func (t *Template) enchanceMarkup(xnode *xmlNode) {

	// List items - add list item node `w-p` attributes
	// so it's recognized as listitem
	xnode.Walk(func(n *xmlNode) {
		if n.Tag() != "w-p" {
			return
		}

		isListItem, listID := n.IsListItem()
		if !isListItem {
			return
		}

		// n.XMLName.Local = "w-item"
		n.Attrs = append(n.Attrs, xml.Attr{
			Name:  xml.Name{Local: "list-id"},
			Value: listID,
		})

	})
}

// This func is fixing broken placeholders by merging "w-t" nodes.
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

		var isMatchSingleLeftPlaceholder bool
		var isMatchSingleRightPlaceholder bool
		contents := xnode.AllContents()
		xnode.Walk(func(xnode *xmlNode) {
			if xnode.Content == nil || len(xnode.Content) == 0 {
				return
			}
			// Match right }} to sub or delete
			if isMatchSingleLeftPlaceholder {
				isMatchSingleRightPlaceholder = t.matchSingleRightPlaceholder(string(xnode.Content))
				if isMatchSingleRightPlaceholder {
					xnode.Content = xnode.Content[bytes.Index(xnode.Content, []byte("}}"))+2:]
				} else {
					xnode.delete()
					return
				}
			}
			// Match left {{  to fix broken
			isMatchSingleLeftPlaceholder = t.matchSingleLeftPlaceholder(string(xnode.Content))
			if isMatchSingleLeftPlaceholder {
				xnode.Content = append(xnode.Content, contents[bytes.Index(contents, xnode.Content)+len(xnode.Content):bytes.Index(contents, []byte("}}"))+2]...)
			}
			contents = contents[bytes.Index(contents, xnode.Content)+len(xnode.Content):]
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
		fw.Write(buf)
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

// Match single left placeholder ({{)
func (t *Template) matchSingleLeftPlaceholder(content string) bool {
	stack := make([]string, 0)

	for i, char := range content {
		if i > 0 {
			if char == '{' && content[i-1] == '{' {
				stack = append(stack, "{{")
			} else if char == '}' && content[i-1] == '}' && len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	return len(stack) > 0
}

// Match single right placeholder (}})
func (t *Template) matchSingleRightPlaceholder(content string) bool {
	stack := make([]string, 0)

	for i, char := range content {
		if i > 0 {
			if char == '{' && content[i-1] == '{' {
				stack = append(stack, "{{")
			} else if char == '}' && content[i-1] == '}' {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				} else {
					return true
				}
			}
		}
	}

	return false
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

	f := t.documentMain // TODO: loop all xml files
	xnode := t.bytesToXMLStruct(t.modified[f.Name])

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

	return plaintext
}
