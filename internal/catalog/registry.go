package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

const MaxRegistry = 4 << 20

var registryName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Registry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Themes int    `json:"themes"`
}

func decodeRegistry(data []byte) ([]Entry, error) {
	if len(data) > MaxRegistry {
		return nil, fmt.Errorf("catalog exceeds 4 MiB")
	}
	var entries []Entry
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&entries); err != nil {
		return nil, err
	}
	if d.Decode(new(any)) != io.EOF || len(entries) == 0 || len(entries) > 2048 {
		return nil, fmt.Errorf("catalog requires 1–2048 theme records")
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		r := entry.Recipe
		if err := r.Check(); err != nil {
			return nil, err
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("duplicate catalog theme: %s", r.ID)
		}
		seen[r.ID] = true
		if _, err := Find(r.ID); err == nil {
			return nil, fmt.Errorf("catalog cannot replace built-in theme: %s", r.ID)
		}
		u, err := url.Parse(r.Source.URL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || !model.IsDigest(r.ArtifactSHA256) || r.Source.Provider == "builtin" || r.Source.Provider == "debian-installed" {
			return nil, fmt.Errorf("%s requires an HTTPS artifact with a SHA-256 digest", r.ID)
		}
	}
	return entries, nil
}

func Add(config, name string, data []byte) (Registry, error) {
	result := Registry{Name: name, SHA256: model.Hash(data)}
	if !registryName.MatchString(name) || fsx.SafePath(name+".json") != nil {
		return result, fmt.Errorf("catalog name must use lowercase letters, digits and hyphens")
	}
	entries, err := decodeRegistry(data)
	if err != nil {
		return result, err
	}
	result.Themes = len(entries)
	existing, err := List(config)
	if err != nil {
		return result, err
	}
	if len(existing) >= 32 {
		return result, fmt.Errorf("at most 32 catalogs can be configured")
	}
	configured, err := SearchConfigured(config, "")
	if err != nil {
		return result, err
	}
	for _, newEntry := range entries {
		for _, oldEntry := range configured {
			if newEntry.Recipe.ID == oldEntry.Recipe.ID {
				return result, fmt.Errorf("theme already configured: %s", newEntry.Recipe.ID)
			}
		}
	}
	directory := filepath.Join(config, "catalogs")
	if err = fsx.NoLinks(directory); err != nil {
		return result, err
	}
	if err = os.MkdirAll(directory, 0700); err != nil {
		return result, err
	}
	destination := filepath.Join(directory, name+".json")
	if err = fsx.NoLinks(destination); err != nil {
		return result, err
	}
	if _, err = os.Stat(destination); err == nil {
		return result, fmt.Errorf("catalog already exists; remove it before adding a replacement")
	} else if !os.IsNotExist(err) {
		return result, err
	}
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return result, err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		os.Remove(destination)
		return result, err
	}
	return result, closeErr
}

func registries(config string) (map[string][]byte, error) {
	directory := filepath.Join(config, "catalogs")
	if err := fsx.NoLinks(directory); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 32 {
		return nil, fmt.Errorf("at most 32 catalogs can be configured")
	}
	r, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	result := map[string][]byte{}
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		if !registryName.MatchString(name[:len(name)-5]) {
			return nil, fmt.Errorf("invalid catalog filename: %s", name)
		}
		if err := fsx.NoLinks(filepath.Join(directory, name)); err != nil {
			return nil, err
		}
		data, err := fsx.Read(r, name, MaxRegistry)
		if err != nil {
			return nil, err
		}
		result[name[:len(name)-5]] = data
	}
	return result, nil
}

func List(config string) ([]Registry, error) {
	files, err := registries(config)
	if err != nil {
		return nil, err
	}
	result := []Registry{}
	for name, data := range files {
		entries, err := decodeRegistry(data)
		if err != nil {
			return nil, fmt.Errorf("catalog %s: %w", name, err)
		}
		result = append(result, Registry{Name: name, SHA256: model.Hash(data), Themes: len(entries)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func Remove(config, name string) error {
	if !registryName.MatchString(name) || fsx.SafePath(name+".json") != nil {
		return fmt.Errorf("invalid catalog name")
	}
	file := filepath.Join(config, "catalogs", name+".json")
	if err := fsx.NoLinks(file); err != nil {
		return err
	}
	return os.Remove(file)
}

func SearchConfigured(config, query string) ([]Entry, error) {
	entries, err := Search("")
	if err != nil {
		return nil, err
	}
	files, err := registries(config)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Recipe.ID] = true
	}
	for name, data := range files {
		additional, err := decodeRegistry(data)
		if err != nil {
			return nil, fmt.Errorf("catalog %s: %w", name, err)
		}
		for _, entry := range additional {
			if seen[entry.Recipe.ID] {
				return nil, fmt.Errorf("theme appears in multiple catalogs: %s", entry.Recipe.ID)
			}
			seen[entry.Recipe.ID] = true
			entry.Status = name
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Recipe.ID < entries[j].Recipe.ID })
	return filter(entries, query), nil
}

func FindConfigured(config, id string) (Entry, error) {
	entries, err := SearchConfigured(config, "")
	if err != nil {
		return Entry{}, err
	}
	return find(entries, id)
}
