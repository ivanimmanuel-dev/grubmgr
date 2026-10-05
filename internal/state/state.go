// Package state persists receipts and journals. Filesystem changes are not SQL-atomic.
package state

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/system"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"sort"
)

type Store struct{ db *sql.DB }

// Lock serializes receipt publication with fixture apply/recover, across processes.
func Lock(p system.Paths) (func(), error) {
	if e := p.Ensure(); e != nil {
		return nil, e
	}
	name := filepath.Join(p.State, "operation.lock")
	f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return nil, fmt.Errorf("state operation lock exists or cannot be created: %w", e)
	}
	if e = f.Close(); e != nil {
		return nil, e
	}
	return func() { _ = os.Remove(name) }, nil
}

type Snapshot struct {
	Defaults []byte          `json:"defaults"`
	Config   []byte          `json:"config"`
	Packages []model.Package `json:"packages"`
}
type Transaction struct {
	CreatedAt        string   `json:"created_at"`
	ID               string   `json:"id"`
	Plan             string   `json:"plan"`
	Action           string   `json:"action"`
	Revision         string   `json:"revision"`
	Phase            string   `json:"phase"`
	Events           []string `json:"events"`
	Before           Snapshot `json:"before"`
	ExpectedDefaults string   `json:"expected_defaults"`
	ExpectedConfig   string   `json:"expected_config"`
	Destination      string   `json:"destination"`
	CreatedAssets    bool     `json:"created_assets"`
	Error            string   `json:"error,omitempty"`
}

func Open(p system.Paths, write bool) (*Store, error) {
	if write {
		if e := p.Ensure(); e != nil {
			return nil, e
		}
	}
	file := filepath.Join(p.State, "state.sqlite")
	if e := fsx.NoLinks(file); e != nil {
		return nil, e
	}
	if !write {
		if _, e := os.Stat(file); os.IsNotExist(e) {
			return &Store{}, nil
		}
	}
	uriPath := filepath.ToSlash(file)
	if filepath.VolumeName(file) != "" {
		uriPath = "/" + uriPath
	}
	u := url.URL{Scheme: "file", Path: uriPath}
	mode := "ro"
	if write {
		mode = "rwc"
	}
	db, e := sql.Open("sqlite", u.String()+"?mode="+mode+"&_pragma=busy_timeout(5000)")
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db}
	if e = db.Ping(); e != nil {
		db.Close()
		return nil, e
	}
	var schema int
	if e = db.QueryRow("PRAGMA user_version").Scan(&schema); e != nil {
		db.Close()
		return nil, e
	}
	if schema > 1 {
		db.Close()
		return nil, fmt.Errorf("database schema %d is newer than supported schema 1", schema)
	}
	if write {
		_, e = db.Exec(`PRAGMA journal_mode=DELETE; PRAGMA synchronous=FULL; CREATE TABLE IF NOT EXISTS packages (revision TEXT PRIMARY KEY, body BLOB NOT NULL); CREATE TABLE IF NOT EXISTS transactions (id TEXT PRIMARY KEY, body BLOB NOT NULL); PRAGMA user_version=1;`)
		if e != nil {
			db.Close()
			return nil, e
		}
	}
	return s, nil
}
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
func (s *Store) Packages() ([]model.Package, error) {
	result := []model.Package{}
	if s.db == nil {
		return result, nil
	}
	rows, e := s.db.Query("SELECT body FROM packages ORDER BY revision")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		var p model.Package
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &p); e != nil {
			return nil, e
		}
		if !model.IsDigest(p.Manifest.Revision) || p.Manifest.Identity() != p.Manifest.Revision || !model.IsDigest(p.Manifest.TreeSHA256) || !model.IsDigest(p.Manifest.ArtifactSHA256) {
			return nil, fmt.Errorf("invalid immutable package receipt")
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func (s *Store) Put(p model.Package) error {
	b, e := json.Marshal(p)
	if e != nil {
		return e
	}
	_, e = s.db.Exec("INSERT INTO packages(revision,body) VALUES(?,?) ON CONFLICT(revision) DO UPDATE SET body=excluded.body", p.Manifest.Revision, b)
	return e
}
func (s *Store) History() ([]Transaction, error) {
	result := []Transaction{}
	if s.db == nil {
		return result, nil
	}
	rows, e := s.db.Query("SELECT body FROM transactions ORDER BY id")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var b []byte
		var t Transaction
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &t); e != nil {
			return nil, e
		}
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt == result[j].CreatedAt {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt < result[j].CreatedAt
	})
	return result, rows.Err()
}
func (s *Store) Journal(t Transaction) error {
	b, e := json.Marshal(t)
	if e != nil {
		return e
	}
	_, e = s.db.Exec("INSERT INTO transactions(id,body) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET body=excluded.body", t.ID, b)
	return e
}

// Finish changes the receipt set and journal atomically inside SQLite only.
func (s *Store) Finish(packages []model.Package, t Transaction) error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, p := range packages {
		b, e := json.Marshal(p)
		if e != nil {
			return e
		}
		if _, e = tx.Exec("INSERT INTO packages(revision,body) VALUES(?,?) ON CONFLICT(revision) DO UPDATE SET body=excluded.body", p.Manifest.Revision, b); e != nil {
			return e
		}
	}
	b, e := json.Marshal(t)
	if e != nil {
		return e
	}
	if _, e = tx.Exec("UPDATE transactions SET body=? WHERE id=?", b, t.ID); e != nil {
		return e
	}
	return tx.Commit()
}
func Select(packages []model.Package, id string) (model.Package, error) {
	var found []model.Package
	for _, p := range packages {
		if p.Manifest.ID == id || p.Manifest.Revision == id || p.Manifest.ID+"@"+p.Manifest.Revision == id {
			found = append(found, p)
		}
	}
	if len(found) != 1 {
		return model.Package{}, fmt.Errorf("expected one fetched revision for %q, found %d; use ID@full-revision", id, len(found))
	}
	return found[0], nil
}
func Content(p system.Paths, revision string) string {
	return filepath.Join(p.Data, "packages", revision, "content")
}
