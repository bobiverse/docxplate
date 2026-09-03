package docxplate

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

var imgXMLTpl = "<w:pict><v:shape style='width:%dpt;height:%dpt'><v:imagedata r:id='%s'/></v:shape></w:pict>"

// Relationship ID format for images. The prefix is namespaced on purpose:
// one ID is shared by every part referencing the image, so it must not
// collide with Word's own rIdN in any of them
const imageRelIDFormat = "rIdDocxplateImage%d"

// Process image placeholder - add file, rels and return replace val
func processImage(img *Image) (imgXMLStr string, err error) {
	var imgPath string

	imgPath = img.Path // default
	if img.Path == "" {
		imgPath, err = DefaultDownloader.DownloadFile(context.Background(), img.URL)
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
	var contentTypesNode *xmlNode
	if contentTypesBytes, ok := t.modified[contentTypesName]; ok {
		contentTypesNode = t.bytesToXMLStruct(contentTypesBytes)
	} else {
		contentTypesNode = t.fileToXMLStruct(contentTypesName)
	}
	contentTypesNode.childFirst.iterate(func(node *xmlNode) bool {
		if strings.ToLower(node.Attr("Extension")) == imgExt {
			isContainType = true
			return true
		}
		return false
	})
	if !isContainType {
		contentTypesNode.addSub(&xmlNode{
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

	// The same image value can be used in the document, header, or footer.
	// Each part needs its own relationship file, but the relationship ID can be shared.
	// Images are processed while collecting params, before the file walk in Params(),
	// so the part holding the placeholder is not known yet - register in all of them.
	// Word ignores a relationship nothing references.
	relNames := []string{}
	for filename := range t.files {
		for _, keyword := range modFileNamesLike {
			if strings.Contains(filename, keyword) {
				relNames = append(relNames, path.Dir(filename)+"/_rels/"+path.Base(filename)+".rels")
				break
			}
		}
	}

	// Parse every part once, so the nodes checked for used IDs
	// are the same nodes the new relationship is appended to
	relNodes := map[string]*xmlNode{}
	usedIDs := map[string]bool{}
	for _, relName := range relNames {
		relNodes[relName] = t.imageRelationshipNode(relName)
		relNodes[relName].Walk(func(node *xmlNode) {
			usedIDs[node.Attr("Id")] = true
		})
	}

	rid := ""
	for i := 1; rid == ""; i++ {
		if candidate := fmt.Sprintf(imageRelIDFormat, i); !usedIDs[candidate] {
			rid = candidate
		}
	}

	for relName, relNode := range relNodes {
		relNode.addSub(&xmlNode{
			XMLName: xml.Name{Local: "Relationship"},
			Attrs: []xml.Attr{
				{Name: xml.Name{Local: "Id"}, Value: rid},
				{Name: xml.Name{Local: "Type"}, Value: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"},
				{Name: xml.Name{Local: "Target"}, Value: "media/" + imgPath},
			},
			parent: relNode,
			isNew:  true,
		})

		relBytes := structToXMLBytes(relNode)
		if _, exists := t.files[relName]; exists {
			t.modified[relName] = relBytes
		} else {
			t.added[relName] = relBytes
		}
	}

	// Get replace xml of image
	imgXMLStr = fmt.Sprintf(imgXMLTpl, img.Width, img.Height, rid)

	return
}
