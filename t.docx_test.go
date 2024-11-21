package docxplate_test

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/bobiverse/docxplate"
)

// Using this data for all examples below
type User struct {
	Name                   string
	Age                    int
	Nicknames              []string
	Friends                []*User
	Motto                  string
	BrokenStylePlaceholder string
	TriggerRemove          string
	ImageLocal             *docxplate.Image
	ImageURL               *docxplate.Image
	Images                 []*docxplate.Image
}

// Primite tests with all valid values
// and no param triggers
func TestPlaceholders(t *testing.T) {

	var user = User{
		Name:      "Alice",
		Age:       27,
		Nicknames: []string{"amber", "", "AL", "ice", "", "", "", "", "", "", "", ""},
		Friends: []*User{
			{Name: "Bob", Age: 28, ImageLocal: &docxplate.Image{Path: "images/avatar-4.png", Width: 25, Height: 25}},
			{Name: "Cecilia", Age: 29, ImageLocal: &docxplate.Image{Path: "images/avatar-5.png", Width: 25, Height: 25}},
			{Name: "", Age: 999},
			{Name: "", Age: 999},
			{Name: "Den", Age: 30},
			{Name: "", Age: 999},
			{Name: "Edgar", Age: 31, ImageLocal: &docxplate.Image{Path: "images/avatar-6.png", Width: 25, Height: 25}},
			{Name: "", Age: 999},
			{Name: "", Age: 999},
		},
		BrokenStylePlaceholder: "(NOT ANYMORE)",
		ImageLocal: &docxplate.Image{
			Path:   "images/avatar-1.png",
			Width:  25,
			Height: 25,
		},
		ImageURL: &docxplate.Image{
			URL:    "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png",
			Width:  25,
			Height: 25,
		},
		Images: []*docxplate.Image{
			{
				Path:   "images/avatar-2.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
			{
				Path:   "images/avatar-3.png",
				Width:  25,
				Height: 25,
			},
		},
	}

	filenames := []string{
		"tables.docx",
		"lists.docx",
		"header-footer.docx",
		"user.template-no-images.docx",
		"user.template-with-images.docx",
	}

	for _, fname := range filenames {

		inputs := []string{
			"struct",
			"json",
		}

		// Test param setup byu different input types
		for _, inType := range inputs {
			tdoc, err := docxplate.OpenTemplate("test-data/" + fname)
			if err != nil {
				t.Fatalf("[%s] ERR: %s", fname, err)
			}

			plaintext := tdoc.Plaintext()
			params := []string{
				"{{Name}}",
				"{{Friends.1.Name}}",
				"{{Friends.Name :empty",
				"{{Friends.Age}}",
			}
			for _, p := range params {
				if !strings.Contains(plaintext, p) {
					t.Fatalf("[%s] Param `%s` should be found in plaintext: \n\n%s", fname, p, plaintext)
				}
			}

			// Run
			switch inType {
			case "struct":
				tdoc.Params(user)
			case "json":
				buf, _ := json.Marshal(user)
				tdoc.Params(buf)
			}

			plaintext = tdoc.Plaintext()
			if err := tdoc.ExportDocx("test-data/~test-" + fname + "-" + inType + ".docx"); err != nil {
				t.Fatalf("[%s] ExportDocx: %s", inType, err)
			}

			lines := strings.Split(plaintext, "\n")
			if len(lines) > 900 { // some loops have limit at 1000k iterations, so here is simple check
				t.Fatalf("Plaintext too long: \n%s", tdoc.Plaintext())

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
					t.Fatalf("[%s][%s] User[%s] friends Name must be found: \n\n%s", inType, fname, u.Name, tdoc.Plaintext())
				}

				years := fmt.Sprintf("%d y/o", u.Age)
				if !strings.Contains(plaintext, years) {
					t.Fatalf("[%s][%s] User[%s] friends Age[%d] must be found: \n\n%s", inType, fname, u.Name, u.Age, tdoc.Plaintext())
				}
			}

			// Test empty struct
			// non-empty-trigger placeholders must stay as is
			tdoc, _ = docxplate.OpenTemplate("test-data/" + fname)
			tdoc.Params(struct{ Dummy string }{Dummy: "never"})
			if err := tdoc.ExportDocx("test-data/~test-" + inType + ".docx"); err != nil {
				t.Fatalf("[%s] ExportDocx: %s", inType, err)
			}
			notreplacedPlaceholdersStr := strings.Join(tdoc.Placeholders(), ", ")

			plaintext = tdoc.Plaintext()
			mustBeMissing := []string{
				"{{Name}}",
				"{{Friends.1.Name}}",
				"{{NotReplacable}}",
				"{{NotReplacable , }}",
			}
			for _, s := range mustBeMissing {
				if !strings.Contains(notreplacedPlaceholdersStr, s) {
					t.Fatalf("Placeholder [%s] must be inPlaceholders() slice. Found: %s\n\n%s\n\n", s, notreplacedPlaceholdersStr, plaintext)
				}
				if !strings.Contains(plaintext, s) {
					t.Fatalf("Text [%s] must be found: \n\n%s", s, plaintext)
				}
			}
		}
	}

}

func TestDepthStructToParams(t *testing.T) {
	var user = User{
		Name:  "Alice",
		Age:   27,
		Motto: "Always stay humble.",
		Friends: []*User{
			{Name: "Bob", Age: 28, Motto: "Be\nbrave,\nbe\nbold.", Friends: []*User{
				{Name: "Cecilia", Age: 29, Motto: "Chase\nyour\ndreams."},
				{Name: "Sun (empty motto)", Age: 999, Motto: ""},
				{Name: "Tony (empty motto)", Age: 999, Motto: ""},
			}},
			{Name: "Den", Age: 30, Motto: "Don't give up.", Friends: []*User{
				{Name: "Ben", Age: 999, Motto: "Be brave, be bold as twice as Bob."},
				{Name: "Edgar", Age: 31, Motto: "Embrace the moment."},
				{Name: "Jouny (empty motto)", Age: 999, Motto: ""},
				{Name: "Carrzy", Age: 999, Motto: "Chase your dreamzzz"},
			}},
		},
	}

	tdoc, _ := docxplate.OpenTemplate("test-data/depth.docx")
	tdoc.Params(user)
	if err := tdoc.ExportDocx("test-data/~test-depth.docx"); err != nil {
		log.Fatal(err)
	}
}

func TestWatermark(t *testing.T) {
	var user = User{
		Name: "Walter",
		Friends: []*User{
			{Name: "Bob", Age: 28, ImageLocal: &docxplate.Image{Path: "images/avatar-4.png", Width: 25, Height: 25}},
		},
	}

	tdoc, _ := docxplate.OpenTemplate("test-data/watermark.docx")
	tdoc.Params(user)
	if err := tdoc.ExportDocx("test-data/~test-watermark.docx"); err != nil {
		log.Fatal(err)
	}
}
