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
	val := vMergeRestart
	if rowIndex > 0 {
		val = vMergeContinue
	}

	for _, cell := range vmergeCells(nrow) {
		setCellVMerge(cell, val)
		if rowIndex > 0 {
			clearCellRuns(cell)
		}
	}
}

// vmergeCells - cells of nrow holding a `:vmerge` placeholder, in order
func vmergeCells(nrow *xmlNode) []*xmlNode {
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
	return cells
}

// setCellVMerge - set cell's <w:vMerge w:val="val"/>, reusing one already
// there (CT_TcPr allows only one) or inserting a new one.
// w-tcPr is optional in a template, but w-vMerge must go in it
func setCellVMerge(cell *xmlNode, val string) {
	tcPr := cell.nodeBySelector("w-tcPr")
	if tcPr == nil {
		tcPr = &xmlNode{
			XMLName: xml.Name{Local: "w-tcPr"},
			isNew:   true,
		}
		cell.insertChildAfter(nil, tcPr)
	}

	valAttr := []xml.Attr{{
		Name:  xml.Name{Local: "w-val"},
		Value: val,
	}}

	if vmerge := tcPr.nodeBySelector("w-vMerge"); vmerge != nil {
		vmerge.Attrs = valAttr
		return
	}

	vmerge := &xmlNode{
		XMLName: xml.Name{Local: "w-vMerge"},
		Attrs:   valAttr,
		isNew:   true,
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
	tcPr.insertChildAfter(mark, vmerge)
}

// clearCellRuns - remove cell's text runs, keeping paragraph properties.
// Used on a continuation cell, whose value merges into the restart cell
func clearCellRuns(cell *xmlNode) {
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
