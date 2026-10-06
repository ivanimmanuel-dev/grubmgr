package planner

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"grubmgr/internal/backend"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"grubmgr/internal/validate"
	"os"
	"path"
	"strings"
)

type Request struct {
	Action  string `json:"action"`
	Target  string `json:"target"`
	Variant string `json:"variant"`
}
type Token struct {
	Schema      int     `json:"schema"`
	Request     Request `json:"request"`
	Fingerprint string  `json:"fingerprint"`
}
type Plan struct {
	ID             string         `json:"plan_id"`
	Request        Request        `json:"request"`
	Revision       string         `json:"revision,omitempty"`
	ThemeID        string         `json:"theme_id,omitempty"`
	ArtifactSHA256 string         `json:"artifact_sha256,omitempty"`
	TreeSHA256     string         `json:"tree_sha256,omitempty"`
	Destination    string         `json:"destination,omitempty"`
	Add            []string       `json:"files_to_add"`
	Remove         []string       `json:"files_to_remove"`
	Retained       []string       `json:"retained"`
	BeforeTheme    string         `json:"theme_before"`
	AfterTheme     string         `json:"theme_after"`
	DefaultsBefore string         `json:"defaults_before"`
	DefaultsAfter  string         `json:"defaults_after"`
	Generator      []string       `json:"would_invoke"`
	Privileges     string         `json:"privileges"`
	Validation     string         `json:"validation"`
	Compatibility  string         `json:"compatibility"`
	Recovery       string         `json:"recovery"`
	ThemeBackup    string         `json:"theme_backup,omitempty"`
	Applicable     bool           `json:"applicable"`
	Reason         string         `json:"reason"`
	Fingerprint    string         `json:"fingerprint"`
	Package        *model.Package `json:"-"`
	Before         state.Snapshot `json:"-"`
	Candidate      []byte         `json:"-"`
}

