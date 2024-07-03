package docxplate

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/logrusorgru/aurora/v4"
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
	XMLName     xml.Name
	Attrs       []xml.Attr `xml:",any,attr"`
	Content     []byte     `xml:",chardata"`
	childFirst  *xmlNode
	childLast   *xmlNode
	next        *xmlNode
	priv        *xmlNode
	parent      *xmlNode
	childLenght int
	isNew       bool // added recently
}

func (xnode *xmlNode) addSub(n *xmlNode) {
	xnode.childLenght++
	if xnode.childFirst == nil {
		xnode.childFirst = n
		xnode.childLast = n
		return
	}
	xnode.childLast.next = n
	n.priv = xnode.childLast
	xnode.childLast = n

}

func (xnode *xmlNode) add(n *xmlNode) {
	if xnode.parent == nil {
		nn := xnode
		for nn.next != nil {
			nn = nn.next
		}
		nn.next = n
		n.priv = nn
		return
	}
	xnode.parent.addSub(n)
}

func (xnode *xmlNode) iterate(fn func(node *xmlNode) bool) {
	if xnode == nil {
		return
	}
	n := xnode
	if fn(n) {
		return
	}
	for n.next != nil {
		n = n.next
		if fn(n) {
			break
		}
	}
}

func (xnode xmlNode) GetContentPrefixList() (ret []string) {
	var record strings.Builder
	start := false
	length := len(xnode.Content)
	for i, v := range xnode.Content {
		if i == 0 {
			continue
		}

		if v == '{' && xnode.Content[i-1] == '{' {
			start = true
			continue
		}
		if start && (v == ' ' || (v == '}' && length-1 > i && xnode.Content[i+1] == '}')) {
			ret = append(ret, record.String())
			record.Reset()
			start = false
		}
		if start {
			record.WriteByte(v)
		}
	}
	return
}

func (xnode xmlNode) ContentHasPrefix(str string) bool {
	splitContent := bytes.Split(xnode.Content, []byte(str))
	if len(splitContent) == 1 {
		return false
	}
	contentSuffix := splitContent[1]
	return bytes.HasPrefix(contentSuffix, []byte{'.'}) || bytes.HasPrefix(contentSuffix, []byte{'}', '}'}) || bytes.HasPrefix(contentSuffix, []byte{' '})
}

// UnmarshalXML ..
func (xnode *xmlNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// n.Attrs = start.Attr
	xnode.Attrs = start.Attr
	xnode.XMLName = start.Name
	n := xnode
	for {
		token, err := d.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			sub := &xmlNode{
				XMLName: t.Name,
				Attrs:   t.Attr,
				parent:  n,
			}
			n.addSub(sub)
			n = sub
		case xml.EndElement:
			n = n.parent
		case xml.CharData:
			n.Content = t.Copy()
		}
	}
	return nil
}

// MarshalXML ..
func (xnode *xmlNode) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// n.Attrs = start.Attr
	defer e.Close()
	return xnode.XMLEncode(e)

}

func (xnode *xmlNode) XMLEncode(e *xml.Encoder) error {
	err := e.EncodeToken(xml.StartElement{
		Name: xnode.XMLName,
		Attr: xnode.Attrs,
	})
	if err != nil {
		return err
	}
	if len(xnode.Content) != 0 {
		err = e.EncodeToken(xml.CharData(xnode.Content))
		if err != nil {
			return err
		}
	}
	if xnode.childFirst != nil {
		xnode.childFirst.iterate(func(node *xmlNode) bool {
			err = node.XMLEncode(e)
			return err != nil
		})
	}

	if err != nil {
		return err
	}
	return e.EncodeToken(xml.EndElement{
		Name: xnode.XMLName,
	})
}

// Walk down all nodes and do custom stuff with given function
func (xnode *xmlNode) Walk(fn func(*xmlNode)) {
	if xnode.childFirst == nil {
		return
	}
	xnode.childFirst.walk(fn)
}

