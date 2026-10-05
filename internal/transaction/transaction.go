package transaction

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"grubmgr/internal/backend"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/planner"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Phases = []string{"planned", "prepared", "assets_staged", "settings_staged", "candidate_validated", "activated", "committed"}

// Faults selects failure points for transaction tests.
type Faults struct {
	FailAfter   string
	CrashAfter  string
	RestoreFail bool
}

func lock(r *os.Root) (func(), error) {
	if e := r.MkdirAll(".grubmgr", 0700); e != nil {
		return nil, e
	}
	f, e := r.OpenFile(".grubmgr/apply.lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return nil, output.Fail(output.Conflict, "LOCKED", "another operation or stale fixture lock exists: %v", e)
	}
	f.Close()
	return func() { _ = r.Remove(".grubmgr/apply.lock") }, nil
}
func checkPaths(p system.Paths) error {
	for _, s := range []string{".grubmgr", backend.Defaults, backend.Config, "boot/grub/themes/grubmgr"} {
		if e := fsx.NoLinks(filepath.Join(p.Root, filepath.FromSlash(s))); e != nil {
			return e
		}
	}
	return nil
}
func Apply(p system.Paths, id string, faults Faults) (state.Transaction, error) {
	var tx state.Transaction
	r, e := system.RequireFixture(p)
	if e != nil {
		return tx, output.Fail(output.Unsupported, "REAL_ACTIVATION_DISABLED", "%v", e)
	}
	defer r.Close()
	if e = checkPaths(p); e != nil {
		return tx, e
	}
	unlock, e := lock(r)
	if e != nil {
		return tx, e
	}
	defer unlock()
	unlockState, e := state.Lock(p)
	if e != nil {
		return tx, output.Fail(output.Conflict, "LOCKED", "%v", e)
	}
	defer unlockState()
	token, e := planner.Decode(id)
	if e != nil {
		return tx, e
	}
	plan, e := planner.Build(p, token.Request)
	if e != nil {
		return tx, e
	}
	if plan.ID != id {
		return tx, output.Fail(output.Conflict, "STALE_PLAN", "root, package, receipt, history or system state changed; replan")
	}
	if !plan.Applicable {
		return tx, output.Fail(output.Unsupported, "BACKEND_REFUSED", "%s", plan.Reason)
	}
	db, e := state.Open(p, true)
	if e != nil {
		return tx, e
	}
	defer db.Close()
	var random [12]byte
	if _, e = rand.Read(random[:]); e != nil {
		return tx, e
	}
	tx = state.Transaction{CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z"), ID: hex.EncodeToString(random[:]), Plan: id, Action: plan.Request.Action, Revision: plan.Revision, Before: plan.Before, ExpectedDefaults: model.Hash([]byte(plan.DefaultsAfter)), ExpectedConfig: model.Hash(plan.Candidate), Destination: strings.TrimPrefix(plan.Destination, "/")}
	phase := func(s string) error {
		tx.Phase = s
		tx.Events = append(tx.Events, s)
		if e := db.Journal(tx); e != nil {
			return e
		}
		if faults.CrashAfter == s {
			return fmt.Errorf("simulated process interruption after %s", s)
		}
		if faults.FailAfter == s {
			return fmt.Errorf("injected failure after %s", s)
		}
		return nil
	}
	run := func() error {
		if e := phase("planned"); e != nil {
			return e
		}
		if e := phase("prepared"); e != nil {
			return e
		}
		if plan.Package != nil && (plan.Request.Action == "install" || plan.Request.Action == "rollback") {
			if e := fsx.NoLinks(filepath.Join(p.Root, filepath.FromSlash(tx.Destination))); e != nil {
				return e
			}
			if _, e := r.Stat(tx.Destination); os.IsNotExist(e) {
				tx.CreatedAssets = true
				if e = db.Journal(tx); e != nil {
					return e
				}
				if e = r.MkdirAll(tx.Destination, 0700); e != nil {
					return e
				}
				if e = fsx.CopyTree(state.Content(p, plan.Revision), r, tx.Destination); e != nil {
					return e
				}
			}
			_, digest, e := fsx.Inventory(filepath.Join(p.Root, filepath.FromSlash(tx.Destination)))
			if e != nil || digest != plan.TreeSHA256 {
				return fmt.Errorf("staged asset verification failed")
			}
		} else if plan.Package != nil && plan.Request.Action == "switch" {
			_, digest, e := fsx.Inventory(filepath.Join(p.Root, filepath.FromSlash(tx.Destination)))
			if e != nil || digest != plan.TreeSHA256 {
				return fmt.Errorf("installed assets differ from immutable receipt")
			}
		}
		if e := phase("assets_staged"); e != nil {
			return e
		}
		liveDefaults, err := fsx.Read(r, backend.Defaults, 1<<20)
		if err != nil || model.Hash(liveDefaults) != model.Hash(tx.Before.Defaults) {
			return fmt.Errorf("defaults drift before settings write")
		}
		if plan.DefaultsAfter != plan.DefaultsBefore {
			if e := fsx.Replace(r, backend.Defaults, []byte(plan.DefaultsAfter)); e != nil {
				return e
			}
		}
		if e := phase("settings_staged"); e != nil {
			return e
		}
		candidatePath := ".grubmgr/candidates/" + tx.ID + ".cfg"
		if e := fsx.Write(r, candidatePath, plan.Candidate); e != nil {
			return e
		}
		candidate, e := fsx.Read(r, candidatePath, 4<<20)
		if e != nil || model.Hash(candidate) != tx.ExpectedConfig {
			return fmt.Errorf("candidate verification failed")
		}
		if e = phase("candidate_validated"); e != nil {
			return e
		}
		liveDefaults, err = fsx.Read(r, backend.Defaults, 1<<20)
		if err != nil || model.Hash(liveDefaults) != tx.ExpectedDefaults {
			return fmt.Errorf("defaults drift before activation")
		}
		live, e := fsx.Read(r, backend.Config, 4<<20)
		if e != nil || model.Hash(live) != model.Hash(tx.Before.Config) {
			return fmt.Errorf("configuration drift before activation")
		}
		if model.Hash(candidate) != model.Hash(live) {
			if e = fsx.Replace(r, backend.Config, candidate); e != nil {
				return e
			}
		}
		if e = phase("activated"); e != nil {
			return e
		}
		packages := append([]model.Package{}, plan.Before.Packages...)
		for i := range packages {
			q := &packages[i]
			if plan.Request.Action == "switch" || plan.Request.Action == "rollback" {
				q.Active = q.Manifest.Revision == plan.Revision
				if q.Active {
					q.Installed = true
					q.Variant = plan.Request.Variant
				}
			}
			if q.Manifest.Revision == plan.Revision {
				switch plan.Request.Action {
				case "install":
					q.Installed = true
					if !q.Active {
						q.Variant = plan.Request.Variant
					}
				case "remove":
					q.Installed = false
					q.Active = false
				}
			}
		}
		if faults.FailAfter == "committed" {
			return fmt.Errorf("injected failure at commit boundary")
		}
		tx.Phase = "committed"
		tx.Events = append(tx.Events, "committed")
		if e = db.Finish(packages, tx); e != nil {
			return e
		}
		_ = r.Remove(candidatePath)
		return nil
	}
	e = run()
	if e == nil {
		return tx, nil
	}
	tx.Error = e.Error()
	if faults.CrashAfter == tx.Phase {
		return tx, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "%v; run recover", e)
	}
	if recoveryErr := restore(r, db, &tx, faults.RestoreFail); recoveryErr != nil {
		return tx, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "%v; recovery: %v", e, recoveryErr)
	}
	return tx, output.Fail(output.IO, "TRANSACTION_ROLLED_BACK", "%v", e)
}
func restore(r *os.Root, db *state.Store, tx *state.Transaction, fail bool) error {
	idBytes, err := hex.DecodeString(tx.ID)
	if err != nil || len(idBytes) != 12 || !model.IsDigest(tx.ExpectedConfig) || !model.IsDigest(tx.ExpectedDefaults) {
		return fmt.Errorf("invalid recovery journal identity or hashes")
	}
	tx.Phase = "recovering"
	tx.Events = append(tx.Events, "recovering")
	if e := db.Journal(*tx); e != nil {
		return e
	}
	recoverErr := func() error {
		if fail {
			return fmt.Errorf("injected restoration failure")
		}
		for _, v := range []struct {
			path     string
			before   []byte
			expected string
		}{{backend.Defaults, tx.Before.Defaults, tx.ExpectedDefaults}, {backend.Config, tx.Before.Config, tx.ExpectedConfig}} {
			b, e := fsx.Read(r, v.path, 4<<20)
			if e != nil {
				return e
			}
			hash := model.Hash(b)
			if hash != model.Hash(v.before) && hash != v.expected {
				return fmt.Errorf("external drift at %s; manual fixture recovery required", v.path)
			}
			if hash != model.Hash(v.before) {
				if e = fsx.Replace(r, v.path, v.before); e != nil {
					return e
				}
			}
			b, e = fsx.Read(r, v.path, 4<<20)
			if e != nil || model.Hash(b) != model.Hash(v.before) {
				return fmt.Errorf("restore verification failed: %s", v.path)
			}
		}
		if tx.CreatedAssets {
			if !strings.HasPrefix(tx.Destination, "boot/grub/themes/grubmgr/") || fsx.SafePath(tx.Destination) != nil {
				return fmt.Errorf("invalid journal destination")
			}
			if e := r.RemoveAll(tx.Destination); e != nil {
				return e
			}
		}
		_ = r.Remove(".grubmgr/candidates/" + tx.ID + ".cfg")
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
func Recover(p system.Paths) ([]state.Transaction, error) {
	r, e := system.RequireFixture(p)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	if e = checkPaths(p); e != nil {
		return nil, e
	}
	unlock, e := lock(r)
	if e != nil {
		return nil, e
	}
	defer unlock()
	unlockState, e := state.Lock(p)
	if e != nil {
		return nil, output.Fail(output.Conflict, "LOCKED", "%v", e)
	}
	defer unlockState()
	db, e := state.Open(p, true)
	if e != nil {
		return nil, e
	}
	defer db.Close()
	history, e := db.History()
	if e != nil {
		return nil, e
	}
	recovered := []state.Transaction{}
	for _, tx := range history {
		if tx.Phase == "committed" || tx.Phase == "rolled_back" {
			continue
		}
		if e = restore(r, db, &tx, false); e != nil {
			return recovered, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "%v", e)
		}
		recovered = append(recovered, tx)
	}
	return recovered, nil
}
