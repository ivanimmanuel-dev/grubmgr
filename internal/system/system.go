// Package system performs rooted, read-only inspection; it never runs GRUB programs.
package system

import (
	"fmt"
	"grubmgr/internal/fsx"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const Marker = "grubmgr synthetic fixture v1\n"

type Paths struct {
	Root, Config, Data, Cache, State string
	Fixture                          bool
}

func Locations(root string) (Paths, error) {
	if root != "" {
		a, e := filepath.Abs(root)
		if e != nil {
			return Paths{}, e
		}
		if e = fsx.NoLinks(a); e != nil {
			return Paths{}, e
		}
		return Paths{a, filepath.Join(a, ".grubmgr/config"), filepath.Join(a, ".grubmgr/data"), filepath.Join(a, ".grubmgr/cache"), filepath.Join(a, ".grubmgr/state"), true}, nil
	}
	home, e := os.UserHomeDir()
	if e != nil {
		return Paths{}, e
	}
	xdg := func(key, fallback string) string {
		p := os.Getenv(key)
		if !filepath.IsAbs(p) {
			p = filepath.Join(home, fallback)
		}
		return filepath.Join(p, "grubmgr")
	}
	hostRoot := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		hostRoot = filepath.VolumeName(home) + `\`
	}
	return Paths{hostRoot, xdg("XDG_CONFIG_HOME", ".config"), xdg("XDG_DATA_HOME", ".local/share"), xdg("XDG_CACHE_HOME", ".cache"), xdg("XDG_STATE_HOME", ".local/state"), false}, nil
}
func (p Paths) Ensure() error {
	for _, d := range []string{p.Data, p.Cache, p.State} {
		if e := fsx.NoLinks(d); e != nil {
			return e
		}
		if e := os.MkdirAll(d, 0700); e != nil {
			return e
		}
	}
	return nil
}

type Report struct {
	OperatingSystem  string            `json:"operating_system"`
	Distribution     string            `json:"distribution"`
	Version          string            `json:"version"`
	Architecture     string            `json:"architecture"`
	Firmware         string            `json:"firmware"`
	GRUBInstalled    bool              `json:"grub_appears_installed"`
	GRUBVersion      string            `json:"grub_version"`
	Utilities        map[string]string `json:"utilities"`
	Defaults         string            `json:"defaults"`
	Configs          []string          `json:"config_locations"`
	ProtectedConfigs []string          `json:"protected_configs"`
	BootLocations    []string          `json:"boot_locations"`
	MountInfo        string            `json:"mount_info,omitempty"`
	Theme            string            `json:"theme"`
	Backend          string            `json:"backend"`
	Status           string            `json:"status"`
	Reason           string            `json:"reason"`
	Fixture          bool              `json:"fixture"`
	Editable         bool              `json:"simple_defaults"`
	MissingOptional  []string          `json:"missing_optional"`
	Preview          string            `json:"preview"`
	Warnings         []string          `json:"warnings"`
}

var assignment = regexp.MustCompile(`^GRUB_THEME=(?:"([^"$` + "`" + `\\]*)"|'([^']*)'|([A-Za-z0-9_./-]*))\s*(?:#.*)?$`)
var literalAssignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=(?:"[^"$` + "`" + `\\]*"|'[^']*'|[A-Za-z0-9_./,-]*)\s*(?:#.*)?$`)

func Theme(data []byte) (string, bool) {
	value := ""
	count := 0
	ok := true
	for _, line := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if !literalAssignment.MatchString(s) {
			ok = false
		}
		if strings.Contains(s, "GRUB_THEME") {
			m := assignment.FindStringSubmatch(s)
			if m == nil {
				ok = false
				continue
			}
			count++
			value = m[1] + m[2] + m[3]
		}
	}
	return value, ok && count <= 1
}
func Inspect(p Paths) (Report, error) {
	q := Report{Architecture: "unknown", Firmware: "unknown", GRUBVersion: "unknown (utilities are not executed)", Utilities: map[string]string{}, Configs: []string{}, BootLocations: []string{}, MissingOptional: []string{}, Warnings: []string{}, Status: "unsupported", Backend: "none", Reason: "No conventional GRUB layout established", Preview: "unavailable"}
	r, e := os.OpenRoot(p.Root)
	if e != nil {
		return q, e
	}
	defer r.Close()
	read := func(s string) []byte {
		b, err := fsx.Read(r, s, 1<<20)
		if err != nil && !os.IsNotExist(err) {
			q.Warnings = append(q.Warnings, s+": "+err.Error())
		}
		return b
	}
	exists := func(s string) bool { _, err := r.Stat(s); return err == nil }
	release := string(read("etc/os-release"))
	fields := map[string]string{}
	for _, s := range strings.Split(release, "\n") {
		kv := strings.SplitN(s, "=", 2)
		if len(kv) == 2 {
			fields[kv[0]] = strings.Trim(kv[1], `"'`)
		}
	}
	q.Distribution = fields["ID"]
	q.OperatingSystem = "unknown"
	if release != "" {
		q.OperatingSystem = "linux"
	} else if !p.Fixture {
		q.OperatingSystem = runtime.GOOS
	}
	q.Version = fields["VERSION_ID"]
	if q.Distribution == "" {
		q.Distribution = "unknown"
	}
	q.Fixture = p.Fixture && string(read(".grubmgr-fixture")) == Marker
	if p.Fixture {
		q.Architecture = strings.TrimSpace(string(read("etc/grubmgr-architecture")))
		if q.Architecture == "" {
			q.Architecture = "unknown"
		}
		q.Firmware = strings.TrimSpace(string(read("etc/grubmgr-firmware")))
		if q.Firmware == "" {
			q.Firmware = "unknown"
		}
	} else {
		q.Architecture = runtime.GOARCH
	}
	if exists("sys/firmware/efi") {
		q.Firmware = "UEFI"
	}
	for _, tool := range []string{"grub-mkconfig", "grub2-mkconfig", "update-grub", "grub-script-check", "grub2-theme-preview", "qemu-system-x86_64", "xorriso", "mformat"} {
		for _, prefix := range []string{"usr/bin/", "usr/sbin/", "bin/", "sbin/"} {
			if exists(prefix + tool) {
				q.Utilities[tool] = "/" + prefix + tool
				break
			}
		}
	}
	for _, tool := range []string{"grub2-theme-preview", "qemu-system-x86_64", "xorriso", "mformat"} {
		if q.Utilities[tool] == "" {
			q.MissingOptional = append(q.MissingOptional, tool)
		}
	}
	if len(q.MissingOptional) == 0 {
		q.Preview = "dependencies detected; execution disabled pending sandbox adapter review"
	}
	for _, s := range []string{"boot/grub/grub.cfg", "boot/grub2/grub.cfg"} {
		if exists(s) {
			q.Configs = append(q.Configs, "/"+s)
			cfg := string(read(s))
			v := regexp.MustCompile(`(?i)GRUB version ([0-9][0-9A-Za-z.+-]*)`).FindStringSubmatch(cfg)
			if len(v) > 1 {
				q.GRUBVersion = v[1] + " (configuration evidence)"
			}
		}
	}
	q.ProtectedConfigs = []string{}
	if exists("boot/efi/EFI/fedora/grub.cfg") {
		q.ProtectedConfigs = append(q.ProtectedConfigs, "/boot/efi/EFI/fedora/grub.cfg")
	}
	q.GRUBInstalled = len(q.Configs) > 0
	if exists("etc/default/grub") {
		q.Defaults = "/etc/default/grub"
		q.Theme, q.Editable = Theme(read("etc/default/grub"))
	}
	for _, s := range []string{"boot", "boot/efi", "efi"} {
		if exists(s) {
			q.BootLocations = append(q.BootLocations, "/"+s)
		}
	}
	q.MountInfo = string(read("proc/self/mountinfo"))
	if q.MountInfo == "" {
		q.Warnings = append(q.Warnings, "Mount visibility/boot readability unverified")
	}
	if len(q.Configs) > 1 {
		q.Status = "ambiguous"
		q.Reason = "Multiple plausible GRUB layouts"
	} else if q.GRUBInstalled && exists("boot/loader/loader.conf") {
		q.Status = "ambiguous"
		q.Reason = "GRUB and another bootloader have configuration evidence"
	} else if q.GRUBInstalled {
		q.Status = "read-only"
		q.Reason = "Activation requires a tested disposable VM profile"
		switch q.Distribution {
		case "debian", "ubuntu", "kali":
			if q.Configs[0] == "/boot/grub/grub.cfg" && q.Defaults != "" && q.Utilities["grub-mkconfig"] != "" {
				q.Backend = "debian-conventional"
			}
		case "arch":
			q.Backend = "arch-conventional"
		case "fedora":
			q.Backend = "fedora-conventional"
			q.Warnings = append(q.Warnings, "Protect EFI forwarding stub; preserve BLS and grubenv")
		}
		if q.Backend == "debian-conventional" && q.Fixture && q.Editable {
			q.Status = "fixture-only"
			q.Reason = "Synthetic Debian backend; no GRUB utility will run"
		}
	}
	if exists("etc/default/grub.d") || exists("etc/grub.d/proxifiedScripts") {
		q.Editable = false
		q.Status = "read-only"
		q.Reason = "Defaults fragments or GRUB Customizer proxies need separate review"
	}
	if !q.Editable && q.Defaults != "" {
		q.Warnings = append(q.Warnings, "Theme assignments are complex or conflicting")
	}
	q.Warnings = append(q.Warnings, "File evidence cannot prove the active firmware bootloader or successful boot")
	if !p.Fixture {
		if q.Status == "ambiguous" {
			q.Status = "AMBIGUOUS"
		} else {
			q.Status = "UNSUPPORTED"
		}
		q.Reason = "Activation requires a supported disposable Linux VM"
	}
	return q, nil
}
func RequireFixture(p Paths) (*os.Root, error) {
	if !p.Fixture || filepath.Dir(p.Root) == p.Root {
		return nil, fmt.Errorf("real activation disabled: an explicit non-volume fixture root is required")
	}
	if e := fsx.NoLinks(p.Root); e != nil {
		return nil, e
	}
	r, e := os.OpenRoot(p.Root)
	if e != nil {
		return nil, e
	}
	b, e := fsx.Read(r, ".grubmgr-fixture", 128)
	if e != nil || string(b) != Marker {
		r.Close()
		return nil, fmt.Errorf("fixture marker absent or invalid")
	}
	return r, nil
}
