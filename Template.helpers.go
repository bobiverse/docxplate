package docxplate

import (
	"bytes"
	"encoding/xml"
	"log"
	"strings"
)

// Convert given bytes to struct of xml nodes
func (t *Template) bytesToXMLStruct(buf []byte) *xmlNode {
	// Do not strip <w: entiraly, but keep reference as w-t
	// So any string without w: would stay same, but all w- will be replaced again
	buf = bytes.ReplaceAll(buf, []byte("<w:"), []byte("<w-"))
	buf = bytes.ReplaceAll(buf, []byte("</w:"), []byte("</w-"))
	buf = bytes.ReplaceAll(buf, []byte("<v:"), []byte("<v-"))
	buf = bytes.ReplaceAll(buf, []byte("</v:"), []byte("</v-"))

	xdocNode := &xmlNode{}
	if err := xml.Unmarshal(buf, xdocNode); err != nil {
		log.Printf("fileToXMLStruct: %v", err)
	}

	xdocNode.FixNamespaceDuplication()
	// Assign parent nodes to all nodes
	xdocNode.Walk(func(xnode *xmlNode) {
		xnode.FixNamespaceDuplication()

		if xnode.Tag() == "w-body" {
			xnode.parent = xdocNode
		}
		if xnode.childFirst != nil {
			xnode.childFirst.iterate(func(node *xmlNode) bool {
				node.parent = xnode
				return false
			})
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

// wrapper for simple param replace func
func (t *Template) replaceTextParam(xnode *xmlNode, param *Param) {
	xnode.Content = param.replaceIn(xnode.Content)
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
			xnode.add(contentNode)
		}
		// image node
		if len(contentSlice)-i > 1 {
			imgNode := t.bytesToXMLStruct([]byte(param.Value))
			imgNode.parent = xnode.parent
			xnode.add(imgNode)
		}
	}
	// Empty the content before deleting to prevent reprocessing when params walk
	xnode.Content = []byte("")
	xnode.delete()
}

// Check for broken placeholders
func (t *Template) matchBrokenPlaceholder(content string, isLeft bool) bool {
	stack := 0

	for i := 1; i < len(content); i++ {
		if content[i] == '{' && content[i-1] == '{' {
			stack++
			i++ // Skip next character
			continue
		}
		if content[i] == '}' && content[i-1] == '}' {
			if stack > 0 {
				stack--
				i++ // Skip next character
				continue
			}

			if !isLeft {
				return true // Broken right placeholder
			}
		}
	}

	return isLeft && stack > 0 // Broken left placeholder
}

// Match left part placeholder `{{`
func (t *Template) matchBrokenLeftPlaceholder(content string) bool {
	return t.matchBrokenPlaceholder(content, true)
}

// UNUSED
// // Match right placeholder part `}}`
// func (t *Template) matchBrokenRightPlaceholder(content string) bool {
// 	return t.matchBrokenPlaceholder(content, false)
// }

// GetAttrParam - extracts and returns substrings enclosed in double curly braces "{{...}}" from the given string
func (t Template) GetAttrParam(attr string) []string {
	var ret []string
	var record strings.Builder
	start := false
	length := len(attr)
	for i := 1; i < length-1; i++ {

		if attr[i] == '{' && attr[i-1] == '{' {
			start = true
			continue
		}

		if start && (attr[i] == ' ' || (attr[i] == '}' && length-1 > i && attr[i+1] == '}')) {
			ret = append(ret, record.String())
			record.Reset()
			start = false
		}

		if start {
			record.WriteByte(attr[i])
		}
	}

	return ret
}
