package docxplate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// NodeSingleTypes - NB! sequence is important
var NodeSingleTypes = []string{"w-r", "w-t"}

// NodeCellTypes - NB! sequence is important
var NodeCellTypes = []string{"w-tc"}

// NodeRowTypes - NB! sequence is important
var NodeRowTypes = []string{"w-tr", "w-p"}

// NodeSectionTypes - NB! sequence is important
var NodeSectionTypes = []string{"w-tbl", "w-p"}

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content []byte     `xml:",chardata"`
	Nodes   []*xmlNode `xml:",any"`

	parent    *xmlNode
	isNew     bool // added recently
	isDeleted bool
	haveParam bool // have some kind of param placeholder
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

		if n.Nodes != nil {
			// continue only if have deeper nodes
			n.Walk(fn)
		}
	}
}

// Walk down all nodes and do custom stuff with given function
func (xnode *xmlNode) WalkTree(depth int, fn func(int, *xmlNode)) {
	for _, n := range xnode.Nodes {
		if n == nil {
			continue
		}

		fn(depth+1, n) // do your custom stuff

		if n.Nodes != nil {
			n.WalkTree(depth+1, fn)
		}
	}
}

// Contents - return contents of this and all childs contents merge
func (xnode *xmlNode) AllContents() []byte {
	if xnode == nil || xnode.isDeleted {
		return nil
	}

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
	rgx := regexp.MustCompile(`<w:sz.+?</w:sz(|.+?)>`) // w:sz, w:szCs
	buf = rgx.ReplaceAll(buf, nil)

	rgx = regexp.MustCompile(`<w:la(|n)g.+?</w:la(|n)g>`) // w:lang
	buf = rgx.ReplaceAll(buf, nil)

	rgx = regexp.MustCompile(`<w:rFonts.+?</w:rFonts>`) // w:rFonts
	buf = rgx.ReplaceAll(buf, nil)

	// remove any contents from <w:t>...</w:t>
	rgx = regexp.MustCompile(`<w:t>.+?</w:t>`)
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
	buf := xnode.AllContents()

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

//// Show node parents as string chain
//// p --> p1 --> p2
// func (xnode *xmlNode) parentString(limit int) string {
//	s := xnode.XMLName.Local
//
//	n := xnode
//	for i := 0; i < limit; i++ {
//		if n.parent == nil {
//			break
//		}
//		s = n.parent.XMLName.Local + " . " + s
//		n = n.parent
//	}
//
//	return s
//}

// index of element inside parent.Nodes slice
func (xnode *xmlNode) index() int {
	if xnode != nil && xnode.parent != nil {
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
	nnew := xnode.clone() // parent cleaned
	nnew.isDeleted = false
	nnew.isNew = true

	// Find node index in parent hierarchy and chose next index as copy place
	i := xnode.index()
	if i == -1 {
		// Return existing instance to avoid nil errors
		// But this node not added to xml structure list, so dissapears in output
		return nnew
	}

	// Insert into specific index
	parent.Nodes = append(parent.Nodes[:i], append([]*xmlNode{nnew}, parent.Nodes[i:]...)...)

	// cloned element have incorrect parents - so fixing it here
	nnew.parent.Walk(func(nnew *xmlNode) {
		for _, n := range nnew.Nodes {
			if n != nil {
				n.parent = nnew
			}
		}
	})

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
	xnodeCopy.isDeleted = false
	xnodeCopy.isNew = true

	for _, n := range xnode.Nodes {
		xnodeCopy.Nodes = append(xnodeCopy.Nodes, n.clone())
	}

	return xnodeCopy
}

// Delete node
func (xnode *xmlNode) delete() {
	// xnode.printTree("Delete")

	// remove from list
	index := xnode.index()
	if index != -1 {
		xnode.parent.Nodes[index] = nil
	}
	xnode.isDeleted = true
}

// Find closest parent way up by node type
func (xnode *xmlNode) closestUp(nodeTypes []string) *xmlNode {
	for _, ntype := range nodeTypes {
		if xnode.parent == nil {
			continue
		}
		if xnode.parent.isDeleted {
			continue
		}

		// color.Magenta("[%s] == [%s]", xnode.parent.Tag(), ntype)
		if xnode.parent.Tag() == ntype {
			// color.Green("found parent: [%s] == [%s]", xnode.parent.Tag(), ntype)
			return xnode.parent
		}

		for _, n := range xnode.parent.Nodes {
			if n.Tag() == ntype {
				// color.Green("found parent: [%s] == [%s]", n.Tag(), ntype)
				return n
			}

		}

		if pn := xnode.parent.closestUp([]string{ntype}); pn != nil {
			return pn
		}
	}

	return nil
}

// ReplaceInContents - replace plain text contents with something
func (xnode *xmlNode) ReplaceInContents(old, new []byte) []byte {
	xnode.Walk(func(n *xmlNode) {
		n.Content = bytes.ReplaceAll(n.Content, old, new)
	})
	return xnode.AllContents()
}

// Tag ..
func (xnode *xmlNode) Tag() string {
	if xnode == nil {
		return "(nil)"
	}

	return xnode.XMLName.Local
}

// String get node as string for debugging purposes
// prints useful information
func (xnode *xmlNode) String() string {
	s := fmt.Sprintf("#%d: ", xnode.index())
	if xnode.isDeleted {
		s += color.RedString(" !!DELETED!! ")

	}
	s += fmt.Sprintf("-- %p -- ", xnode)
	s += fmt.Sprintf("%s: ", xnode.Tag())

	if isListItem, listID := xnode.IsListItem(); isListItem {
		s += fmt.Sprintf("( List:%s ) ", listID)
	}

	s += fmt.Sprintf("[Content:%s]", xnode.Content)
	s += fmt.Sprintf(" %3d", len(xnode.Nodes))
	// s += fmt.Sprintf("[%s]", xnode.AllContents())
	s += fmt.Sprintf("\tParent: %s", xnode.parent.Tag())
	// s += fmt.Sprintf("\t-- %s", xnode.StylesString())
	return s
}

//
//// Print tree of node and down
// func (xnode *xmlNode) printTree(label string) {
//	fmt.Printf("[ %s ]", label)
//	fmt.Println("|" + strings.Repeat("-", 80))
//
//	if xnode == nil {
//		color.Red("Empty node.")
//		return
//	}
//	fmt.Printf("|%s |%p| %s\n", xnode.XMLName.Local, xnode, xnode.Content)
//
//	xnode.WalkTree(0, func(depth int, n *xmlNode) {
//		s := "|"
//		s += strings.Repeat(" ", depth*4)
//
//		// tag
//		s += fmt.Sprintf("%-10s", n.XMLName.Local)
//		if xnode.isNew {
//			s = color.CyanString(s)
//		}
//		if xnode.isDeleted {
//			s = color.HiRedString(s)
//		}
//
//		// pointers
//		s += fmt.Sprintf("|%p|", n)
//		sptr := fmt.Sprintf("|%p| ", n.parent)
//		if n.parent == nil {
//			sptr = color.HiRedString(sptr)
//		}
//		s += sptr
//
//		if isListItem, listID := n.IsListItem(); isListItem {
//			s += color.HiBlueString(" (List:%s) ", listID)
//		}
//
//		if bytes.TrimSpace(n.Content) != nil {
//			s += color.YellowString("[%s]", n.Content)
//		} else if n.haveParam {
//			s += color.HiMagentaString("<< empty param value >>")
//		}
//
//		// s += color.CyanString(" -- %s", n.StylesString())
//
//		fmt.Println(s)
//	})
//
//	fmt.Println("|" + strings.Repeat("-", 80))
//}
//
// func (xnode *xmlNode) attrID() string {
//	if xnode == nil {
//		return ""
//	}
//
//	for _, attr := range xnode.Attrs {
//		if attr.Name.Local == "id" {
//			return attr.Value
//		}
//	}
//	return ""
//}

// ^ > w-p > w-pPr > w-numPr > w-numId
func (xnode *xmlNode) nodeBySelector(selector string) *xmlNode {
	selector = strings.TrimSpace(selector)
	selector = strings.ReplaceAll(selector, " ", "")
	tags := strings.Split(selector, ">")

	for i, tag := range tags {
		for _, n := range xnode.Nodes {
			if n.Tag() == tag {
				if len(tags[i:]) == 1 {
					// color.HiGreen("FOUND: %s", tag)
					return n
				}

				selector = strings.Join(tags[i:], ">")
				// color.Green("NEXT: %s", selector)

				return n.nodeBySelector(selector)
			}
		}
	}

	// color.Red("Selector not found: [%s]", selector)
	return nil
}

// get attribute value
func (xnode *xmlNode) Attr(key string) string {
	for _, attr := range xnode.Attrs {
		if attr.Name.Local == key {
			return attr.Value
		}
	}

	return ""
}

// w-p > w-pPr > w-numPr item
func (xnode *xmlNode) IsListItem() (bool, string) {
	if xnode.Tag() != "w-p" {
		return false, ""
	}

	// Quick
	if listID := xnode.Attr("list-id"); listID != "" {
		return true, listID
	}

	// Raw method
	numNode := xnode.nodeBySelector("w-pPr > w-numPr > w-numId")
	if numNode == nil {
		return false, ""
	}

	// Get list ID from attrs
	var listID = numNode.Attr("val")

	return true, listID
}
