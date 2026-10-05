package debian

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/state"
	"grubmgr/internal/transaction"
	"grubmgr/internal/validate"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Serve is the helper entrypoint. There is no caller-selectable root or fault mode.
func Serve(in io.Reader, out io.Writer) int {
	var result any
	err := helperIdentity()
	if err == nil {
		var req Request
		req, err = ReadRequest(in)
		if err == nil {
			result, err = execute(req)
		}
	}
	response := Response{}
	if err != nil {
		response.Error = err.Error()
		response.Code = output.IO
		if e, ok := err.(*output.Error); ok {
			response.Code = e.Exit
		}
	} else {
		response.Result, err = json.Marshal(result)
		if err != nil {
			response.Error = err.Error()
			response.Code = output.IO
		}
	}
	_ = json.NewEncoder(out).Encode(response)
	return response.Code
}

func execute(req Request) (any, error) {
	if req.Operation == "inspect" {
		return Inspect()
	}
	env, err := Inspect()
	if err != nil {
		return nil, err
	}
	if env.Status != "SUPPORTED" && env.Status != "SUPPORTED WITH WARNINGS" {
		return nil, output.Fail(output.Unsupported, "BACKEND_REFUSED", "%s", env.Reason)
	}
	if err = CheckBundle(req.Package); err != nil {
		return nil, err
	}
	switch req.Operation {
	case "plan":
		return build("/", req.Action, req.Package)
	case "apply":
		return apply(req, transaction.Faults{})
	case "recover":
		return recoverAll()
	case "status", "history":
		if err := checkStore(); err != nil {
			return nil, err
		}
		db, err := state.Open(paths("/"), false)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		if req.Operation == "history" {
			return db.History()
		}
		return db.Packages()
	}
	return nil, fmt.Errorf("unsupported operation")
}

func rootStore() error {
	if err := secure("/var/lib/grubmgr", true); err != nil {
		return err
	}
	p := paths("/")
	if err := p.Ensure(); err != nil {
		return err
	}
	return checkStore()
}

func checkStore() error {
	p := paths("/")
	if err := secure("/var/lib/grubmgr", true); err != nil {
		return err
	}
	for _, directory := range []string{p.Data, p.Cache, p.State} {
		if _, err := os.Lstat(directory); os.IsNotExist(err) {
			continue
		}
		if err := secure(directory, true); err != nil {
			return err
		}
	}
	if _, err := os.Stat(path.Join(p.State, "state.sqlite")); err == nil {
		return secure(path.Join(p.State, "state.sqlite"), false)
	}
	return nil
}

func stagePackage(r *os.Root, b *Bundle) error {
	if b == nil {
		return nil
	}
	p := paths("/")
	dest := state.Content(p, b.Manifest.Revision)
	if _, err := os.Stat(dest); err == nil {
		_, digest, err := fsx.Inventory(dest)
		if err != nil || digest != b.Manifest.TreeSHA256 {
			return fmt.Errorf("root cache content changed")
		}
		return nil
	}
	parent := filepath.Dir(filepath.Dir(dest))
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".import-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	prefix := strings.TrimPrefix(staging, "/")
	for _, file := range b.Manifest.Files {
		if err := fsx.Write(r, path.Join(prefix, "content", file.Path), b.Files[file.Path]); err != nil {
			return err
		}
	}
	content := filepath.Join(staging, "content")
	if result := validate.Run(content, b.Manifest); !result.Valid {
		return fmt.Errorf("package failed helper validation: %v", result.Findings)
	}
	if err = syncTree(staging); err != nil {
		return err
	}
	if err = os.Rename(staging, filepath.Dir(dest)); err != nil {
		return err
	}
	d, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// A missing destination is allowed only beneath protected, root-owned parents.
func protectedDestination(destination string) error {
	if err := fsx.NoLinks(destination); err != nil {
		return err
	}
	current := destination
	for {
		if _, err := os.Lstat(current); err == nil {
			if err := secure(current, true); err != nil {
				return err
			}
			if current != destination {
				return nil
			}
			return filepath.WalkDir(destination, func(name string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				return secure(name, d.IsDir())
			})
		} else if !os.IsNotExist(err) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("missing destination root")
		}
		current = parent
	}
}

