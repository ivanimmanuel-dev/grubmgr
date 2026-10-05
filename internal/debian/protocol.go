// Package debian implements the shared, VM-only Linux GRUB activation service.
package debian

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"grubmgr/internal/catalog"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/planner"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"io"
	"os"
	"sort"
	"strings"
)

const Helper = "/usr/libexec/grubmgr-helper"
const MaxRequest = 180 << 20

// Request contains data and identifiers only. Paths and commands are not protocol fields.
type Request struct {
	Operation string          `json:"operation"`
	Plan      string          `json:"plan,omitempty"`
	Action    planner.Request `json:"action,omitempty"`
	Package   *Bundle         `json:"package,omitempty"`
}
type Bundle struct {
	Manifest model.Manifest    `json:"manifest"`
	Files    map[string][]byte `json:"files"`
}
type Token struct {
	Schema      int             `json:"schema"`
	Request     planner.Request `json:"request"`
	Fingerprint string          `json:"fingerprint"`
	Revision    string          `json:"revision,omitempty"`
}
type Response struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
	Code   int             `json:"code,omitempty"`
}

func ReadRequest(in io.Reader) (Request, error) {
	var req Request
	d := json.NewDecoder(io.LimitReader(in, MaxRequest+1))
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		return req, err
	}
	if d.InputOffset() > MaxRequest {
		return req, fmt.Errorf("request exceeds limit")
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return req, fmt.Errorf("trailing request data")
	}
	switch req.Operation {
	case "inspect", "status", "history", "recover":
		if req.Plan != "" || req.Package != nil || req.Action != (planner.Request{}) {
			return req, fmt.Errorf("unexpected operation fields")
		}
	case "plan":
		if req.Plan != "" {
			return req, fmt.Errorf("plan operation cannot supply a token")
		}
	case "apply":
		if req.Action != (planner.Request{}) {
			return req, fmt.Errorf("apply derives its action from the token")
		}
	default:
		return req, fmt.Errorf("unknown helper operation")
	}
	return req, nil
}

func DecodeToken(id string) (Token, error) {
	var t Token
	if len(id) > 16384 || !strings.HasPrefix(id, "p2.") {
		return t, fmt.Errorf("expected a p2.* plan token")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, "p2."))
	if err != nil {
		return t, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&t); err != nil {
		return t, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF || t.Schema != 2 || !model.IsDigest(t.Fingerprint) || t.Revision != "" && !model.IsDigest(t.Revision) {
		return t, fmt.Errorf("invalid plan token")
	}
	return t, nil
}

func LoadBundle(p system.Paths, target string) (*Bundle, error) {
	db, err := state.Open(p, false)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	packages, err := db.Packages()
	if err != nil {
		return nil, err
	}
	pkg, err := state.Select(packages, target)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(state.Content(p, pkg.Manifest.Revision))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b := &Bundle{Manifest: pkg.Manifest, Files: map[string][]byte{}}
	for _, file := range pkg.Manifest.Files {
		data, err := fsx.Read(r, file.Path, fsx.MaxFile)
		if err != nil {
			return nil, err
		}
		b.Files[file.Path] = data
	}
	return b, CheckBundle(b)
}

// CheckBundle checks a compiled, reviewed recipe as well as every byte. A caller's
// reviewed/license flags alone never grant access to the root package store.
func CheckBundle(b *Bundle) error {
	if b == nil {
		return nil
	}
	m := b.Manifest
	if m.Recipe.Check() != nil || !model.IsDigest(m.Revision) || m.Identity() != m.Revision {
		return fmt.Errorf("invalid package identity")
	}
	entry, err := catalog.Find(m.ID)
	if entry.Recipe.Source.Provider == "builtin" && entry.Recipe.ArtifactSHA256 == "" && m.ArtifactSHA256 == m.TreeSHA256 {
		entry.Recipe.ArtifactSHA256 = m.ArtifactSHA256
	}
	if err != nil || !entry.Recipe.Reviewed || !entry.Recipe.License.Verified || model.Digest(entry.Recipe) != model.Digest(m.Recipe) {
		return fmt.Errorf("package recipe is not approved by this helper build")
	}
	if len(b.Files) == 0 || len(b.Files) > fsx.MaxMembers || len(b.Files) != len(m.Files) {
		return fmt.Errorf("invalid package inventory size")
	}
	files := []model.File{}
	seen := map[string]bool{}
	var size int64
	for name, data := range b.Files {
		if fsx.SafePath(name) != nil || seen[strings.ToLower(name)] || int64(len(data)) > fsx.MaxFile {
			return fmt.Errorf("invalid package member")
		}
		seen[strings.ToLower(name)] = true
		size += int64(len(data))
		if size > fsx.MaxTree {
			return fmt.Errorf("package exceeds limit")
		}
		files = append(files, model.File{Path: name, Size: int64(len(data)), SHA256: model.Hash(data)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	if model.Digest(files) != m.TreeSHA256 || model.Digest(files) != model.Digest(m.Files) {
		return fmt.Errorf("package content digest mismatch")
	}
	if m.Source.Provider == "builtin" {
		original := catalog.DemoFiles()
		if len(original) != len(b.Files) {
			return fmt.Errorf("builtin package content changed")
		}
		for name, data := range original {
			if !bytes.Equal(data, b.Files[name]) {
				return fmt.Errorf("builtin package content changed")
			}
		}
	} else if entry.TreeSHA256 == "" || m.TreeSHA256 != entry.TreeSHA256 {
		return fmt.Errorf("package tree is not pinned by the installed catalogue")
	}
	return nil
}
