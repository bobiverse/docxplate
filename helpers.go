package docxplate

import (
	"bytes" // #nosec  G501 - allowed weak hash
	"encoding/xml"
	"io"
	"log"
)

func readerBytes(rdr io.ReadCloser) []byte {
	buf := new(bytes.Buffer)

	if rdr == nil {
		log.Printf("can't read bytes from empty reader")
		return nil

	}

	if _, err := buf.ReadFrom(rdr); err != nil {
		log.Printf("can't read bytes: %s", err)
		return nil
	}

	if err := rdr.Close(); err != nil {
		log.Printf("can't close reader: %s", err)
		return nil
	}

	return buf.Bytes()
}

// xmlDeclaration - standard XML declaration for OOXML parts
const xmlDeclaration = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\r\n"

// replaceInXMLTags - replace `from` with `to` byte inside XML tag regions
// (element and attribute names), except within quoted attribute values.
// Comments, CDATA sections, processing instructions and doctype
// are copied verbatim
func replaceInXMLTags(buf []byte, from, to byte) []byte {
	out := make([]byte, 0, len(buf))
	n := len(buf)

	for i := 0; i < n; {
		if buf[i] != '<' {
			out = append(out, buf[i])
			i++
			continue
		}

		// copy comments, CDATA, processing instructions and doctype verbatim
		var closer []byte
		switch {
		case bytes.HasPrefix(buf[i:], []byte("<!--")):
			closer = []byte("-->")
		case bytes.HasPrefix(buf[i:], []byte("<![CDATA[")):
			closer = []byte("]]>")
		case bytes.HasPrefix(buf[i:], []byte("<?")):
			closer = []byte("?>")
		case bytes.HasPrefix(buf[i:], []byte("<!")):
			closer = []byte(">")
		}
		if closer != nil {
			end := bytes.Index(buf[i+2:], closer)
			if end < 0 {
				end = n
			} else {
				end += 2 + len(closer)
			}
			out = append(out, buf[i:i+end]...)
			i += end
			continue
		}

		// inside tag: replace until unquoted '>'
		out = append(out, '<')
		i++
		var quote byte
	tag:
		for i < n {
			c := buf[i]
			if quote != 0 {
				// inside quoted attribute value - copy as is
				if c == quote {
					quote = 0
				}
			} else {
				switch c {
				case '>', '/':
					if c == '>' {
						out = append(out, c)
						i++
						break tag
					}
				case '"', '\'':
					quote = c
				default:
					if c == from {
						c = to
					}
				}
			}
			out = append(out, c)
			i++
		}
	}

	return out
}

// Encode struct to xml code string
func structToXMLBytes(v any) []byte {
	// internal list marker attrs must not leak into output
	if xnode, ok := v.(*xmlNode); ok {
		xnode.Walk(func(n *xmlNode) {
			for i := 0; i < len(n.Attrs); i++ {
				if n.Attrs[i].Name.Local == "list-id" {
					n.Attrs = append(n.Attrs[:i], n.Attrs[i+1:]...)
					i--
				}
			}
		})
	}

	// buf, err := xml.MarshalIndent(v, "", "  ")
	buf, err := xml.Marshal(v)
	if err != nil {
		// fmt.Printf("error: %v\n", err)
		return nil
	}

	// restore original name prefixes encoded in bytesToXMLStruct
	buf = replaceInXMLTags(buf, '-', ':')

	// restore default namespace declaration
	buf = bytes.ReplaceAll(buf, []byte(` xmlns_="`), []byte(` xmlns="`))

	return append([]byte(xmlDeclaration), buf...)
}

// Is slice contains item
func inSlice(a string, slice []string) bool {
	for index := range slice {
		if a == slice[index] {
			return true
		}
	}
	return false
}
