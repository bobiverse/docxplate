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
func TestParamsReplace(t *testing.T) {

	var user = User{
		Name:      "Alice",
		Age:       27,
		Nicknames: []string{"amber", "AL", "ice"},
		Friends: []*User{
			&User{Name: "Bob", Age: 28},
			&User{Name: "Cecilia", Age: 29},
			&User{Name: "Den", Age: 30},
			&User{Name: "Edgar", Age: 31},
		},
		BrokenStylePlaceholder: "(NOT ANYMORE)",
	}

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

		plaintext := tdoc.Plaintext()
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
			t.Fatalf("Some params not replaced: \n\n%s", plaintext)
		}

		// Checl for "must remove" text
		isTriggerSuccess := !strings.Contains(plaintext, "must be removed")
		isTriggerSuccess = isTriggerSuccess && !strings.Contains(plaintext, "999 y/o")
		if !isTriggerSuccess {
			t.Fatalf("Some items not removed: \n\n%s", plaintext)
		}

		// Test for know lines
		for _, u := range user.Friends {
			line := fmt.Sprintf("%s\n%d y/o\nis friend to %s", u.Name, u.Age, user.Name)
			if !strings.Contains(plaintext, line) {
				t.Fatalf("User friends info must be found: \n\n%s", plaintext)
			}
		}

		// fmt.Println(plaintext)
	}

}

func TestOnTables(t *testing.T) {
	var user = User{
		Name:      "Alice",
		Age:       27,
		Nicknames: []string{"amber", "", "AL", "ice", "", "", "", "", "", "", "", ""},
		Friends: []*User{
			&User{Name: "", Age: 999},
			&User{Name: "Bob", Age: 28},
			&User{Name: "Cecilia", Age: 29},
			&User{Name: "", Age: 999},
			&User{Name: "Den", Age: 30},
			&User{Name: "", Age: 999},
			&User{Name: "Edgar", Age: 31},
			&User{Name: "", Age: 999},
			&User{Name: "", Age: 999},
		},
		BrokenStylePlaceholder: "(NOT ANYMORE)",
	}

	inputs := []string{
		"struct",
		"json",
	}

	// Test param setup byu different input types
	for _, inType := range inputs {
		// tdoc, _ := docxplate.OpenTemplate("test-data/user.template.docx")
		tdoc, _ := docxplate.OpenTemplate("test-data/tables.docx")
		switch inType {
		case "struct":
			buf, _ := json.Marshal(user)
			tdoc.Params(buf)
		case "json":
			tdoc.Params(user)
		}

		plaintext := tdoc.Plaintext()
		tdoc.ExportDocx("test-data/~test-" + inType + ".docx")

		// All valid params replaced
		leftParams := strings.Contains(plaintext, "{{")
		leftParams = leftParams && strings.Contains(plaintext, "}}")
		if leftParams {
			t.Fatalf("Some params not replaced: \n\n%s", plaintext)
		}

		// Check for "must remove" text
		isTriggerSuccess := !strings.Contains(plaintext, "must be removed")
		isTriggerSuccess = isTriggerSuccess && !strings.Contains(plaintext, "999 y/o")
		if !isTriggerSuccess {
			t.Fatalf("Some items not removed: \n\n%s", plaintext)
		}

		// fmt.Println(plaintext)

	}

}
