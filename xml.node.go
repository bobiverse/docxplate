package docxplate

import "encoding/xml"

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content []byte     `xml:",chardata"`
	Nodes   []*xmlNode `xml:",any"`

	parent *xmlNode
	isNew  bool // added recently
}

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

func (xnode *xmlNode) Contents() []byte {
	var buf []byte
	xnode.Walk(func(n *xmlNode) {
		buf = append(buf, n.Content...)
	})
	return buf
}

// Row element means it's available for multiplying
// p, tblRow
func (xnode *xmlNode) isRowElement() bool {
	switch xnode.XMLName.Local {
	case "w-p", "w-tblRow":
		return true
	}
	return false
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
func (n *xmlNode) cloneAndAppend() *xmlNode {
	parent := n.parent

	// new copy node
	nnew := n.clone()
	nnew.isNew = true
	nnew.Walk(func(nnew *xmlNode) {
		// nnew.Content = bytes.Replace(nnew.Content, []byte("}}"), []byte(" CLONE }}"), -1)
	})

	// Find node index in parent hierarchy and chose next index as copy place
	i := n.index()
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
	index := xnode.index()
	if index != -1 {
		xnode.parent.Nodes[index] = nil
	}
}
