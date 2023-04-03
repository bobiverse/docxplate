package docxplate

import (
	"encoding/json"
	"log"
	"reflect"
	"regexp"
)

// ParamList ..
type ParamList []*Param

// StructParams - load params from given any struct
// 1) Convert struct to JSON
// 2) Now convert JSON to map[string]interface{}
// 3) Clear params from nil
func StructParams(v interface{}) ParamList {
	// to JSON output
	buf, _ := json.MarshalIndent(v, "", "\t")
	return JSONToParams(buf)
}

// JSONToParams - load params from JSON
func JSONToParams(buf []byte) ParamList {
	// to map
	m := map[string]interface{}{}
	if err := json.Unmarshal(buf, &m); err != nil {
		log.Printf("JSONToParams: %s", err)
		return nil
	}
	// to filtered/clean map
	params := mapToParams(m)
	params.Walk(func(p *Param) {
		// use Walk func built-in logic to assign keys
	})

	return params
}

// walk map[string]interface{} and collect valid params
func mapToParams(m map[string]interface{}) ParamList {
	var params ParamList
	for mKey, mVal := range m {
		p := NewParam(mKey)

		switch v := mVal.(type) {
		case map[string]interface{}:
			p.Type = StructParam
			p.Params = mapToParams(v)
		case []interface{}:
			p.Type = SliceParam
			p.Params = sliceToParams(v)
		default:
			p.Type = StringParam
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
			p.Type = StructParam
			p.Params = mapToParams(v)
		default:
			p.Type = StringParam
			p.SetValue(v)
		}

		if val == nil && p.Params == nil {
			continue
		}
		params = append(params, p)
	}

	return params
}

// Walk struct and collect valid params
func StructToParams(paramStruct interface{}) ParamList {
	var params ParamList
	var keys reflect.Type
	var vals reflect.Value
	var ok bool

	if vals, ok = paramStruct.(reflect.Value); !ok {
		vals = reflect.ValueOf(paramStruct)
	}
	keys = vals.Type()

	keynum := keys.NumField()
	for i := 0; i < keynum; i++ {
		key := keys.Field(i)
		val := vals.Field(i)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if !val.IsValid() {
			continue
		}

		p := NewParam(key.Name)
		switch val.Kind() {
		case reflect.Struct:
			if image, ok := val.Interface().(Image); ok {
				p.Type = ImageParam
				imgVal, err := processImage(&image)
				if err != nil {
					log.Printf("ProcessImage: %s", err)
					continue
				}
				p.SetValue(imgVal)
			} else {
				p.Type = StructParam
				p.Params = StructToParams(val)
			}
		case reflect.Slice:
			p.Type = SliceParam
			p.Params = reflectSliceToParams(val)
		default:
			p.Type = StringParam
			p.SetValue(val)
		}

		params = append(params, p)
	}

	params.Walk(func(p *Param) {
		// use Walk func built-in logic to assign keys
	})

	return params
}

// reflectSliceToParams - slice of unknown - simple slice or complex
func reflectSliceToParams(slice reflect.Value) ParamList {
	var params ParamList

	for i := 0; i < slice.Len(); i++ {
		// Use index +1 because in template for user not useful see
		// 0 as start number. Only programmers will understand
		p := NewParam(i + 1)

		val := slice.Index(i)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if !val.IsValid() {
			continue
		}

		switch val.Kind() {
		case reflect.Struct:
			if image, ok := val.Interface().(Image); ok {
				p.Type = ImageParam
				imgVal, err := processImage(&image)
				if err != nil {
					log.Printf("ProcessImage: %s", err)
					continue
				}
				p.SetValue(imgVal)
			} else {
				p.Type = StructParam
				p.Params = StructToParams(val)
			}
		default:
			p.Type = StringParam
			p.SetValue(val)
		}

		params = append(params, p)
	}

	return params
}

// Parse row content to param list
func rowParams(row []byte) ParamList {
	// extract from raw contents
	re := regexp.MustCompile(ParamPattern)
	matches := re.FindAllSubmatch(row, -1)
	if matches == nil || matches[0] == nil {
		return nil
	}
	var list []*Param
	for _, match := range matches {
		p := NewParam(string(match[2]))
		p.RowPlaceholder = string(match[0])
		p.Separator = string(match[3])
		p.Trigger = NewParamTrigger(match[4])
		list = append(list, p)
	}
	return list
}

// Walk through params
func (params ParamList) Walk(fn func(*Param)) {
	for _, p := range params {
		fn(p)
		p.Walk(fn)
	}
}
