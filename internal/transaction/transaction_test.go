package transaction

import (
	"bytes"
	"encoding/json"
	"grubmgr/internal/backend"
	"grubmgr/internal/catalog"
	"grubmgr/internal/fetch"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/planner"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"grubmgr/internal/testutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fetchDemo(t *testing.T, p system.Paths) model.Package {
	t.Helper()
	pkg, e := fetch.Import(p, "cyberpunk-demo", "")
	if e != nil {
		t.Fatal(e)
	}
	return pkg
}
func plan(t *testing.T, p system.Paths, action, target, variant string) planner.Plan {
	t.Helper()
	v, e := planner.Build(p, planner.Request{Action: action, Target: target, Variant: variant})
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func apply(t *testing.T, p system.Paths, action, target, variant string) state.Transaction {
	t.Helper()
	tx, e := Apply(p, plan(t, p, action, target, variant).ID, Faults{})
	if e != nil {
		t.Fatal(e)
	}
	return tx
}
func packages(t *testing.T, p system.Paths) []model.Package {
	t.Helper()
	db, e := state.Open(p, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	v, e := db.Packages()
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func TestInstallSwitchRemoveRollback(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	before := testutil.Read(t, p.Root, backend.Config)
	install := apply(t, p, "install", pkg.Manifest.ID, "")
	if !reflect.DeepEqual(install.Events, Phases) {
		t.Fatal(install.Events)
	}
	if !bytes.Equal(before, testutil.Read(t, p.Root, backend.Config)) {
		t.Fatal("install activated theme")
	}
	apply(t, p, "switch", pkg.Manifest.ID, "")
	q := packages(t, p)[0]
	if !q.Active || !q.Installed {
		t.Fatal(q)
	}
	second := apply(t, p, "switch", pkg.Manifest.ID, "hd")
	cfg := testutil.Read(t, p.Root, backend.Config)
	cfg = append(cfg, []byte("menuentry \"New kernel\" { linux /vmlinuz-new }\n")...)
	testutil.Write(t, p.Root, backend.Config, cfg)
	apply(t, p, "rollback", second.ID, "")
	now := testutil.Read(t, p.Root, backend.Config)
	if !strings.Contains(string(now), "New kernel") || strings.Contains(string(now), "/hd/theme.txt") {
		t.Fatal("rollback lost current kernel or selected wrong variant")
	}
	removed := apply(t, p, "remove", pkg.Manifest.ID, "")
	if packages(t, p)[0].Active {
		t.Fatal("remove left active")
	}
	destination := "boot/grub/themes/grubmgr/" + pkg.Manifest.ID + "/" + pkg.Manifest.Revision
	if _, e := os.Stat(filepath.Join(p.Root, destination)); e != nil {
		t.Fatal("rollback assets removed")
	}
	apply(t, p, "rollback", removed.ID, "")
	if !packages(t, p)[0].Active {
		t.Fatal("retained theme not restored")
	}
}
func TestFailuresEveryPhase(t *testing.T) {
	for _, phase := range Phases {
		t.Run(phase, func(t *testing.T) {
			p := testutil.Root(t, "debian")
			pkg := fetchDemo(t, p)
			pl := plan(t, p, "install", pkg.Manifest.ID, "")
			defaults := testutil.Read(t, p.Root, backend.Defaults)
			cfg := testutil.Read(t, p.Root, backend.Config)
			before := packages(t, p)
			tx, e := Apply(p, pl.ID, Faults{FailAfter: phase})
			if e == nil || tx.Phase != "rolled_back" {
				t.Fatal(tx, e)
			}
			if !bytes.Equal(defaults, testutil.Read(t, p.Root, backend.Defaults)) || !bytes.Equal(cfg, testutil.Read(t, p.Root, backend.Config)) || !reflect.DeepEqual(before, packages(t, p)) {
				t.Fatal("pretransaction state not restored")
			}
			if _, e = os.Stat(filepath.Join(p.Root, tx.Destination)); !os.IsNotExist(e) {
				t.Fatal("new incomplete assets not removed", e)
			}
			if tx.Events[len(tx.Events)-2] != "recovering" {
				t.Fatal(tx.Events)
			}
		})
	}
}
func TestSwitchFailuresRetainActiveAssets(t *testing.T) {
	for _, phase := range Phases {
		t.Run(phase, func(t *testing.T) {
			p := testutil.Root(t, "debian")
			pkg := fetchDemo(t, p)
			apply(t, p, "install", pkg.Manifest.ID, "")
			apply(t, p, "switch", pkg.Manifest.ID, "")
			pl := plan(t, p, "switch", pkg.Manifest.ID, "hd")
			defaults := testutil.Read(t, p.Root, backend.Defaults)
			cfg := testutil.Read(t, p.Root, backend.Config)
			tx, e := Apply(p, pl.ID, Faults{FailAfter: phase})
			if e == nil || tx.Phase != "rolled_back" {
				t.Fatal(e)
			}
			if !bytes.Equal(defaults, testutil.Read(t, p.Root, backend.Defaults)) || !bytes.Equal(cfg, testutil.Read(t, p.Root, backend.Config)) {
				t.Fatal("switch recovery failed")
			}
			_, h, e := fsx.Inventory(filepath.Join(p.Root, tx.Destination))
			if e != nil || h != pkg.Manifest.TreeSHA256 {
				t.Fatal("active assets lost")
			}
			if !packages(t, p)[0].Active {
				t.Fatal("active receipt lost")
			}
		})
	}
}
func TestInterruptedRecovery(t *testing.T) {
	for _, phase := range Phases[:len(Phases)-1] {
		t.Run(phase, func(t *testing.T) {
			p := testutil.Root(t, "debian")
			pkg := fetchDemo(t, p)
			apply(t, p, "install", pkg.Manifest.ID, "")
			before := testutil.Read(t, p.Root, backend.Config)
			tx, e := Apply(p, plan(t, p, "switch", pkg.Manifest.ID, "").ID, Faults{CrashAfter: phase})
			if e == nil || tx.Phase != phase {
				t.Fatal(tx, e)
			}
			if _, e = planner.Build(p, planner.Request{Action: "switch", Target: pkg.Manifest.ID}); e == nil {
				t.Fatal("pending transaction allowed planning")
			}
			r, e := Recover(p)
			if e != nil || len(r) != 1 || r[0].Phase != "rolled_back" {
				t.Fatal(r, e)
			}
			if !bytes.Equal(before, testutil.Read(t, p.Root, backend.Config)) {
				t.Fatal("recovery changed original")
			}
		})
	}
}
func TestRecoveryFailureAndDrift(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	apply(t, p, "install", pkg.Manifest.ID, "")
	tx, e := Apply(p, plan(t, p, "switch", pkg.Manifest.ID, "").ID, Faults{FailAfter: "activated", RestoreFail: true})
	if e == nil || tx.Phase != "recovery_required" {
		t.Fatal(tx, e)
	}
	original := testutil.Read(t, p.Root, backend.Config)
	testutil.Write(t, p.Root, backend.Config, append(original, []byte("# external drift\n")...))
	if _, e = Recover(p); e == nil {
		t.Fatal("recovery overwrote external edits")
	}
	testutil.Write(t, p.Root, backend.Config, original)
	if _, e = Recover(p); e != nil {
		t.Fatal(e)
	}
}
func TestPlansReadOnlyAndStale(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	_, before, _ := fsx.Inventory(p.Root)
	pl := plan(t, p, "install", pkg.Manifest.ID, "")
	_, after, _ := fsx.Inventory(p.Root)
	if before != after {
		t.Fatal("planning modified root")
	}
	testutil.Write(t, p.Root, backend.Defaults, []byte("GRUB_TIMEOUT=42\n"))
	if _, e := Apply(p, pl.ID, Faults{}); e == nil {
		t.Fatal("stale plan accepted")
	}
	if strings.Contains(string(testutil.Read(t, p.Root, backend.Defaults)), "GRUB_THEME") {
		t.Fatal("stale plan changed configuration")
	}
}
func TestUnsupportedAndRootBinding(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	pl := plan(t, p, "install", pkg.Manifest.ID, "")
	p2 := testutil.Root(t, "debian")
	fetchDemo(t, p2)
	if _, e := Apply(p2, pl.ID, Faults{}); e == nil {
		t.Fatal("plan portable across roots")
	}
	for _, name := range []string{"arch", "fedora", "non-grub", "ambiguous"} {
		q := testutil.Root(t, name)
		v := fetchDemo(t, q)
		if pl, e := planner.Build(q, planner.Request{Action: "install", Target: v.Manifest.ID}); e == nil {
			if pl.Applicable {
				t.Fatal("unsupported backend applicable")
			}
			if _, e = Apply(q, pl.ID, Faults{}); e == nil {
				t.Fatal("unsupported backend applied")
			}
		}
	}
	p.Fixture = false
	if _, e := Apply(p, pl.ID, Faults{}); e == nil {
		t.Fatal("real activation allowed")
	}
}
func TestNoRootEscape(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	outside := t.TempDir()
	testutil.Write(t, outside, "sentinel", []byte("unchanged"))
	link := filepath.Join(p.Root, "boot/grub/themes")
	if e := os.Symlink(outside, link); e != nil {
		t.Skip("symlink creation unavailable:", e)
	}
	pl := plan(t, p, "install", pkg.Manifest.ID, "")
	if _, e := Apply(p, pl.ID, Faults{}); e == nil {
		t.Fatal("escape allowed")
	}
	files, _, e := fsx.Inventory(outside)
	if e != nil || len(files) != 1 {
		t.Fatal("outside root was mutated")
	}
}
func TestHardlinkCannotMutateOutside(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	apply(t, p, "install", pkg.Manifest.ID, "")
	outside := filepath.Join(t.TempDir(), "defaults")
	original := testutil.Read(t, p.Root, backend.Defaults)
	if e := os.WriteFile(outside, original, 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(p.Root, backend.Defaults)
	if e := os.Remove(target); e != nil {
		t.Fatal(e)
	}
	if e := os.Link(outside, target); e != nil {
		t.Skip("hardlinks unavailable", e)
	}
	pl := plan(t, p, "switch", pkg.Manifest.ID, "")
	if _, e := Apply(p, pl.ID, Faults{}); e == nil {
		t.Fatal("hardlink accepted")
	}
	b, e := os.ReadFile(outside)
	if e != nil || !bytes.Equal(b, original) {
		t.Fatal("outside hardlink changed")
	}
}
func TestTamperedCacheAndUnreviewed(t *testing.T) {
	p := testutil.Root(t, "debian")
	pkg := fetchDemo(t, p)
	testutil.Write(t, state.Content(p, pkg.Manifest.Revision), "theme.txt", []byte("title-text: \"altered\"\n"))
	if _, e := planner.Build(p, planner.Request{Action: "install", Target: pkg.Manifest.ID}); e == nil {
		t.Fatal("tampered cache planned")
	}
	d := t.TempDir()
	testutil.Write(t, d, "theme.txt", []byte("title-text: \"local\"\n"))
	local, e := fetch.Import(p, d, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = planner.Build(p, planner.Request{Action: "install", Target: local.Manifest.ID}); e == nil {
		t.Fatal("unknown license planned")
	}
}
func TestImmutableRevisionsNeedSelection(t *testing.T) {
	p := testutil.Root(t, "debian")
	first := fetchDemo(t, p)
	d := t.TempDir()
	for name, b := range catalog.DemoFiles() {
		testutil.Write(t, d, name, b)
	}
	testutil.Write(t, d, "theme.txt", []byte("title-text: \"new version\"\n"))
	entry, _ := catalog.Find(first.Manifest.ID)
	entry.Recipe.Source = model.Source{Provider: "local", URL: d, UpstreamRevision: "test-2"}
	entry.Recipe.Version = "2"
	recipe, _ := json.Marshal(entry.Recipe)
	file := filepath.Join(t.TempDir(), "recipe.json")
	os.WriteFile(file, recipe, 0600)
	second, e := fetch.Import(p, d, file)
	if e != nil || second.Manifest.Revision == first.Manifest.Revision {
		t.Fatal(e)
	}
	if _, e = planner.Build(p, planner.Request{Action: "install", Target: first.Manifest.ID}); e == nil {
		t.Fatal("ambiguous revision guessed")
	}
	apply(t, p, "install", first.Manifest.ID+"@"+first.Manifest.Revision, "")
	if !packages(t, p)[0].Installed && !packages(t, p)[1].Installed {
		t.Fatal("revision selection failed")
	}
}
