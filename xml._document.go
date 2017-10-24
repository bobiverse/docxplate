package docxplate

import "encoding/xml"

// <w:document ..
type xmlDocument struct {
	*xmlCommon
	XMLName xml.Name `xml:"w-document"`
	Body    *xmlBody `xml:"w-body,omitempty"`
}

// <w:body ..
type xmlBody struct {
	*xmlCommon
	XMLName  xml.Name    `xml:"w-body"`
	Elements xmlElements `xml:",any"`
	// Paragraphs []*xmlParagraph `xml:"w-p,omitempty"`
	// Tables     []*xmlTable     `xml:"w-tbl,omitempty"`

	// Do not care about contents of properties <*Pr>..</*Pr>
	// just save it as string
	SectionProperties *xmlSectionProperties `xml:"w-sectPr,omitempty"`
}

type xmlSectionProperties struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-sectPr"`
	InnerXML string   `xml:",innerxml"`
}

// Walk all document nodes
func (xdoc *xmlDocument) walk() {

}
