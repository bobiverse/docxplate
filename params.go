package docxplate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
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

// ParamList ..
type ParamList []*Param

type tmpInterface interface{}

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

// ToCompact - convert AbsoluteKey placeholder to ComplexKey placeholder
// {{Users.0.Name}} --> {{Users.Name}}
func (p *Param) ToCompact(placeholder string) string {
	return strings.Replace(placeholder, p.AbsoluteKey, p.CompactKey, 1)
}

// StructParams - load params from given any struct
// 1) Convert struct to JSON
// 2) Now convert JSON to map[string]interface{}
// 3) Clear params from nil
func StructParams(v interface{}) ParamList {
	// to JSON output
	buf, _ := json.MarshalIndent(v, "", "\t")
	color.Magenta("StructParams: %s\n", buf)

	// to map
	m := map[string]interface{}{}
	json.Unmarshal(buf, &m)

	// to filtered/clean map
	params := mapToParams(m)
	params.Walk(func(p *Param) {
		// use Walk func built-in logic to assign keys
	})

	// DEBUG:
	dbg, _ := json.MarshalIndent(params, "", "\t")
	color.Yellow("\n\nParams: %s\n\n", dbg)

	return params
}

// // try any convert to params
// func toParams(v interface{}) ParamList {
// 	switch arr := v.(type) {
// 	case map[string]interface{}:
// 		return mapToParams(arr)
// 	case []interface{}:
// 		return sliceToParams(arr)
// 	}
// 	return nil
// }

// walk map[string]interface{} and collect valid params
func mapToParams(m map[string]interface{}) ParamList {
	var params ParamList
	for mKey, mVal := range m {
		p := NewParam(mKey)

		switch v := mVal.(type) {
		case map[string]interface{}:
			p.Params = mapToParams(v)
		case []interface{}:
			p.IsSlice = true
			p.Params = sliceToParams(v)
		default:
			p.SetValue(mVal)
		}

		if mVal == nil && p.Params == nil {
			continue
		}
		params = append(params, p)

	}

	return params
}

// sliceToParams - slice of unknown - simple slice or complex
func sliceToParams(arr []interface{}) ParamList {
	var params ParamList

	for i, val := range arr {
		// Use index +1 because in template for user not useful see
		// 0 as start number. Only programmers will understand
		p := NewParam(i + 1)

		switch v := val.(type) {
		case map[string]interface{}:
			p.Params = mapToParams(v)
		default:
			p.SetValue(v)
		}

		if val == nil && p.Params == nil {
			continue
		}
		params = append(params, p)
	}

	return params
}

// Walk through params
func (params ParamList) Walk(fn func(*Param)) {
	for _, p := range params {
		fn(p)
		p.Walk(fn)
	}
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
	return strings.Count(p.Placeholder(), ".") + 1
}
