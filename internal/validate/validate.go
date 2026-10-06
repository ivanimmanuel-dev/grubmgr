package validate

import (
	"bufio"
	"bytes"
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/pf2"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path"
	"regexp"
	"strings"
)

var prop = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_-]*)\s*[:=]\s*(?:"([^"\r\n]*)"|([^#\s}]+))\s*(?:#.*)?$`)

func Run(dir string, m model.Manifest) model.Validation {
	v := model.Validation{Valid: true, Validator: "grubmgr/theme-validator-v1", Findings: []model.Finding{}}
	add := func(sev, code, file string, line int, msg string) {
		v.Findings = append(v.Findings, model.Finding{Severity: sev, Code: code, File: file, Line: line, Message: msg})
		if sev == "ERROR" {
			v.Valid = false
		}
	}
	files, tree, e := fsx.Inventory(dir)
	if e != nil {
		add("ERROR", "PACKAGE_PATH", "", 0, e.Error())
		return v
	}
	if m.TreeSHA256 != "" && tree != m.TreeSHA256 {
		add("ERROR", "TREE_CHANGED", "", 0, "Package content differs from its receipt")
	}
	if len(m.Files) > 0 && model.Digest(files) != model.Digest(m.Files) {
		add("ERROR", "INVENTORY_CHANGED", "", 0, "File inventory does not match the package")
	}
	if m.Revision != "" && m.Identity() != m.Revision {
		add("ERROR", "IDENTITY_CHANGED", "", 0, "Manifest differs from immutable revision")
	}
	if e = m.Recipe.Check(); e != nil {
		add("ERROR", "MANIFEST_INVALID", "", 0, e.Error())
	}
	if !m.License.Verified || m.License.SPDX == "" {
		add("WARN", "UNKNOWN_LICENSE", "", 0, "License metadata is unknown; check the author's terms before redistributing")
	}
	r, e := os.OpenRoot(dir)
	if e != nil {
		add("ERROR", "PACKAGE_OPEN", "", 0, e.Error())
		return v
	}
	defer r.Close()
	inventory := map[string]bool{}
	fonts := map[string]bool{"Unifont Regular 16": true}
	for _, f := range files {
		inventory[f.Path] = true
		if handle, err := r.Open(f.Path); err == nil {
			magic := make([]byte, 4)
			n, _ := handle.Read(magic)
			handle.Close()
			magic = magic[:n]
			if bytes.HasPrefix(magic, []byte{0x7f, 'E', 'L', 'F'}) || bytes.HasPrefix(magic, []byte("MZ")) || bytes.HasPrefix(magic, []byte("#!")) {
				add("ERROR", "EXECUTABLE_SIGNATURE", f.Path, 0, "Executable or interpreter signature found in theme data")
			}
		}
		ext := strings.ToLower(path.Ext(f.Path))
		switch ext {
		case ".png", ".jpg", ".jpeg":
			b, e := fsx.Read(r, f.Path, fsx.MaxFile)
			if e != nil {
				add("ERROR", "IMAGE_READ", f.Path, 0, e.Error())
				continue
			}
			cfg, _, e := image.DecodeConfig(bytes.NewReader(b))
			if e != nil {
				add("ERROR", "IMAGE_DECODE", f.Path, 0, e.Error())
				continue
			}
			if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 16777216 {
				add("ERROR", "IMAGE_LIMIT", f.Path, 0, "Image exceeds 16 megapixels")
				continue
			}
			if _, _, e = image.Decode(bytes.NewReader(b)); e != nil {
				add("ERROR", "IMAGE_DECODE", f.Path, 0, e.Error())
			}
		case ".pf2":
			b, e := fsx.Read(r, f.Path, fsx.MaxFile)
			if e != nil {
				add("ERROR", "PF2_INVALID", f.Path, 0, e.Error())
				continue
			}
			name, e := pf2.Inspect(b)
			if e != nil {
				add("ERROR", "PF2_INVALID", f.Path, 0, e.Error())
			} else {
				fonts[name] = true
				add("INFO", "PF2_FONT", f.Path, 0, name)
			}
		case ".sh", ".exe", ".bat", ".cmd", ".ps1", ".py", ".so", ".dll", ".elf", ".efi":
			add("ERROR", "EXECUTABLE_DATA", f.Path, 0, "Executable/script payloads are not accepted as theme data")
		case ".txt", ".md", ".json", "":
		default:
			add("WARN", "UNSUPPORTED_FILE", f.Path, 0, "File type is not supported by the validator")
		}
	}
	for _, notice := range m.License.Notices {
		if fsx.SafePath(notice) != nil || !inventory[notice] {
			add("ERROR", "LICENSE_NOTICE_MISSING", notice, 0, "Declared license notice is absent or unsafe")
		}
	}
	variants := append([]model.Variant{{ID: "default", Root: m.Root, Entry: m.Entry}}, m.Variants...)
	seen := map[string]bool{}
	for _, variant := range variants {
		if seen[variant.ID] || variant.ID == "" {
			add("ERROR", "VARIANT_DUPLICATE", "", 0, "Variants need unique nonempty IDs")
		}
		seen[variant.ID] = true
		if variant.Root != "." && fsx.SafePath(variant.Root) != nil || fsx.SafePath(variant.Entry) != nil {
			add("ERROR", "ENTRY_PATH", variant.Entry, 0, "Unsafe theme root or entry")
			continue
		}
		entry := path.Join(variant.Root, variant.Entry)
		b, e := fsx.Read(r, entry, 1<<20)
		if e != nil {
			add("ERROR", "ENTRY_MISSING", entry, 0, e.Error())
			continue
		}
		depth := 0
		scan := bufio.NewScanner(bytes.NewReader(b))
		line := 0
		for scan.Scan() {
			line++
			s := strings.TrimSpace(scan.Text())
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			if strings.HasPrefix(s, "+") {
				if !strings.HasSuffix(s, "{") {
					add("ERROR", "THEME_SYNTAX", entry, line, "Expected a component declaration ending with {")
				}
				depth++
				continue
			}
			if s == "}" {
				depth--
				if depth < 0 {
					add("ERROR", "THEME_SYNTAX", entry, line, "Unmatched closing brace")
				}
				continue
			}
			a := prop.FindStringSubmatch(s)
			if a == nil {
				add("ERROR", "THEME_SYNTAX", entry, line, "Unsupported or malformed property syntax")
				continue
			}
			key := a[1]
			value := a[2]
			if value == "" {
				value = a[3]
			}
			asset := strings.Contains(key, "image") || strings.Contains(key, "pixmap") || key == "file" || key == "icon" || key == "terminal-box" || strings.HasSuffix(strings.ToLower(value), ".pf2")
			if asset {
				clean := strings.ReplaceAll(value, "*", "part")
				if fsx.SafePath(clean) != nil {
					add("ERROR", "ASSET_ESCAPE", entry, line, "Reference must be a portable relative path: "+value)
					continue
				}
				target := path.Join(path.Dir(entry), value)
				found := false
				for f := range inventory {
					if ok, _ := path.Match(target, f); ok {
						found = true
					}
				}
				if !found {
					add("ERROR", "ASSET_MISSING", entry, line, value)
				}
			} else if strings.Contains(key, "font") && value != "" && !fonts[value] {
				add("ERROR", "FONT_MISSING", entry, line, "Embedded PF2 name not found: "+value)
			}
		}
		if e = scan.Err(); e != nil {
			add("ERROR", "THEME_LIMIT", entry, line, e.Error())
		}
		if depth != 0 {
			add("ERROR", "THEME_SYNTAX", entry, line, "Unbalanced component braces")
		}
	}
	add("WARN", "BOOT_UNVERIFIED", "", 0, "Validation checks theme data; use preview to check rendering")
	if len(m.Compatibility.Backends) == 0 {
		add("WARN", "COMPATIBILITY_UNKNOWN", "", 0, "No tested backend is declared")
	}
	return v
}
func Summary(v model.Validation) string {
	errors := 0
	for _, f := range v.Findings {
		if f.Severity == "ERROR" {
			errors++
		}
	}
	return fmt.Sprintf("%d findings, %d errors", len(v.Findings), errors)
}
