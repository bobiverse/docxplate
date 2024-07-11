package docxplate

import (
	"bytes"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Format - how to format an element
const (
	FormatLower      = ":lower"
	FormatUpper      = ":upper"
	FormatTitle      = ":title"
	FormatCapitalize = ":capitalize"
)

type ParamFormatter struct {
	raw string

	Format string
}

// NewFormatter - take raw ":empty:remove:list" and make formatter and its fields from it
func NewFormatter(raw []byte) *ParamFormatter {
	raw = bytes.TrimSpace(raw)
	raw = bytes.ToLower(raw)

	// init with defaults
	f := &ParamFormatter{
		raw: string(raw),
	}

	// Always must start with ":"
	if !strings.HasPrefix(f.raw, ":") {
		return nil
	}

	// Remove the first ":" so split parts counting is more readable
	// Split into parts
	parts := strings.Split(f.raw[1:], ":")

	for _, part := range parts {
		switch part {
		case "lower", "upper", "title", "capitalize":
			f.Format = ":" + part
		}
	}

	return f
}

// applyFormat - apply formatting to the given content based on the formatter
func (p *ParamFormatter) ApplyFormat(format string, content []byte) []byte {
	switch format {
	case FormatLower:
		return bytes.ToLower(content)
	case FormatUpper:
		return bytes.ToUpper(content)
	case FormatTitle:
		titleCaser := cases.Title(language.Und)
		return []byte(titleCaser.String(string(content)))
	case FormatCapitalize:
		content = bytes.TrimSpace(content)
		if len(content) > 0 {
			content[0] = bytes.ToUpper([]byte{content[0]})[0]
			if len(content) > 1 {
				content = append([]byte{content[0]}, bytes.ToLower(content[1:])...)
			}
		}
		return content
	default:
		return content
	}
}

// String - return rebuilt formatter string
func (p *ParamFormatter) String() string {
	if p == nil {
		return ""
	}
	s := p.Format
	return s
}
