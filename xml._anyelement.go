package docxplate

import "encoding/xml"

type xmlAny interface{}

type xmlElement struct {
	xmlAny // must cast to correct struct before using
}

type xmlElements []*xmlElement

// UnmarshalXML - custom unmarshal to unmarshal/marshal elements in same order
func (x *xmlElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	switch start.Name.Local {
	case "w-p":
		var el *xmlParagraph
		d.DecodeElement(&el, &start)
		x.xmlAny = el
	case "w-tbl":
		var el *xmlTable
		d.DecodeElement(&el, &start)
		x.xmlAny = el
	case "w-sectPr":
		var el *xmlSectionProperties
		d.DecodeElement(&el, &start)
		x.xmlAny = el

	}

	return nil
}
