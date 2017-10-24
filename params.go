package docxplate

import (
	"reflect"
	"strings"

	"github.com/fatih/color"
)

// Make map of available params from interface{}
func collectParams(parentPrefix string, v interface{}) map[string]interface{} {
	m := map[string]interface{}{}

	rval := reflect.ValueOf(v)
	rind := reflect.Indirect(rval)
	rtype := rind.Type()
	for i := 0; i < rval.NumField(); i++ {
		fval := rind.Field(i)
		fname := rtype.Field(i).Name

		// pointer
		if fval.Kind() == reflect.Ptr {
			fval = fval.Elem()
		}

		color.Blue("%T -- %-10s %-10s %#v=%+v", fval, fval.Type(), fval.Kind(), fname, fval)

		// Forst assign any
		m[parentPrefix+fname] = fval.Interface()
		if !strings.Contains(fval.Type().String(), "main.") {
			// Simple slices (without []main.X or []*main.X ) leave as is
			continue
		}

		var m2 map[string]interface{}

		// modify by specific kind
		kind := fval.Kind()
		switch kind {
		case reflect.Slice:
			m[fname] = color.RedString("-- TODO: %s --", kind.String())
			// m2 = collectParams(fname+".", fval)
		case reflect.Struct:
			m2 = collectParams(fname+".", fval.Interface())
		}

		if len(m2) > 0 {
			for k, v := range m2 {
				m[k] = v
			}
			delete(m, fname)
		}

	}

	return m
}
