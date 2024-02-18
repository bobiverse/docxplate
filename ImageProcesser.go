package docxplate

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

var imgXMLTpl = "<w:pict><v:shape style='width:%dpt;height:%dpt'><v:imagedata r:id='%s'/></v:shape></w:pict>"

// Process image placeholder - add file, rels and return replace val
func processImage(img *Image) (imgXMLStr string, err error) {
	var imgPath string

	imgPath = img.Path // default
	if img.Path == "" {
		imgPath, err = downloadFile(img.URL)
		if err != nil {
			return
		}

		defer func() {
			if err := os.Remove(imgPath); err != nil {
				log.Printf("image process: remove: %s", err)
			}
		}()
	}

	// Add image to zip
	imgBytes, err := os.ReadFile(imgPath) // #nosec  G304 - allowed filename as variable here
	if err != nil {
		return
	}
	t.added["word/media/"+imgPath] = imgBytes

	// Add image content type
	var isContainType bool
	imgExt := strings.TrimLeft(strings.ToLower(path.Ext(imgPath)), ".")
	contentTypesName := "[Content_Types].xml"
	contentTypesNode := t.fileToXMLStruct(contentTypesName)
	for _, node := range contentTypesNode.Nodes {
		if strings.ToLower(node.Attr("Extension")) == imgExt {
			isContainType = true
		}
	}
	if !isContainType {
		contentTypesNode.Nodes = append(contentTypesNode.Nodes, &xmlNode{
			XMLName: xml.Name{
				Space: "",
				Local: "Default",
			},
			Attrs: []xml.Attr{
				{Name: xml.Name{Space: "", Local: "Extension"}, Value: imgExt},
				{Name: xml.Name{Space: "", Local: "ContentType"}, Value: "image/" + imgExt},
			},
			parent: contentTypesNode,
			isNew:  true,
		})
		t.modified[contentTypesName] = structToXMLBytes(contentTypesNode)
	}

	// Add image to relations TODO walk all rels
	var relNode *xmlNode
	relName := "word/_rels/document.xml.rels"
	if relNodeBytes, ok := t.modified[relName]; ok {
		relNode = t.bytesToXMLStruct(relNodeBytes)
	} else {
		relNode = t.fileToXMLStruct(relName)
	}
	rid := fmt.Sprintf("rId%d", len(relNode.Nodes)+1)
	relNode.Nodes = append(relNode.Nodes, &xmlNode{
		XMLName: xml.Name{
			Space: "",
			Local: "Relationship",
		},
		Attrs: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "Id"}, Value: rid},
			{Name: xml.Name{Space: "", Local: "Type"}, Value: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"},
			{Name: xml.Name{Space: "", Local: "Target"}, Value: "media/" + imgPath},
		},
		parent: relNode,
		isNew:  true,
	})
	t.modified[relName] = structToXMLBytes(relNode)

	// Get replace xml of image
	imgXMLStr = fmt.Sprintf(imgXMLTpl, img.Width, img.Height, rid)

	return
}
