package docxplate

import (
	"fmt"
	"testing"
)

func TestAnyToParamsMapStringString(t *testing.T) {

	inParams := map[string]string{
		"Name":     "Alice",
		"Greeting": "Hi!",
	}

	outParams := AnyToParams(inParams)
	if len(inParams) != outParams.Len() {
		t.Fatalf("param count don't match. Expected %d, found %d", len(inParams), outParams.Len())
	}

	// check if all params exists
	for k, _ := range inParams {
		if p := outParams.Get(k); p == nil {
			t.Fatalf("param  `%s` not found", k)
		}

	}

}

func TestAnyToParamsMapStringAny(t *testing.T) {

	type dummyTestType int

	inParams := map[string]any{
		"Name":     "Bob",
		"Age":      uint(28),
		"FavColor": dummyTestType(0xF00),
	}

	outParams := AnyToParams(inParams)
	if len(inParams) != outParams.Len() {
		t.Fatalf("param count don't match. Expected %d, found %d", len(inParams), outParams.Len())
	}

	// check if all params exists
	for k, v := range inParams {
		val := outParams.Get(k)
		if val == nil {
			t.Fatalf("param  `%s` not found", k)
		}
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", val) {
			t.Fatalf("param  `%s` value not equal: [%v]!=[%v]", k, v, val)
		}

	}

}
