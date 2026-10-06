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
	Recipe     model.Recipe `json:"recipe"`
	Status     string       `json:"status"`
	TreeSHA256 string       `json:"tree_sha256,omitempty"`
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
	return filter(entries, query), nil
}

func filter(entries []Entry, query string) []Entry {
	result := []Entry{}
	for _, v := range entries {
		if strings.Contains(strings.ToLower(v.Recipe.ID+" "+v.Recipe.Name), strings.ToLower(query)) {
			result = append(result, v)
		}
	}
	return result
}
func Find(id string) (Entry, error) {
	entries, e := Search("")
	if e != nil {
		return Entry{}, e
	}
	return find(entries, id)
}

func find(entries []Entry, id string) (Entry, error) {
	for _, entry := range entries {
		if entry.Recipe.ID == id {
			return entry, nil
		}
	}
	var matches []Entry
	for _, entry := range entries {
		if strings.SplitN(entry.Recipe.ID, "/", 2)[1] == id {
			matches = append(matches, entry)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return Entry{}, fmt.Errorf("ambiguous theme name; use the complete namespace/name")
	}
	return Entry{}, fmt.Errorf("catalog ID %q not found", id)
}
func DemoFiles() map[string][]byte {
	out := map[string][]byte{}
	for _, p := range []string{"theme.txt", "LICENSE", "hd/theme.txt"} {
		b, _ := resources.ReadFile("demo/" + p)
		out[p] = b
	}
	return out
}
