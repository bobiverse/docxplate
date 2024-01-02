package docxplate

import "testing"

func TestSingleLeft(t *testing.T) {
	var temp *Template
	singleLeft := "{title}}"
	if !temp.matchSingleLeftPlaceholder(singleLeft) {
		t.Fatalf("Match left fail: %s", singleLeft)
	}
}

func TestSingleRight(t *testing.T) {
	var temp *Template
	singleRight := "{{model}"
	if !temp.matchSingleRightPlaceholder(singleRight) {
		t.Fatalf("Match right fail: %s", singleRight)
	}
}
