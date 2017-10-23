package docxplate

import (
	"encoding/xml"
	"fmt"
)

// AttrList - Every tag in xml can hold attributes
// <tag attrKey=val1 attrKey2=val2 ... >
type AttrList []xml.Attr

type tagAttributes struct {
	AttrList `xml:",any,attr"`
}

// String - all attributes as string for tag
//  attrKey=val1 attrKey2=val2 ...
func (attrs AttrList) String() string {
	s := ""
	for _, attr := range attrs {
		s += fmt.Sprintf(`%s="%s" `, attr.Name.Local, attr.Value)
	}
	return s
}