func Decode(id string) (Token, error) {
	var t Token
	if len(id) > 16384 || !strings.HasPrefix(id, "p1.") {
		return t, output.Fail(output.Usage, "PLAN_TOKEN", "expected the complete p1.* token from plan")
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(id, "p1."))
	if e != nil {
		return t, e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&t); e != nil {
		return t, e
	}
	if t.Schema != 1 || !model.IsDigest(t.Fingerprint) {
		return t, fmt.Errorf("invalid plan token")
	}
	return t, nil
}
func Build(p system.Paths, req Request) (Plan, error) {
	plan := Plan{Request: req, Add: []string{}, Remove: []string{}, Retained: []string{}, Privileges: "fixture operations run as the current user", Recovery: "Immediate failure restores journaled bytes; later rollback selects a retained theme against current configuration", Validation: "not applicable", Compatibility: "unverified", Reason: "Fixture plan; a marked synthetic root is required"}
	if req.Variant == "" {
		req.Variant = "default"
		plan.Request = req
	}
	switch req.Action {
	case "install", "switch", "remove", "rollback":
	default:
		return plan, output.Fail(output.Usage, "ACTION", "unknown plan action %q", req.Action)
	}
	sys, e := system.Inspect(p)
	if e != nil {
		return plan, e
	}
	db, e := state.Open(p, false)
	if e != nil {
		return plan, e
	}
	defer db.Close()
	packages, e := db.Packages()
	if e != nil {
		return plan, e
	}
	history, e := db.History()
	if e != nil {
		return plan, e
	}
	for _, t := range history {
		if t.Phase != "committed" && t.Phase != "rolled_back" {
			return plan, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "transaction %s is %s; recover before planning", t.ID, t.Phase)
		}
	}
	r, e := os.OpenRoot(p.Root)
	if e != nil {
		return plan, e
	}
	defer r.Close()
	defaults, e := fsx.Read(r, backend.Defaults, 1<<20)
	if e != nil && !os.IsNotExist(e) {
		return plan, e
	}
	cfg, e := fsx.Read(r, backend.Config, 4<<20)
	if e != nil && !os.IsNotExist(e) {
		return plan, e
	}
	plan.Before = state.Snapshot{Defaults: defaults, Config: cfg, Packages: packages}
	plan.BeforeTheme = sys.Theme
	plan.AfterTheme = sys.Theme
	plan.DefaultsBefore = string(defaults)
	plan.DefaultsAfter = string(defaults)
	var selected model.Package
	have := true
	if req.Action == "rollback" {
		var tx *state.Transaction
		for i := range history {
			if history[i].ID == req.Target {
				tx = &history[i]
			}
		}
		if tx == nil || tx.Phase != "committed" {
			return plan, output.Fail(output.Invalid, "ROLLBACK_TARGET", "committed transaction not found")
		}
		have = false
		plan.AfterTheme = ""
		for _, old := range tx.Before.Packages {
			if old.Active {
				selected, e = state.Select(packages, old.Manifest.Revision)
				if e != nil {
					return plan, e
				}
				req.Variant = old.Variant
				plan.Request.Variant = req.Variant
				have = true
				break
			}
		}
		if !have {
			oldTheme, _ := system.Theme(tx.Before.Defaults)
			if oldTheme != "" {
				return plan, output.Fail(output.Unsupported, "UNMANAGED_ROLLBACK", "prior theme is unmanaged; rollback requires a retained managed revision")
			}
		}
	} else {
		selected, e = state.Select(packages, req.Target)
		if e != nil {
			return plan, output.Fail(output.Invalid, "PACKAGE_SELECTION", "%v", e)
		}
	}
	if have {
		m := selected.Manifest
		if !model.IsDigest(m.Revision) {
			return plan, fmt.Errorf("invalid package revision")
		}
		if req.Action == "install" || req.Action == "switch" || req.Action == "rollback" {
			v := validate.Run(state.Content(p, m.Revision), m)
			if !v.Valid {
				return plan, output.Fail(output.Invalid, "VALIDATION_FAILED", "cached package fails current validation")
			}
			plan.Validation = "passed with compatibility warnings"
			compatible := len(m.Compatibility.Backends) == 0
			for _, b := range m.Compatibility.Backends {
				if b == "fixture-debian" || b == "linux-grub" {
					compatible = true
				}
			}
			if compatible {
				plan.Compatibility = "synthetic Debian fixture only"
			} else {
				plan.Compatibility = "unsupported"
			}
			matches := func(allowed []string, actual string) bool {
				if len(allowed) == 0 {
					return true
				}
				for _, v := range allowed {
					if v == actual {
						return true
					}
				}
				return false
			}
			if !matches(m.Compatibility.Architectures, sys.Architecture) || !matches(m.Compatibility.Firmware, sys.Firmware) {
				plan.Compatibility = "unsupported architecture or firmware"
			}
			if req.Action == "switch" && !selected.Installed {
				return plan, output.Fail(output.Conflict, "NOT_INSTALLED", "install this revision before switching")
			}
		}
		variant, e := m.Variant(req.Variant)
		if e != nil {
			return plan, e
		}
		if fsx.SafePath(variant.Entry) != nil || variant.Root != "." && fsx.SafePath(variant.Root) != nil {
			return plan, fmt.Errorf("invalid variant path")
		}
		plan.Package = &selected
		plan.Revision = m.Revision
		plan.ThemeID = m.ID
		plan.ArtifactSHA256 = m.ArtifactSHA256
		plan.TreeSHA256 = m.TreeSHA256
		plan.Destination = "/boot/grub/themes/grubmgr/" + m.ID + "/" + m.Revision
		if req.Action == "switch" || req.Action == "rollback" {
			plan.AfterTheme = path.Join(plan.Destination, variant.Root, variant.Entry)
		}
		if req.Action == "remove" {
			if !selected.Installed {
				return plan, output.Fail(output.Conflict, "NOT_INSTALLED", "package is not installed")
			}
			if selected.Active {
				plan.AfterTheme = ""
			}
			plan.Retained = append(plan.Retained, plan.Destination+" (retained for rollback)")
		}
		if req.Action == "install" || req.Action == "rollback" {
			if !selected.Installed {
				for _, f := range m.Files {
					plan.Add = append(plan.Add, path.Join(plan.Destination, f.Path))
				}
			}
		}
		if req.Action == "rollback" {
			plan.Retained = append(plan.Retained, "all previous active assets")
		}
	}
	if plan.AfterTheme != plan.BeforeTheme {
		after, err := backend.Settings(defaults, plan.AfterTheme)
		if err != nil {
			return plan, output.Fail(output.Unsupported, "DEFAULTS_COMPLEX", "%v", err)
		}
		plan.DefaultsAfter = string(after)
		plan.Generator = []string{sys.Utilities["grub-mkconfig"], "-o", "<backend-controlled-candidate>"}
	} else {
		plan.Generator = []string{}
	}
	plan.Candidate, e = backend.Candidate(cfg, plan.AfterTheme)
	if e != nil {
		return plan, e
	}
	if req.Action == "install" || plan.AfterTheme == plan.BeforeTheme {
		plan.Candidate = cfg
	}
	plan.Applicable = sys.Status == "fixture-only" && (plan.Compatibility == "synthetic Debian fixture only" || req.Action == "remove" || !have)
	if plan.Applicable {
		plan.Reason = "Applies to the synthetic fixture; the listed GRUB command is informational"
	} else {
		plan.Reason = sys.Reason + "; " + plan.Compatibility
	}
	plan.Fingerprint = model.Digest(struct {
		Root    string
		System  system.Report
		Before  state.Snapshot
		History []state.Transaction
	}{p.Root, sys, plan.Before, history})
	token := Token{1, plan.Request, plan.Fingerprint}
	b, _ := json.Marshal(token)
	plan.ID = "p1." + base64.RawURLEncoding.EncodeToString(b)
	return plan, nil
}