func apply(req Request, faults transaction.Faults) (state.Transaction, error) {
	var tx state.Transaction
	if err := rootStore(); err != nil {
		return tx, err
	}
	unlock, err := acquire("/")
	if err != nil {
		return tx, output.Fail(output.Conflict, "LOCKED", "%v", err)
	}
	defer unlock()
	token, err := DecodeToken(req.Plan)
	if err != nil {
		return tx, err
	}
	pl, err := build("/", token.Request, req.Package)
	if err != nil {
		return tx, err
	}
	if pl.ID != req.Plan {
		return tx, output.Fail(output.Conflict, "STALE_PLAN", "system or receipts changed; create a new plan")
	}
	r, err := os.OpenRoot("/")
	if err != nil {
		return tx, err
	}
	defer r.Close()
	if err = stagePackage(r, req.Package); err != nil {
		return tx, err
	}
	db, err := state.Open(paths("/"), true)
	if err != nil {
		return tx, err
	}
	defer db.Close()
	var nonce [12]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return tx, err
	}
	tx = state.Transaction{ID: hex.EncodeToString(nonce[:]), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Plan: req.Plan, Action: pl.Request.Action, Revision: pl.Revision, Before: pl.Before, ExpectedDefaults: model.Hash([]byte(pl.DefaultsAfter)), ExpectedConfig: model.Hash(pl.Before.Config), Destination: strings.TrimPrefix(pl.Destination, "/")}
	candidate := "boot/grub/.grubmgr-" + tx.ID + ".cfg"
	phase := func(name string) error {
		tx.Phase = name
		tx.Events = append(tx.Events, name)
		if err := db.Journal(tx); err != nil {
			return err
		}
		vmPhaseHook(name)
		if faults.CrashAfter == name {
			return fmt.Errorf("injected interruption at %s", name)
		}
		if faults.FailAfter == name {
			return fmt.Errorf("injected failure at %s", name)
		}
		return nil
	}
	run := func() error {
		if err := phase("planned"); err != nil {
			return err
		}
		if err := phase("prepared"); err != nil {
			return err
		}
		if pl.Package != nil {
			if err := protectedDestination("/" + tx.Destination); err != nil {
				return err
			}
			if err := fsx.NoLinks("/" + tx.Destination); err != nil {
				return err
			}
			if _, err := r.Stat(tx.Destination); os.IsNotExist(err) {
				tx.CreatedAssets = true
				if err = db.Journal(tx); err != nil {
					return err
				}
				if err = r.MkdirAll(tx.Destination, 0700); err != nil {
					return err
				}
				if err = fsx.CopyTree(state.Content(paths("/"), pl.Revision), r, tx.Destination); err != nil {
					return err
				}
				if err = syncTree("/" + tx.Destination); err != nil {
					return err
				}
			}
			if _, digest, err := fsx.Inventory("/" + tx.Destination); err != nil || digest != pl.TreeSHA256 {
				return fmt.Errorf("installed assets failed verification")
			}
		}
		if err := phase("assets_staged"); err != nil {
			return err
		}
		current, err := fsx.Read(r, Defaults, 1<<20)
		if err != nil || model.Hash(current) != model.Hash(tx.Before.Defaults) {
			return fmt.Errorf("defaults changed before write")
		}
		if pl.DefaultsAfter != pl.DefaultsBefore {
			if err = atomicReplace(r, Defaults, []byte(pl.DefaultsAfter)); err != nil {
				return err
			}
		}
		if err = phase("settings_staged"); err != nil {
			return err
		}
		bytes := tx.Before.Config
		if len(pl.Generator) > 0 {
			if err = runTool(pl.Generator[0], "-o", "/"+candidate); err != nil {
				return err
			}
			if err = secure("/"+candidate, false); err != nil {
				return err
			}
			if err = runTool("/usr/bin/grub-script-check", "/"+candidate); err != nil {
				return err
			}
			bytes, err = fsx.Read(r, candidate, 8<<20)
			if err != nil {
				return err
			}
			if !strings.Contains(string(bytes), "menuentry ") || !strings.Contains(string(bytes), "linux") {
				return fmt.Errorf("generated candidate lacks Linux boot entries")
			}
			if pl.AfterTheme != "" && !strings.Contains(string(bytes), strings.TrimPrefix(pl.AfterTheme, "/boot")) {
				return fmt.Errorf("candidate does not reference selected theme")
			}
		}
		tx.ExpectedConfig = model.Hash(bytes)
		if err = phase("candidate_validated"); err != nil {
			return err
		}
		current, err = fsx.Read(r, Defaults, 1<<20)
		if err != nil || model.Hash(current) != tx.ExpectedDefaults {
			return fmt.Errorf("defaults changed before activation")
		}
		current, err = fsx.Read(r, Config, 8<<20)
		if err != nil || model.Hash(current) != model.Hash(tx.Before.Config) {
			return fmt.Errorf("generated configuration changed before activation")
		}
		if model.Hash(bytes) != model.Hash(current) {
			if err = atomicReplace(r, Config, bytes); err != nil {
				return err
			}
		}
		if err = phase("activated"); err != nil {
			return err
		}
		packages := append([]model.Package{}, tx.Before.Packages...)
		for i := range packages {
			pkg := &packages[i]
			if pl.Request.Action == "switch" || pl.Request.Action == "rollback" {
				pkg.Active = pkg.Manifest.Revision == pl.Revision
				if pkg.Active {
					pkg.Installed = true
					pkg.Variant = pl.Request.Variant
				}
			}
			if pkg.Manifest.Revision == pl.Revision {
				pkg.Validation = validate.Run(state.Content(paths("/"), pl.Revision), pkg.Manifest)
				if !pkg.Validation.Valid {
					return fmt.Errorf("package changed before receipt commit")
				}
				switch pl.Request.Action {
				case "install":
					pkg.Installed = true
					if !pkg.Active {
						pkg.Variant = pl.Request.Variant
					}
				case "remove":
					pkg.Installed = false
					pkg.Active = false
				}
			}
		}
		if faults.FailAfter == "committed" {
			return fmt.Errorf("injected receipt commit failure")
		}
		tx.Phase = "committed"
		tx.Events = append(tx.Events, "committed")
		if err = db.Finish(packages, tx); err != nil {
			return err
		}
		_ = r.Remove(candidate)
		return nil
	}
	err = run()
	if err == nil {
		return tx, nil
	}
	tx.Error = err.Error()
	if faults.CrashAfter == tx.Phase {
		return tx, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "%v", err)
	}
	if recoveryErr := restore(r, db, &tx); recoveryErr != nil {
		return tx, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "%v; %v", err, recoveryErr)
	}
	return tx, output.Fail(output.IO, "TRANSACTION_ROLLED_BACK", "%v", err)
}