func (xnode *xmlNode) walk(fn func(*xmlNode)) {
	xnode.iterate(func(node *xmlNode) bool {
		fn(node)
		if node.childFirst != nil {
			node.childFirst.walk(fn)
		}
		return false
	})
}

// fn return true ,end walk
func (xnode *xmlNode) WalkWithEnd(fn func(*xmlNode) bool) {
	if xnode.childFirst == nil {
		return
	}
	xnode.childFirst.walkWithEnd(fn)
}

func (xnode *xmlNode) walkWithEnd(fn func(*xmlNode) bool) {
	xnode.iterate(func(node *xmlNode) bool {
		if (!fn(node)) && node.childFirst != nil {
			node.childFirst.walkWithEnd(fn)
		}
		return false
	})
}

// Walk down all nodes and do custom stuff with given function
func (xnode *xmlNode) WalkTree(depth int, fn func(int, *xmlNode)) {
	if xnode.childFirst == nil {
		return
	}
	xnode.childFirst.walkTree(depth, fn)
}

func (xnode *xmlNode) walkTree(depth int, fn func(int, *xmlNode)) {
	xnode.iterate(func(node *xmlNode) bool {
		fn(depth, xnode)
		if node.childFirst != nil {
			fn(depth+1, node.childFirst)
		}
		return false
	})
}

