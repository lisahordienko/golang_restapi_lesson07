package model

import (
	"reflect"
	"testing"
)

func TestBookAssociations(t *testing.T) {
	bookType := reflect.TypeOf(Book{})
	category, ok := bookType.FieldByName("Category")
	if !ok || category.Type != reflect.TypeOf(Category{}) {
		t.Fatal("Book.Category must use the Category model")
	}
	authors, ok := bookType.FieldByName("Authors")
	if !ok || authors.Type != reflect.TypeOf([]Author{}) {
		t.Fatal("Book.Authors must use a slice of Author models")
	}
	if got := authors.Tag.Get("gorm"); got != "many2many:book_authors;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" {
		t.Fatalf("unexpected authors association tag: %q", got)
	}
}
