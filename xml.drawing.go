package docxplate

import "encoding/xml"

type xmlDrawing struct {
	*xmlCommon
	XMLName  xml.Name `xml:"w-drawing"`
	InnerXML string   `xml:",innerxml"`
}
