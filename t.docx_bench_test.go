package docxplate_test

import (
	"log"
	"testing"

	"github.com/bobiverse/docxplate"
)

func BenchmarkLists100(b *testing.B) {
	var user = User{
		Name: "Walter",
	}
	for i := 0; i < 100; i++ {
		user.Friends = append(user.Friends, &User{Name: "Bob", Age: 28})
	}

	tdoc, _ := docxplate.OpenTemplate("test-data/lists.docx")
	tdoc.Params(user)
	if err := tdoc.ExportDocx("test-data/~test-lists.docx"); err != nil {
		log.Fatal(err)
	}
}

func BenchmarkLists200(b *testing.B) {
	var user = User{
		Name: "Walter",
	}
	for i := 0; i < 200; i++ {
		user.Friends = append(user.Friends, &User{Name: "Bob", Age: 28})
	}

	tdoc, _ := docxplate.OpenTemplate("test-data/lists.docx")
	tdoc.Params(user)
	if err := tdoc.ExportDocx("test-data/~test-lists.docx"); err != nil {
		log.Fatal(err)
	}
}

func BenchmarkLists400(b *testing.B) {
	var user = User{
		Name: "Walter",
	}
	for i := 0; i < 400; i++ {
		user.Friends = append(user.Friends, &User{Name: "Bob", Age: 28})
	}

	tdoc, _ := docxplate.OpenTemplate("test-data/lists.docx")
	tdoc.Params(user)
	if err := tdoc.ExportDocx("test-data/~test-lists.docx"); err != nil {
		log.Fatal(err)
	}
}

func BenchmarkLists1000(b *testing.B) {
	var user = User{
		Name: "Walter",
	}
	for i := 0; i < 1000; i++ {
		user.Friends = append(user.Friends, &User{Name: "Bob", Age: 28})
	}

	tdoc, _ := docxplate.OpenTemplate("test-data/lists.docx")
	tdoc.Params(user)
	if err := tdoc.ExportDocx("test-data/~test-lists.docx"); err != nil {
		log.Fatal(err)
	}
}
