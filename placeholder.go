package docxplate

type placeholderType int8

const (
	singlePlaceholder placeholderType = iota
	inlinePlaceholder
	rowPlaceholder
)

type placeholder struct {
	Type         placeholderType
	Placeholders []string
	Separator    string
}
