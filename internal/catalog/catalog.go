package catalog

import (
	"embed"
	"encoding/json"
	"fmt"
	"grubmgr/internal/model"
	"strings"
)

//go:embed catalog.json demo/* demo/hd/*
var resources embed.FS

type Entry struct {
	Recipe model.Recipe `json:"recipe"`
	Status string       `json:"status"`
}

func Search(query string) ([]Entry, error) {
	b, e := resources.ReadFile("catalog.json")
	if e != nil {
		return nil, e
	}
	var entries []Entry
	if e = json.Unmarshal(b, &entries); e != nil {
		return nil, e
	}
	result := []Entry{}
	for _, v := range entries {
		if strings.Contains(strings.ToLower(v.Recipe.ID+" "+v.Recipe.Name), strings.ToLower(query)) {
			result = append(result, v)
		}
	}
	return result, nil
}
func Find(id string) (Entry, error) {
	entries, e := Search("")
	if e != nil {
		return Entry{}, e
	}
	for _, v := range entries {
		if v.Recipe.ID == id || strings.SplitN(v.Recipe.ID, "/", 2)[1] == id {
			return v, nil
		}
	}
	return Entry{}, fmt.Errorf("catalogue ID %q not found", id)
}
func DemoFiles() map[string][]byte {
	out := map[string][]byte{}
	for _, p := range []string{"theme.txt", "LICENSE", "hd/theme.txt"} {
		b, _ := resources.ReadFile("demo/" + p)
		out[p] = b
	}
	return out
}
