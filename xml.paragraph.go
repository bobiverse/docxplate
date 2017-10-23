package docxplate

import "encoding/xml"

type xmlParagraph struct {
	*xmlCommon
	XMLName    xml.Name                `xml:"w-p"`
	Properties *xmlParagraphProperties `xml:"w-pPr,omitempty"`
	Records    []*xmlRecord            `xml:"w-r,omitempty"`
}

type xmlParagraphProperties struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-pPr"`
	InnerXML string   `xml:",innerxml"`
}
