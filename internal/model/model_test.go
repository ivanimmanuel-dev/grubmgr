package model

import "testing"

func TestRecipeRejectsControlCharactersInDisplayMetadata(t *testing.T) {
	for _, change := range []func(*Recipe){
		func(r *Recipe) { r.Name = "Theme\x1b[2J" },
		func(r *Recipe) { r.Variants = []Variant{{ID: "hd\nforged output", Root: ".", Entry: "theme.txt"}} },
		func(r *Recipe) { r.Compatibility.Notes = []string{"note\rhidden"} },
	} {
		r := Recipe{Schema: 1, ID: "author/theme", Name: "Theme", Root: ".", Entry: "theme.txt", RecipeRevision: "1"}
		change(&r)
		if r.Check() == nil {
			t.Fatal("terminal control metadata accepted")
		}
	}
}
