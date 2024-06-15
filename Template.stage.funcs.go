package docxplate

import (
	"bytes"
	"encoding/xml"
	"strings"
)

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

		// node params
		t.params.Walk(func(p *Param) {
			for i, attr := range n.Attrs {
				if strings.Contains(attr.Value, "{{") {
					n.Attrs[i].Value = string(p.replaceIn([]byte(attr.Value)))
				}
			}
		})

		// node contentt
		if bytes.Contains(n.Content, []byte("{{")) {
			// Try to replace on node that contains possible placeholder
			t.params.Walk(func(p *Param) {
				// Only string and image param to replace
				if p.Type != StringParam && p.Type != ImageParam {
					return
				}
				// Prefix check
				if !n.ContentHasPrefix(p.PlaceholderPrefix()) {
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
					t.replaceTextParam(n, p)
				case ImageParam:
					t.replaceImageParams(n, p)
				}
			})
		}
	})
}

// Enhance some markup (removed when building XML in the end)
// so easier to find some element
func (t *Template) enhanceMarkup(xnode *xmlNode) {

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

// new line variable for reuse
var nl = []byte("\n")

// Enhance content
func (t *Template) enhanceContent(xnode *xmlNode) {

	// New lines from text as docx new lines
	xnode.Walk(func(n *xmlNode) {
		if !n.isSingle() {
			return
		}

		if !bytes.Contains(n.Content, nl) {
			return
		}

		nrow := n.closestUp([]string{"w-p"})
		// log.Printf("NEW LINE: %s..%s [%q] %d new lines", aurora.Cyan(nrow.Tag()), aurora.Blue(n.Tag()), aurora.Yellow(n.Content), bytes.Count(n.Content, nl))

		parts := bytes.Split(n.Content, nl)
		for i, buf := range parts {
			// clone the original node to preserve styles and append the cloned node
			nlast := nrow.cloneAndAppend()

			// first and last node can hold other text node prefixes, skip
			if i >= 1 && i <= len(parts) {
				nlast.Walk(func(n2 *xmlNode) {
					if n2.isSingle() && len(n2.Content) > 0 && !bytes.Contains(n.Content, n2.Content) {
						// delete all other text nodes because we need the same text node
						n2.delete()
					}
				})
			}
			nlast.ReplaceInContents(n.Content, buf)
			// nlast.printTree("NROW")
		}

		// delete the original node after cloning and adjusting (otherwise it shows at the end)
		nrow.delete()

	})
}

// This func is fixing broken placeholders by merging "w-t" nodes.
// "w-p" (Record) can hold multiple "w-r". And "w-r" holts "w-t" node
// -
// If these nodes not fixed than params replace can not be done as
// replacer process nodes one by one
func (t *Template) fixBrokenPlaceholders(xnode *xmlNode) {

	xnode.Walk(func(nrow *xmlNode) {
		if !nrow.isRowElement() {
			return
		}

		var brokenNode *xmlNode
		nrow.Walk(func(n *xmlNode) {
			// broken node state? merge next nodes

			if !n.isSingle() && len(n.AllContents()) > 0 {
				// fmt.Printf("\t RESET -- %s->%s [%s]\n", n.parent.Tag(), aurora.Blue(n.Tag()), aurora.Red(n.AllContents()))
				brokenNode = nil
				return
			}

			if brokenNode != nil {
				// fmt.Printf("OK [%s] + [%s]\n", aurora.Green(brokenNode.AllContents()), aurora.Green(n.AllContents()))
				brokenNode.Content = append(brokenNode.Content, n.AllContents()...)
				// aurora.Magenta("[%s] %v -- %v -- %v -- %v", brokenNode.Content, brokenNode.Tag(), brokenNode.parent.Tag(), brokenNode.parent.parent.Tag(), brokenNode.parent.parent.parent.Tag())
				n.Nodes = nil
				n.delete()
				return
			}

			if t.matchBrokenLeftPlaceholder(string(n.Content)) {
				// nrow.printTree("BROKEN")
				brokenNode = n
				return
			}

			brokenNode = nil

		})
	})
}
