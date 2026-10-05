// Command checklicenses detects unreviewed module/version changes and notice loss.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

type notice struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type component struct {
	Module  string   `json:"module"`
	Version string   `json:"version"`
	License string   `json:"license"`
	Notices []notice `json:"notices"`
}

func main() {
	if e := check(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("Dependency versions and preserved license notice hashes verified")
}
func check() error {
	b, e := os.ReadFile("third_party/dependencies.json")
	if e != nil {
		return e
	}
	var components []component
	if e = json.Unmarshal(b, &components); e != nil {
		return e
	}
	known := map[string]string{}
	for _, c := range components {
		if c.License == "" || len(c.Notices) == 0 {
			return fmt.Errorf("missing license for %s", c.Module)
		}
		known[c.Module] = c.Version
		for _, n := range c.Notices {
			b, e := os.ReadFile(n.Path)
			if e != nil {
				return e
			}
			sum := sha256.Sum256(b)
			if hex.EncodeToString(sum[:]) != n.SHA256 {
				return fmt.Errorf("notice changed: %s", n.Path)
			}
		}
	}
	b, e = os.ReadFile("go.mod")
	if e != nil {
		return e
	}
	re := regexp.MustCompile(`(?m)^\s*(?:require\s+)?([^\s()]+)\s+(v[^\s]+)`)
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		if known[m[1]] != m[2] {
			return fmt.Errorf("unreviewed dependency: %s %s", m[1], m[2])
		}
	}
	return nil
}
