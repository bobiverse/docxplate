package docxplate

import "encoding/xml"

type xmlTable struct {
	*xmlCommon
	XMLName    xml.Name            `xml:"w-tbl"`
	Properties *xmlTableProperties `xml:"w-tblPr,omitempty"`
	Grid       *xmlTableGrid       `xml:"w-tblGrid,omitempty"`
	Rows       []*xmlTableRow      `xml:"w-tr,omitempty"`
}

type xmlTableProperties struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-tblPr"`
	InnerXML string   `xml:",innerxml"`
}

type xmlTableGrid struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-tblGrid"`
	InnerXML string   `xml:",innerxml"`
}

// Row
type xmlTableRow struct {
	*xmlCommon
	XMLName    xml.Name               `xml:"w-tr"`
	Properties *xmlTableRowProperties `xml:"w-trPr,omitempty"`
	Columns    []*xmlTableColumn      `xml:"w-tc,omitempty"`
}

type xmlTableRowProperties struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-trPr"`
	InnerXML string   `xml:",innerxml"`
}

// Column
type xmlTableColumn struct {
	*xmlCommon
	XMLName    xml.Name                  `xml:"w-tc"`
	Properties *xmlTableColumnProperties `xml:"w-tcPr,omitempty"`
	Paragraphs []*xmlParagraph           `xml:"w-p,omitempty"`
}

type xmlTableColumnProperties struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-tcPr"`
	InnerXML string   `xml:",innerxml"`
}
