package docxplate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
)

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content []byte     `xml:",chardata"`
	Nodes   []*xmlNode `xml:",any"`

	parent *xmlNode
	isNew  bool // added recently
}

// UnmarshalXML ..
func (xnode *xmlNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// n.Attrs = start.Attr
	type x xmlNode
	return d.DecodeElement((*x)(xnode), &start)
}

// Walk down all nodes and do custom stuff with given function
func (xnode *xmlNode) Walk(fn func(*xmlNode)) {
	for _, n := range xnode.Nodes {
		if n == nil {
			continue
		}

		fn(n) // do your custom stuff

		if len(n.Nodes) > 0 {
			//continue only if have deeper nodes
			n.Walk(fn)
		}
	}
}

// Contents - return contents of this and all childs contents merge
func (xnode *xmlNode) Contents() []byte {
	buf := xnode.Content
	xnode.Walk(func(n *xmlNode) {
		buf = append(buf, n.Content...)
	})
	return buf
}

// StylesString - string representation of styles of node
func (xnode *xmlNode) StylesString() string {
	buf := structToXMLBytes(xnode)

	// ignore some tags
	rgx, _ := regexp.Compile(`<w:sz.+?</w:sz(|.+?)>`) //w:sz, w:szCs
	buf = rgx.ReplaceAll(buf, nil)

	rgx, _ = regexp.Compile(`<w:la(|n)g.+?</w:la(|n)g>`) //w:lang
	buf = rgx.ReplaceAll(buf, nil)

	rgx, _ = regexp.Compile(`<w:rFonts.+?</w:rFonts>`) //w:rFonts
	buf = rgx.ReplaceAll(buf, nil)

	// remove any contents from <w:t>...</w:t>
	rgx, _ = regexp.Compile(`<w:t>.+?</w:t>`)
	buf = rgx.ReplaceAll(buf, nil)

	// fmt.Printf("\t\t%s\n\n", buf)
	return string(buf)
}

// Row element means it's available for multiplying
// p, tblRow
func (xnode *xmlNode) isRowElement() bool {
	switch xnode.XMLName.Local {
	case "w-p", "w-tr":
		return true
	}
	return false
}

// HaveParams - does node contents contains any param
func (xnode *xmlNode) HaveParams() bool {
	buf := xnode.Contents()

	// if bytes.Contains(buf, []byte("{{")) && !bytes.Contains(buf, []byte("}}")) {
	// 	log.Printf("ERROR: Broken param: [%s]", string(buf))
	// 	log.Printf("Param node: [%+v]", xnode)
	// }

	have := bytes.Contains(buf, []byte("{{"))        // start
	have = have && bytes.Contains(buf, []byte("}}")) // end
	return have
}

// Does any of child holds contents
// DIFFERENCE: xnode.Contents() returns plaintext concatenated from all childs
// and this function checks every child node separately
func (xnode *xmlNode) AnyChildContains(buf []byte) bool {
	var found bool
	xnode.Walk(func(n *xmlNode) {
		if bytes.Contains(n.Content, buf) {
			found = true
		}
	})
	return found
}

// Show node parents as string chain
// p --> p1 --> p2
func (xnode *xmlNode) parentString(limit int) string {
	s := xnode.XMLName.Local

	n := xnode
	for i := 0; i < limit; i++ {
		if n.parent == nil {
			break
		}
		s = n.parent.XMLName.Local + " . " + s
		n = n.parent
	}

	return s
}

// index of element inside parent.Nodes slice
func (xnode *xmlNode) index() int {
	if xnode.parent != nil {
		for i, n := range xnode.parent.Nodes {
			if xnode == n {
				return i
			}
		}
	}
	return -1
}

// Clone and Add after this
// return new xmlNode
func (xnode *xmlNode) cloneAndAppend() *xmlNode {
	parent := xnode.parent

	// new copy node
	nnew := xnode.clone()
	nnew.isNew = true
	// nnew.Walk(func(nnew *xmlNode) {
	// 	// nnew.Content = bytes.Replace(nnew.Content, []byte("}}"), []byte(" CLONE }}"), -1)
	// })

	// Find node index in parent hierarchy and chose next index as copy place
	i := xnode.index()
	if i == -1 {
		// Return existing instance to avoid nil errors
		// But this node not added to xml structure list, so dissapears in output
		return nnew
	}

	// Insert into specific index
	parent.Nodes = append(parent.Nodes[:i], append([]*xmlNode{nnew}, parent.Nodes[i:]...)...)

	return nnew
}

// Copy node as new and all childs as new too
// no shared addresses as it would be by only copying it
func (xnode *xmlNode) clone() *xmlNode {
	if xnode == nil {
		return nil
	}

	xnodeCopy := &xmlNode{}
	*xnodeCopy = *xnode
	xnodeCopy.Nodes = nil
	xnodeCopy.isNew = true

	for _, n := range xnode.Nodes {
		xnodeCopy.Nodes = append(xnodeCopy.Nodes, n.clone())
	}

	return xnodeCopy
}

// Delete node
func (xnode *xmlNode) delete() {
	// clear contents first
	xnode.Walk(func(nrow *xmlNode) {
		xnode.Content = nil
	})

	// remove from list
	index := xnode.index()
	if index != -1 {
		xnode.parent.Nodes[index] = nil
	}
}

// ReplaceInContents - replace plain text contents with something
func (xnode *xmlNode) ReplaceInContents(old, new []byte) []byte {
	xnode.Walk(func(n *xmlNode) {
		n.Content = bytes.Replace(n.Content, old, new, -1)
	})
	return xnode.Contents()
}

// String get node as string for debugging purposes
// prints useful information
func (xnode *xmlNode) String() string {
	s := fmt.Sprintf("%s: ", xnode.XMLName.Local)
	s += fmt.Sprintf("[%s] == ", xnode.Content)
	s += fmt.Sprintf("[%s]", xnode.Contents())
	s += fmt.Sprintf("\tParent: %s", xnode.parent.XMLName.Local)
	// s += fmt.Sprintf("\t-- %s", xnode.StylesString())
	return s
}
