package docxplate

import "encoding/xml"

// Basic text node
// Text - using as exported to allow this struct to embed as anonymous
// in other structs
type xmlText struct {
	*xmlCommon
	XMLName xml.Name `xml:"w-t,omitempty"`
	Value   string   `xml:",chardata"`
	*tagAttributes
}

// Text - plain text of element
func (x *xmlText) String() string {
	return x.Value
}
