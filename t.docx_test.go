package docxplate_test

import (
	"encoding/json"
	"strings"
	"testing"

	"bitbucket.org/briiC/docxplate"
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

func TestParamsReplace(t *testing.T) {
	inputs := []string{
		"struct",
		"json",
	}

	// Test param setup byu different input types
	for _, inType := range inputs {
		tdoc, _ := docxplate.OpenTemplate("test-data/user.template.docx")
		switch inType {
		case "struct":
			buf, _ := json.Marshal(user)
			tdoc.Params(buf)
		case "json":
			tdoc.Params(user)
		}

		tdoc.Bytes()

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

}