func restore(r *os.Root, db *state.Store, tx *state.Transaction) error {
	id, err := hex.DecodeString(tx.ID)
	if err != nil || len(id) != 12 || !model.IsDigest(tx.ExpectedDefaults) || !model.IsDigest(tx.ExpectedConfig) {
		return fmt.Errorf("invalid recovery journal")
	}
	tx.Phase = "recovering"
	tx.Events = append(tx.Events, tx.Phase)
	if err = db.Journal(*tx); err != nil {
		return err
	}
	recoverErr := func() error {
		for _, file := range []struct {
			name     string
			before   []byte
			expected string
		}{{Defaults, tx.Before.Defaults, tx.ExpectedDefaults}, {Config, tx.Before.Config, tx.ExpectedConfig}} {
			if err := secure("/"+file.name, false); err != nil {
				return err
			}
			current, err := fsx.Read(r, file.name, 8<<20)
			if err != nil {
				return err
			}
			digest := model.Hash(current)
			if digest != model.Hash(file.before) && digest != file.expected {
				return fmt.Errorf("external edit at /%s; recovery material retained", file.name)
			}
			if digest != model.Hash(file.before) {
				if err = atomicReplace(r, file.name, file.before); err != nil {
					return err
				}
			}
			current, err = fsx.Read(r, file.name, 8<<20)
			if err != nil || model.Hash(current) != model.Hash(file.before) {
				return fmt.Errorf("restoration verification failed")
			}
		}
		// Retain even newly staged assets. Nothing may delete an asset another
		// administrator selected while a transaction was interrupted.
		_ = r.Remove("boot/grub/.grubmgr-" + tx.ID + ".cfg")
		return nil
	}()
	if recoverErr != nil {
		tx.Phase = "recovery_required"
		tx.Error += "; " + recoverErr.Error()
		tx.Events = append(tx.Events, tx.Phase)
		_ = db.Journal(*tx)
		return recoverErr
	}
	tx.Phase = "rolled_back"
	tx.Events = append(tx.Events, tx.Phase)
	return db.Finish(tx.Before.Packages, *tx)
}

func recoverAll() ([]state.Transaction, error) {
	if err := rootStore(); err != nil {
		return nil, err
	}
	unlock, err := acquire("/")
	if err != nil {
		return nil, err
	}
	defer unlock()
	db, err := state.Open(paths("/"), true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	history, err := db.History()
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot("/")
	if err != nil {
		return nil, err
	}
	defer r.Close()
	recovered := []state.Transaction{}
	for _, tx := range history {
		if tx.Phase == "committed" || tx.Phase == "rolled_back" {
			continue
		}
		if err = restore(r, db, &tx); err != nil {
			return recovered, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "%v", err)
		}
		recovered = append(recovered, tx)
	}
	return recovered, nil
}
