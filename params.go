package docxplate

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
)

// Param ..
type Param struct {
	Key    string
	Value  string
	Params ParamList

	parent      *Param
	AbsoluteKey string
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

// Placeholder ..
func (p *Param) Placeholder() string {
	return "{{" + p.AbsoluteKey + "}}"
}

// PlaceholderKey ..
func (p *Param) PlaceholderKey() string {
	return "{{#" + p.AbsoluteKey + "}}"
}

// PlaceholderMultiple ..
func (p *Param) PlaceholderMultiple() string {
	return "{{#" + p.AbsoluteKey + "."
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

		p2.parent = p
		p2.AbsoluteKey = p.AbsoluteKey + "." + p2.Key
		if p.AbsoluteKey == "" {
			p2.AbsoluteKey = p.Key + "." + p2.Key
		}
		fn(p2)

		p2.Walk(fn)
	}
}

// // Make map of available params from interface{}
// func collectParams(parentPrefix string, v interface{}) map[string]interface{} {
// 	m := map[string]interface{}{}
//
// 	rval := reflect.ValueOf(v)
// 	rind := reflect.Indirect(rval)
// 	rtype := rind.Type()
// 	for i := 0; i < rval.NumField(); i++ {
// 		fval := rind.Field(i)
// 		fname := rtype.Field(i).Name
//
// 		// pointer
// 		if fval.Kind() == reflect.Ptr {
// 			fval = fval.Elem()
// 		}
//
// 		color.Blue("%T -- %-10s %-10s %#v=%+v", fval, fval.Type(), fval.Kind(), fname, fval)
//
// 		// First assign any
// 		m[parentPrefix+fname] = fval.Interface()
// 		if !strings.Contains(fval.Type().String(), "main.") {
// 			// Simple slices (without []main.X or []*main.X ) leave as is
// 			continue
// 		}
//
// 		var m2 map[string]interface{}
//
// 		// modify by specific kind
// 		kind := fval.Kind()
// 		switch kind {
// 		case reflect.Slice:
// 			// m[fname] = color.RedString("-- TODO: %s --", kind.String())
// 			// m2 = collectParams(fname+".", fval[0].Interface())
// 		case reflect.Struct:
// 			m2 = collectParams(fname+".", fval.Interface())
// 		}
//
// 		if len(m2) > 0 {
// 			for k, v := range m2 {
// 				m[k] = v
// 			}
// 			delete(m, fname)
// 		}
//
// 	}
//
// 	return m
// }
