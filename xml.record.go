package docxplate

import "encoding/xml"

type xmlRecord struct {
	*xmlCommon
	XMLName    xml.Name             `xml:"w-r"`
	Properties *xmlRecordProperties `xml:"w-rPr,omitempty"`
	Texts      []*xmlText           `xml:"w-t,omitempty"`
	Drawings   []*xmlDrawing        `xml:"w-drawing,omitempty"`
}
type xmlRecordProperties struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-rPr"`
	InnerXML string   `xml:",innerxml"`
}
