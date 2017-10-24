package docxplate

import "encoding/xml"

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content []byte     `xml:",chardata"`
	Nodes   []*xmlNode `xml:",any"`

	parent *xmlNode
}

func (xnode *xmlNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// n.Attrs = start.Attr
	type x xmlNode
	return d.DecodeElement((*x)(xnode), &start)
}

// Walk down all nodes and do custom stuff with given function
func (xnode *xmlNode) Walk(fn func(*xmlNode)) {
	for _, n := range xnode.Nodes {

		fn(n) // do your custom stuff

		if len(n.Nodes) > 0 {
			//continue only if have deeper nodes
			n.Walk(fn)
		}
	}
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
