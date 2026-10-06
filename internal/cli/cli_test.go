package cli

import (
	"bytes"
	"encoding/json"
	"grubmgr/internal/testutil"
	"strings"
	"testing"
)

func TestStructuredLogs(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"--json", "--log-json", "version"}, &out, &err); code != 0 {
		t.Fatal(code)
	}
	if !json.Valid(out.Bytes()) {
		t.Fatal(out.String())
	}
	lines := strings.Split(strings.TrimSpace(err.String()), "\n")
	if len(lines) != 2 {
		t.Fatal(err.String())
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatal(line)
		}
	}
}

func TestEndToEnd(t *testing.T) {
	p := testutil.Root(t, "debian")
	run := func(args ...string) map[string]any {
		t.Helper()
		var out, err bytes.Buffer
		code := Run(append([]string{"--root", p.Root, "--json"}, args...), &out, &err)
		if code != 0 {
			t.Fatalf("%v: exit %d: %s", args, code, err.String())
		}
		var v map[string]any
		if e := json.Unmarshal(out.Bytes(), &v); e != nil {
			t.Fatal(e, out.String())
		}
		return v
	}
	run("doctor")
	run("info", "cyberpunk-demo")
	run("fetch", "cyberpunk-demo")
	run("validate", "cyberpunk-demo")
	pl := run("plan", "install", "cyberpunk-demo")
	run("apply", pl["plan_id"].(string))
	pl = run("plan", "switch", "cyberpunk-demo")
	run("apply", pl["plan_id"].(string))
	run("status")
	for _, args := range [][]string{{"search"}, {"list"}, {"history"}} {
		var out, err bytes.Buffer
		if code := Run(append([]string{"--root", p.Root, "--json"}, args...), &out, &err); code != 0 || !json.Valid(out.Bytes()) {
			t.Fatal(code, out.String(), err.String())
		}
	}
}
func TestErrorsAndPreview(t *testing.T) {
	p := testutil.Root(t, "debian")
	for _, tt := range []struct {
		args []string
		code int
	}{{[]string{"wat"}, 2}, {[]string{"apply", "bad-token"}, 2}, {[]string{"fetch", "https://example.invalid/theme.zip"}, 4}, {[]string{"--root"}, 2}} {
		var out, err bytes.Buffer
		code := Run(append([]string{"--root", p.Root, "--json"}, tt.args...), &out, &err)
		if code != tt.code || !json.Valid(err.Bytes()) {
			t.Fatal(tt, code, err.String())
		}
	}
	var out, err bytes.Buffer
	Run([]string{"--root", p.Root, "fetch", "cyberpunk-demo"}, &out, &err)
	if code := Run([]string{"--root", p.Root, "preview", "cyberpunk-demo"}, &out, &err); code != 4 {
		t.Fatal(code, err.String())
	}
}

func TestDirectCommandsConfirmBeforeApplying(t *testing.T) {
	p := testutil.Root(t, "debian")
	call := func(answer string, args ...string) (int, string) {
		t.Helper()
		var out, diagnostics bytes.Buffer
		code := RunWithInput(append([]string{"--root", p.Root}, args...), strings.NewReader(answer), &out, &diagnostics)
		return code, out.String() + diagnostics.String()
	}
	if code, text := call("n\n", "install", "cyberpunk-demo"); code != 2 || !strings.Contains(text, "CANCELLED") {
		t.Fatal(code, text)
	}
	if code, text := call("y\n", "install", "cyberpunk-demo"); code != 0 {
		t.Fatal(code, text)
	}
	if code, text := call("", "--json", "switch", "cyberpunk-demo"); code != 2 || !strings.Contains(text, "CONFIRMATION") {
		t.Fatal(code, text)
	}
	if code, text := call("", "--yes", "--json", "switch", "cyberpunk-demo"); code != 0 || !json.Valid([]byte(text)) {
		t.Fatal(code, text)
	}
}
