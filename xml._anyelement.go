package docxplate

import "encoding/xml"

type xmlAny interface{}

type xmlElement struct {
	xmlAny // must cast to correct struct before using
}

type xmlElements []*xmlElement

// UnmarshalXML - custom unmarshal to unmarshal/marshal elements in same order
func (x *xmlElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var el interface{}

	// Make sure correct type selected for destination
	switch start.Name.Local {
	case "w-p":
		el = &xmlParagraph{}
	case "w-tbl":
		el = &xmlTable{}
	case "w-sectPr":
		el = &xmlSectionProperties{}
	}

	// xml string to struct
	d.DecodeElement(&el, &start)

	// Save inside of universal tag type
	x.xmlAny = el

	return nil
}

// MarshalXML - custom marshal xmlElement to avoid output <Elements>..</Elements> around node
// WITHOUT: <Elements><w:p>...</w:p></Elements>
// WITH: <w:p>...</w:p> (as supposed to be in docx)
func (x *xmlElement) MarshalXML(e *xml.Encoder, start xml.StartElement) (err error) {
	return e.Encode(x.xmlAny)
}
