package catalog

import (
	"encoding/json"
	"grubmgr/internal/model"
	"strings"
	"testing"
)

func registryRecord() Entry {
	return Entry{Recipe: model.Recipe{Schema: 1, ID: "collection/example", Name: "Example",
		Source:         model.Source{Provider: "https", URL: "https://example.com/theme.zip"},
		ArtifactSHA256: strings.Repeat("a", 64), Root: ".", Entry: "theme.txt", RecipeRevision: "1", Reviewed: true}}
}

func TestRegistryLifecycleAndIdentity(t *testing.T) {
	config := t.TempDir()
	entry := registryRecord()
	data, _ := json.Marshal([]Entry{entry})
	result, err := Add(config, "collection", data)
	if err != nil || result.Themes != 1 || result.SHA256 != model.Hash(data) {
		t.Fatal(result, err)
	}
	found, err := FindConfigured(config, "example")
	if err != nil || found.Recipe.ID != entry.Recipe.ID || found.Status != "collection" {
		t.Fatal(found, err)
	}
	if _, err := Add(config, "other", data); err == nil {
		t.Fatal("duplicate theme accepted")
	}
	if err := Remove(config, "collection"); err != nil {
		t.Fatal(err)
	}
	if list, err := List(config); err != nil || len(list) != 0 {
		t.Fatal(list, err)
	}
}

func TestRegistryRejectsUnpinnedAndInvalidSources(t *testing.T) {
	for _, change := range []func(*Entry){
		func(e *Entry) { e.Recipe.ArtifactSHA256 = "" },
		func(e *Entry) { e.Recipe.Source.URL = "http://example.com/theme.zip" },
		func(e *Entry) { e.Recipe.Source.URL = "https://user:password@example.com/theme.zip" },
		func(e *Entry) { e.Recipe.ID = "grubmgr/cyberpunk-demo" },
	} {
		entry := registryRecord()
		change(&entry)
		data, _ := json.Marshal([]Entry{entry})
		if _, err := Add(t.TempDir(), "collection", data); err == nil {
			t.Fatal("invalid registry accepted", entry)
		}
	}
	entry := registryRecord()
	data, _ := json.Marshal([]Entry{entry, entry})
	if _, err := Add(t.TempDir(), "collection", data); err == nil {
		t.Fatal("duplicate entry accepted")
	}
}

func TestRegistryAliasesCannotHideAnotherTheme(t *testing.T) {
	config := t.TempDir()
	entry := registryRecord()
	entry.Recipe.ID = "collection/starfield"
	data, _ := json.Marshal([]Entry{entry})
	if _, err := Add(config, "collection", data); err != nil {
		t.Fatal(err)
	}
	if _, err := FindConfigured(config, "starfield"); err == nil {
		t.Fatal("ambiguous alias guessed")
	}
	if found, err := FindConfigured(config, entry.Recipe.ID); err != nil || found.Recipe.ID != entry.Recipe.ID {
		t.Fatal(found, err)
	}
}
