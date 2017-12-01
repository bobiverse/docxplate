package docxplate

import (
	"fmt"
	"strings"
)

// Param ..
type Param struct {
	Key   string
	Value string

	IsSlice bool // mark param created from slice
	Params  ParamList

	parent *Param

	AbsoluteKey string // Users.1.Name
	CompactKey  string // Users.Name
}

// NewParam ..
func NewParam(key interface{}) *Param {
	p := &Param{
		Key: fmt.Sprintf("%v", key),
	}
	p.AbsoluteKey = p.Key
	p.CompactKey = p.Key
	return p
}

// SetValue - any value to string
func (p *Param) SetValue(val interface{}) {
	switch v := val.(type) {
	case string:
		p.Value = v
	default:
		p.Value = fmt.Sprintf("%v", val)
	}

}

// Placeholder .. {{Key}}
func (p *Param) Placeholder() string {
	return "{{" + p.AbsoluteKey + "}}"
}

// PlaceholderKey .. {{#Key}}
func (p *Param) PlaceholderKey() string {
	return "{{#" + p.AbsoluteKey + "}}"
}

// PlaceholderInline .. {{Key ,}}
func (p *Param) PlaceholderInline() string {
	return "{{" + p.AbsoluteKey + " " // "{{Key " - space suffix
}

// PlaceholderKeyInline .. {{#Key ,}}
func (p *Param) PlaceholderKeyInline() string {
	return "{{#" + p.AbsoluteKey + " " // "{{#Key " - space suffix
}

// PlaceholderPrefix .. {{Key.
func (p *Param) PlaceholderPrefix() string {
	return "{{" + p.AbsoluteKey + "." // "{{Key."
}

// PlaceholderKeyPrefix .. {{Key.
func (p *Param) PlaceholderKeyPrefix() string {
	return "{{#" + p.AbsoluteKey + "." // "{{#Key."
}

// ToCompact - convert AbsoluteKey placeholder to ComplexKey placeholder
// {{Users.0.Name}} --> {{Users.Name}}
func (p *Param) ToCompact(placeholder string) string {
	return strings.Replace(placeholder, p.AbsoluteKey, p.CompactKey, 1)
}

// Walk down
func (p *Param) Walk(fn func(*Param)) {
	for _, p2 := range p.Params {
		if p2 == nil {
			continue
		}

		// Assign parent
		p2.parent = p

		// Absolute key with slice indexes
		p2.AbsoluteKey = p.AbsoluteKey + "." + p2.Key
		if p.AbsoluteKey == "" {
			p2.AbsoluteKey = p.Key + "." + p2.Key

		}

		// Complex key with no slice indexes
		if p2.parent.IsSlice {
			p2.CompactKey = p.Key
		} else {
			p2.CompactKey = p.CompactKey + "." + p2.Key
		}

		fn(p2)

		p2.Walk(fn)
	}
}

// Depth - how many levels param have of child nodes
// {{Users.1.Name}} --> 3
func (p *Param) Depth() int {
	return strings.Count(p.AbsoluteKey, ".") + 1
}
