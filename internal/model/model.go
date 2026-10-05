// Package model defines the data-only package protocol shared by frontends.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}
type Variant struct {
	ID          string   `json:"id"`
	Root        string   `json:"root"`
	Entry       string   `json:"entry"`
	Resolutions []string `json:"resolutions,omitempty"`
}
type License struct {
	SPDX     string   `json:"spdx"`
	Notices  []string `json:"notices"`
	Verified bool     `json:"verified"`
	Evidence string   `json:"evidence,omitempty"`
}
type Source struct {
	Provider         string `json:"provider"`
	URL              string `json:"url"`
	UpstreamRevision string `json:"upstream_revision"`
}
type Compatibility struct {
	Backends      []string `json:"backends"`
	Architectures []string `json:"architectures,omitempty"`
	Firmware      []string `json:"firmware,omitempty"`
	Requires      []string `json:"requires"`
	Notes         []string `json:"notes,omitempty"`
}
type Preview struct {
	Kind        string `json:"kind"`
	Source      string `json:"source,omitempty"`
	Attribution string `json:"attribution,omitempty"`
}
type Recipe struct {
	Schema         int           `json:"schema_version"`
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Author         string        `json:"author"`
	ProjectURL     string        `json:"project_url"`
	Version        string        `json:"version"`
	Source         Source        `json:"source"`
	ArtifactSHA256 string        `json:"artifact_sha256,omitempty"`
	Root           string        `json:"theme_root"`
	Entry          string        `json:"entry"`
	Variants       []Variant     `json:"variants"`
	License        License       `json:"license"`
	Compatibility  Compatibility `json:"compatibility"`
	Preview        Preview       `json:"preview"`
	RecipeRevision string        `json:"recipe_revision"`
	Reviewed       bool          `json:"reviewed"`
}
type Manifest struct {
	Recipe
	Revision   string `json:"revision"`
	TreeSHA256 string `json:"tree_sha256"`
	Files      []File `json:"files"`
}
type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}
type Validation struct {
	Valid     bool      `json:"valid"`
	Findings  []Finding `json:"findings"`
	Validator string    `json:"validator"`
}
type Package struct {
	Manifest   Manifest   `json:"manifest"`
	Validation Validation `json:"validation"`
	Variant    string     `json:"selected_variant"`
	Installed  bool       `json:"installed"`
	Active     bool       `json:"active"`
	Pinned     bool       `json:"pinned"`
}

func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Digest(v any) string  { b, _ := json.Marshal(v); return Hash(b) }

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*/[a-z0-9][a-z0-9.-]*$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func IsDigest(s string) bool { return digestPattern.MatchString(s) }
func (r Recipe) Check() error {
	if r.Schema != 1 || !idPattern.MatchString(r.ID) || r.RecipeRevision == "" || r.Name == "" {
		return fmt.Errorf("schema 1, namespaced lowercase ID, name and recipe_revision required")
	}
	if r.ArtifactSHA256 != "" && !IsDigest(r.ArtifactSHA256) {
		return fmt.Errorf("artifact_sha256 must be a lowercase SHA-256 digest")
	}
	return nil
}
func (m Manifest) Identity() string { m.Revision = ""; return Digest(m) }
func (m Manifest) Variant(id string) (Variant, error) {
	if id == "" || id == "default" {
		return Variant{"default", m.Root, m.Entry, nil}, nil
	}
	for _, v := range m.Variants {
		if v.ID == id {
			return v, nil
		}
	}
	return Variant{}, fmt.Errorf("unknown variant %q", id)
}