// Contents - return contents of this and all childs contents merge
func (xnode *xmlNode) AllContents() []byte {
	if xnode == nil {
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

// Single type
// w-t, w-r
func (xnode *xmlNode) isSingle() bool {
	return slices.Contains[[]string](NodeSingleTypes, xnode.XMLName.Local)
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
// func (xnode *xmlNode) index() int {
// 	if xnode != nil && xnode.parent != nil {
// 		for i, n := range xnode.parent.Nodes {
// 			if xnode == n {
// 				return i
// 			}
// 		}
// 	}
// 	return -1
// }

// Clone and Add after this
// return new xmlNode
func (xnode *xmlNode) cloneAndAppend() *xmlNode {
	if xnode == nil {
		return xnode
	}
	// new copy node
	nnew := xnode.clone(xnode.parent) //set parent
	nnew.isNew = true

	tmp := xnode.next
	xnode.next = nnew
	nnew.priv = xnode
	nnew.next = tmp
	if tmp != nil {
		tmp.priv = nnew
	}

	return nnew
}

// Copy node as new and all childs as new too
// no shared addresses as it would be by only copying it
func (xnode *xmlNode) clone(parent *xmlNode) *xmlNode {
	if xnode == nil {
		return nil
	}

	xnodeCopy := &xmlNode{
		XMLName: xnode.XMLName,
		Attrs:   xnode.Attrs,
		Content: xnode.Content,
		isNew:   true,
		parent:  parent,
	}
	if xnode.childFirst != nil {
		xnode.childFirst.iterate(func(node *xmlNode) bool {
			xnodeCopy.addSub(node.clone(xnodeCopy))
			return false
		})
	}

	return xnodeCopy
}

// Delete node
func (xnode *xmlNode) delete() {
	xnode.childLenght = 0
	xnode.childFirst = nil
	xnode.childLast = nil
	if xnode.parent != nil {
		xnode.parent.childLenght--
		if xnode.parent.childFirst == xnode {
			xnode.parent.childFirst = xnode.next
		}
		if xnode.parent.childLast == xnode {
			xnode.parent.childLast = xnode.priv
		}
	}
	if xnode.priv != nil {
		xnode.priv.next = xnode.next
	}
	if xnode.next != nil {
		xnode.next.priv = xnode.priv
	}
}

// Find closest parent way up by node type
func (xnode *xmlNode) closestUp(nodeTypes []string) *xmlNode {
	if xnode.parent == nil {
		return nil
	}
	var n *xmlNode
	for _, ntype := range nodeTypes {

		// aurora.Magenta("[%s] == [%s]", xnode.parent.Tag(), ntype)
		if xnode.parent.Tag() == ntype {
			// aurora.Green("found parent: [%s] == [%s]", xnode.parent.Tag(), ntype)
			return xnode.parent
		}
		xnode.parent.childFirst.iterate(func(node *xmlNode) bool {
			if node.Tag() == ntype {
				n = node
				return true
			}
			return false
		})
		if n != nil {
			return n
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
	var s string
	s += fmt.Sprintf("-- %p -- ", xnode)
	s += fmt.Sprintf("%s: ", xnode.Tag())

	if isListItem, listID := xnode.IsListItem(); isListItem {
		s += fmt.Sprintf("( List:%s ) ", listID)
	}

	s += fmt.Sprintf("[Content:%s]", xnode.Content)
	s += fmt.Sprintf("[%s]", xnode.AllContents())
	s += fmt.Sprintf("\tParent: %s", xnode.parent.Tag())
	// s += fmt.Sprintf("\t-- %s", xnode.StylesString())
	return s
}

// Print tree of node and down
func (xnode *xmlNode) printTree(label string) {
	fmt.Printf("[ %s ]", label)
	fmt.Println("|" + strings.Repeat("-", 80))

	if xnode == nil {
		aurora.Red("Empty node.")
		return
	}
	fmt.Printf("|%s |%p| %s\n", xnode.XMLName.Local, xnode, xnode.Content)

	xnode.WalkTree(0, func(depth int, n *xmlNode) {
		s := "|"
		s += strings.Repeat(" ", depth*4)

		// tag
		s += fmt.Sprintf("%-10s", n.XMLName.Local)
		if xnode.isNew {
			s = aurora.Cyan(s).String()
		}

		// pointers
		s += fmt.Sprintf("|%p|", n)
		sptr := fmt.Sprintf("|%p| ", n.parent)
		if n.parent == nil {
			sptr = aurora.Red(sptr).String()
		}
		s += sptr

		if isListItem, listID := n.IsListItem(); isListItem {
			s += fmt.Sprintf(" (List:%s) ", aurora.Blue(listID))
		}

		if bytes.TrimSpace(n.Content) != nil {
			s += fmt.Sprintf("[%s]", aurora.Yellow(n.Content))
		} else if n.HaveParams() {
			s += aurora.Magenta("<< empty param value >>").String()
		}

		// s += aurora.Cyan(" -- %s", n.StylesString())

		fmt.Println(s)
	})

	fmt.Println("|" + strings.Repeat("-", 80))
}

func (xnode *xmlNode) attrID() string {
	if xnode == nil {
		return ""
	}

	for _, attr := range xnode.Attrs {
		if attr.Name.Local == "id" {
			return attr.Value
		}
	}
	return ""
}

// ^ > w-p > w-pPr > w-numPr > w-numId
func (xnode *xmlNode) nodeBySelector(selector string) *xmlNode {
	selector = strings.TrimSpace(selector)
	selector = strings.ReplaceAll(selector, " ", "")
	tags := strings.Split(selector, ">")
	var n *xmlNode
	for i, tag := range tags {
		xnode.childFirst.iterate(func(node *xmlNode) bool {
			if node.Tag() == tag {
				if len(tags[i:]) == 1 {
					n = node
					return true
				}
				selector = strings.Join(tags[i:], ">")
				// aurora.Green("NEXT: %s", selector)
				n = node.nodeBySelector(selector)
				if n != nil {
					return true
				}
			}
			return false
		})
	}

	// aurora.Red("Selector not found: [%s]", selector)
	return n
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

// Remove xml namespace attr to fix duplication bug in encoding/xml
// issue: https://github.com/golang/go/issues/7535
func (xnode *xmlNode) FixNamespaceDuplication() {
	for i := 0; i < len(xnode.Attrs); i++ {
		if xnode.Attrs[i].Name.Local == "xmlns" {
			xnode.Attrs = append(xnode.Attrs[:i], xnode.Attrs[i+1:]...)
			i--
		}
	}
}
