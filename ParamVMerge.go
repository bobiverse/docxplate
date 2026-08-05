package docxplate

import (
	"bytes"
	"encoding/xml"
	"slices"
	"strings"
)

// ParamVMerge - placeholder mark to merge cell vertically
// when its table row is multiplied by slice params
// {{Name :vmerge}}
const ParamVMerge = ":vmerge"

// vMerge values for <w:vMerge w:val="..."/>
const (
	vMergeRestart  = "restart"
	vMergeContinue = "continue"
)

// wNamespaceMain - namespace URI of parsed w: prefixed attributes.
// encoding/xml resolves prefixes to URIs, so to be consistent with
// parsed (and re-marshaled) document nodes use the same URI
const wNamespaceMain = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

// isVMergeMark - does raw params part (after param key) contain ":vmerge" mark
func isVMergeMark(raw []byte) bool {
	raw = bytes.TrimSpace(raw)
	raw = bytes.ToLower(raw)

	// Always must start with ":"
	if !bytes.HasPrefix(raw, []byte(":")) {
		return false
	}

	return slices.Contains(strings.Split(string(raw[1:]), ":"), "vmerge")
}

// applyVMerge - mark table cells holding `:vmerge` placeholders
// inside a multiplied (cloned) row.
// First row (rowIndex 0) cell gets <w:vMerge w:val="restart"/> and
// keeps its contents, all the next rows get <w:vMerge w:val="continue"/>
// and cell contents are cleared (merged into the first row cell)
func applyVMerge(nrow *xmlNode, rowIndex int) {

	// collect cells with `:vmerge` placeholders
	var cells []*xmlNode
	seen := map[*xmlNode]bool{}
	nrow.Walk(func(n *xmlNode) {
		if n.Tag() != "w-t" || !bytes.Contains(n.Content, []byte(ParamVMerge)) {
			return
		}
		cell := n.closestUp(NodeCellTypes)
		if cell == nil || seen[cell] {
			return
		}
		seen[cell] = true
		cells = append(cells, cell)
	})

	for _, cell := range cells {

		tcPr := cell.nodeBySelector("w-tcPr")
		if tcPr == nil {
			continue
		}

		val := vMergeRestart
		if rowIndex > 0 {
			val = vMergeContinue
		}
		vmerge := &xmlNode{
			XMLName: xml.Name{Local: "w-vMerge"},
			Attrs: []xml.Attr{{
				Name:  xml.Name{Space: wNamespaceMain, Local: "val"},
				Value: val,
			}},
			isNew: true,
		}

		// w-vMerge goes after w-tcW/w-gridSpan/w-hMerge and before w-tcBorders
		var mark *xmlNode
		tcPr.childFirst.iterate(func(n *xmlNode) bool {
			switch n.Tag() {
			case "w-cnfStyle", "w-tcW", "w-gridSpan", "w-hMerge":
				mark = n
			}
			return false
		})
		insertChildAfter(tcPr, mark, vmerge)

		if rowIndex == 0 {
			continue
		}

		// continuation cell: clear contents, keep paragraph properties
		var runs []*xmlNode
		cell.Walk(func(n *xmlNode) {
			if n.Tag() == "w-r" {
				runs = append(runs, n)
			}
		})
		for _, r := range runs {
			r.delete()
		}
	}
}

// insertChildAfter - insert n into parent's child nodes right after mark.
// If mark is nil, n becomes the first child
func insertChildAfter(parent, mark, n *xmlNode) {
	n.parent = parent

	// as first child
	if mark == nil {
		n.next = parent.childFirst
		if parent.childFirst != nil {
			parent.childFirst.priv = n
		}
		parent.childFirst = n
		if parent.childLast == nil {
			parent.childLast = n
		}
		parent.childLenght++
		return
	}

	n.priv = mark
	n.next = mark.next
	if mark.next != nil {
		mark.next.priv = n
	}
	mark.next = n
	if parent.childLast == mark {
		parent.childLast = n
	}
	parent.childLenght++
}
