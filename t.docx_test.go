package docxplate_test

import (
	"strings"
	"testing"

	"../docxplate"
)

// Using this data for all examples below
type User struct {
	Name      string
	Age       int
	Nicknames []string
	Friends   []*User
}

var user = User{
	Name:      "Alice",
	Age:       27,
	Nicknames: []string{"amber", "AL", "ice"},
	Friends: []*User{
		&User{Name: "Bob", Age: 28},
		&User{Name: "Cecilia", Age: 29},
		&User{Name: "Den", Age: 30},
	},
}

func TestDocxFull(t *testing.T) {

	tdoc, _ := docxplate.OpenTemplate("test-data/user.template.docx")
	tdoc.Params(user)
	// tdoc.ExportDocx("TEST.docx")

	plaintext := tdoc.Plaintext()

	// Does non-replacable placeholders still exists
	if !strings.Contains(plaintext, "{{NotReplacable}}") {
		t.Errorf("Param {{NotReplacable}} be left")
	}
	plaintext = strings.Replace(plaintext, "{{NotReplacable}}", "", -1)

	// All valid params replaced
	leftParams := strings.Contains(plaintext, "{{")
	leftParams = leftParams && strings.Contains(plaintext, "}}")
	if leftParams {
		t.Errorf("Some params not replaced: \n\n%s", plaintext)
	}

}
