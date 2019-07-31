package docxplate_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"bitbucket.org/briiC/docxplate"
)

// Using this data for all examples below
type User struct {
	Name                   string
	Age                    int
	Nicknames              []string
	Friends                []*User
	BrokenStylePlaceholder string
	TriggerRemove          string
}

// Primite tests with all valid values
// and no param triggers
func TestPlaceholders(t *testing.T) {

	var user = User{
		Name:      "Alice",
		Age:       27,
		Nicknames: []string{"amber", "", "AL", "ice", "", "", "", "", "", "", "", ""},
		Friends: []*User{
			&User{Name: "Bob", Age: 28},
			&User{Name: "Cecilia", Age: 29},
			&User{Name: "", Age: 999},
			&User{Name: "", Age: 999},
			&User{Name: "Den", Age: 30},
			&User{Name: "", Age: 999},
			&User{Name: "Edgar", Age: 31},
			&User{Name: "", Age: 999},
			&User{Name: "", Age: 999},
		},
		BrokenStylePlaceholder: "(NOT ANYMORE)",
	}

	filenames := []string{
		"user.template.docx",
		"tables.docx",
		"lists.docx",
	}

	for _, fname := range filenames {

		inputs := []string{
			"struct",
			"json",
		}

		// Test param setup byu different input types
		for _, inType := range inputs {
			tdoc, _ := docxplate.OpenTemplate("test-data/" + fname)

			plaintext := tdoc.Plaintext()
			params := []string{
				"{{Name}}",
				"{{Friends.1.Name}}",
				"{{Friends.Name :empty",
				"{{Friends.Age}}",
			}
			for _, p := range params {
				if !strings.Contains(plaintext, p) {
					t.Fatalf("Param `%s` should be found in plaintext: \n\n%s", p, plaintext)
				}
			}

			// Run
			switch inType {
			case "struct":
				buf, _ := json.Marshal(user)
				tdoc.Params(buf)
			case "json":
				tdoc.Params(user)
			}

			plaintext = tdoc.Plaintext()
			tdoc.ExportDocx("test-data/~test-" + inType + ".docx")

			// Does non-replacable placeholders still exists
			if !strings.Contains(plaintext, "{{NotReplacable}}") {
				t.Fatalf("Param {{NotReplacable}} be left")
			}
			plaintext = strings.Replace(plaintext, "{{NotReplacable}}", "", -1)

			// All valid params replaced
			leftParams := strings.Contains(plaintext, "{{")
			leftParams = leftParams && strings.Contains(plaintext, "}}")
			if leftParams {
				t.Fatalf("Some params not replaced: \n\n%s", tdoc.Plaintext())
			}

			// Check for "must remove" text
			removedTexts := []string{
				":empty",
				":empty:remove",
				"must be removed",
				"999 y/o",
			}
			for _, s := range removedTexts {
				if strings.Contains(plaintext, s) {
					t.Fatalf("Text `%s` must be removed: \n%s", s, tdoc.Plaintext())
				}
			}

			// Test for know lines
			for _, u := range user.Friends {
				if u.Name == "" {
					continue
				}

				if !strings.Contains(plaintext, u.Name) {
					t.Fatalf("User[%s] friends Name must be found: \n\n%s", u.Name, tdoc.Plaintext())
				}

				years := fmt.Sprintf("%d y/o", u.Age)
				if !strings.Contains(plaintext, years) {
					t.Fatalf("User[%s] friends Age[%d] must be found: \n\n%s", u.Name, u.Age, tdoc.Plaintext())
				}
			}

			// Test empty struct
			// non-empty-trigger placeholders must stay as is
			tdoc, _ = docxplate.OpenTemplate("test-data/" + fname)
			tdoc.Params(struct{ Dummy string }{Dummy: "never"})
			tdoc.ExportDocx("test-data/~test-" + inType + ".docx")
			plaintext = tdoc.Plaintext()
			mandatoryTexts := []string{
				"{{Name}}",
				"{{Friends.1.Name}}",
				"{{NotReplacable}}",
			}
			for _, s := range mandatoryTexts {
				if !strings.Contains(plaintext, s) {
					t.Fatalf("Text [%s must be found: \n\n%s", s, plaintext)
				}
			}
		}
	}

}
