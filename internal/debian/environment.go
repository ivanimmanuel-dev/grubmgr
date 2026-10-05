package debian

import (
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/system"
	"io/fs"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
)

const VMMarker = "grubmgr disposable VM v1\n"
const Defaults = "etc/default/grub"
const Config = "boot/grub/grub.cfg"
const Journal = "var/lib/grubmgr"

type Environment struct {
	Status      string   `json:"status"`
	Reason      string   `json:"reason"`
	Backend     string   `json:"backend"`
	GRUBVersion string   `json:"grub_version"`
	Firmware    string   `json:"firmware"`
	Layout      string   `json:"layout"`
	Fingerprint string   `json:"fingerprint"`
	Warnings    []string `json:"warnings"`
}

func paths(root string) system.Paths {
	return system.Paths{Root: root, Data: path.Join(root, Journal, "data"), State: path.Join(root, Journal, "state"), Cache: path.Join(root, Journal, "cache"), Config: path.Join(root, "etc/grubmgr")}
}

func Inspect() (Environment, error) { return inspect("/") }

func inspect(root string) (Environment, error) {
	q := Environment{Status: "UNSUPPORTED", Backend: Backend, Warnings: []string{}}
	refuse := func(reason string) (Environment, error) { q.Reason = reason; return q, nil }
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return refuse("requires Linux amd64")
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return q, err
	}
	defer r.Close()
	read := func(p string) []byte { b, _ := fsx.Read(r, p, 8<<20); return b }
	if string(read("etc/grubmgr/vm-test")) != VMMarker || strings.TrimSpace(string(read("sys/class/dmi/id/product_name"))) != "grubmgr-disposable-v1" {
		return refuse("activation is restricted to the disposable test VM")
	}
	if err := secure(path.Join(root, "etc/grubmgr/vm-test"), false); err != nil {
		return refuse(err.Error())
	}
	release := string(read("etc/os-release"))
	if !strings.Contains("\n"+release, "\nID=debian\n") || !strings.Contains(release, "VERSION_ID=\"13\"") {
		return refuse("only Debian 13 is enabled")
	}
	sb := read("sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c")
	if len(sb) != 5 || sb[4] != 0 {
		return refuse("requires UEFI with Secure Boot reported disabled")
	}
	q.Firmware = "UEFI; Secure Boot disabled"
	for _, p := range []string{"boot/grub2/grub.cfg", "boot/loader/loader.conf", "etc/grub.d/proxifiedScripts", "etc/grub.d/40_custom_proxy", "etc/ostree"} {
		if _, e := r.Stat(p); e == nil {
			q.Status = "AMBIGUOUS"
			return refuse("conflicting boot configuration: /" + p)
		}
	}
	status := string(read("var/lib/dpkg/status"))
	versions := map[string]string{}
	for _, stanza := range strings.Split(status, "\n\n") {
		fields := map[string]string{}
		for _, line := range strings.Split(stanza, "\n") {
			kv := strings.SplitN(line, ": ", 2)
			if len(kv) == 2 {
				fields[kv[0]] = kv[1]
			}
		}
		if fields["Status"] == "install ok installed" {
			versions[fields["Package"]] = fields["Version"]
		}
	}
	q.GRUBVersion = versions["grub-common"]
	if q.GRUBVersion != "2.12-9+deb13u2" || versions["grub2-common"] != q.GRUBVersion {
		return refuse("untested GRUB package version: " + q.GRUBVersion)
	}
	if versions["grub-customizer"] != "" || versions["grub-btrfs"] != "" {
		return refuse("GRUB Customizer and snapshot integration are unsupported")
	}
	mounts := string(read("proc/self/mountinfo"))
	rootOK, efiOK := false, false
	for _, line := range strings.Split(mounts, "\n") {
		halves := strings.SplitN(line, " - ", 2)
		if len(halves) != 2 {
			continue
		}
		left, right := strings.Fields(halves[0]), strings.Fields(halves[1])
		if len(left) < 6 || len(right) < 3 {
			continue
		}
		switch left[4] {
		case "/":
			rootOK = right[0] == "ext4" && strings.Contains(","+left[5]+",", ",rw,")
		case "/boot":
			return refuse("separate /boot has not been tested")
		case "/boot/efi":
			efiOK = right[0] == "vfat"
		}
	}
	if !rootOK || !efiOK {
		return refuse("requires writable ext4 root with /boot on root and a mounted FAT EFI partition")
	}
	q.Layout = "ext4 root including /boot; FAT /boot/efi"
	evidence := []model.File{}
	link, linkErr := os.Readlink(path.Join(root, "usr/lib/grub/grub-mkconfig_lib"))
	if linkErr != nil || link != "../../share/grub/grub-mkconfig_lib" {
		return refuse("unexpected Debian GRUB library link")
	}
	if err := secure(path.Join(root, "usr/lib/grub"), true); err != nil {
		return refuse(err.Error())
	}
	evidence = append(evidence, model.File{Path: "usr/lib/grub/grub-mkconfig_lib", Size: int64(len(link)), SHA256: model.Hash([]byte(link))})
	add := func(name string) error {
		if err := secure(path.Join(root, name), false); err != nil {
			return err
		}
		b, err := fsx.Read(r, name, 128<<20)
		if err != nil {
			return err
		}
		evidence = append(evidence, model.File{Path: name, Size: int64(len(b)), SHA256: model.Hash(b)})
		return nil
	}
	for _, name := range []string{Defaults, Config, "etc/machine-id", "usr/sbin/grub-mkconfig", "usr/bin/grub-script-check", "usr/sbin/grub-probe", "usr/share/grub/grub-mkconfig_lib", "var/lib/dpkg/status"} {
		if err := add(name); err != nil {
			return refuse(err.Error())
		}
	}
	if _, ok := themeValue(read(Defaults)); !ok {
		return refuse("ambiguous GRUB_THEME assignment")
	}
	for _, directory := range []string{"etc/grub.d", "etc/default/grub.d"} {
		err := fs.WalkDir(r.FS(), directory, func(name string, d fs.DirEntry, err error) error {
			if os.IsNotExist(err) && name == directory {
				return nil
			}
			if err != nil {
				return err
			}
			if d.IsDir() {
				return secure(path.Join(root, name), true)
			}
			if err := add(name); err != nil {
				return err
			}
			if directory == "etc/default/grub.d" && strings.Contains(string(read(name)), "GRUB_THEME") {
				return fmt.Errorf("defaults fragment overrides theme: %s", name)
			}
			return nil
		})
		if err != nil {
			return refuse(err.Error())
		}
	}
	boot, err := fs.ReadDir(r.FS(), "boot")
	if err != nil {
		return refuse(err.Error())
	}
	for _, file := range boot {
		if strings.HasPrefix(file.Name(), "vmlinuz-") || strings.HasPrefix(file.Name(), "initrd.img-") {
			if err = add("boot/" + file.Name()); err != nil {
				return refuse(err.Error())
			}
		}
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].Path < evidence[j].Path })
	q.Fingerprint = model.Digest(evidence)
	q.Status = "SUPPORTED WITH WARNINGS"
	q.Reason = "experimental activation in the disposable Debian VM"
	q.Warnings = []string{"No physical-machine support; a successful syntax check is not boot verification"}
	return q, nil
}

func themeValue(b []byte) (string, bool) {
	var relevant []string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, "GRUB_THEME") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			relevant = append(relevant, line)
		}
	}
	return system.Theme([]byte(strings.Join(relevant, "\n")))
}

func settings(before []byte, theme string) ([]byte, error) {
	if _, ok := themeValue(before); !ok {
		return nil, fmt.Errorf("ambiguous theme setting")
	}
	if theme != "" && (!strings.HasPrefix(theme, "/boot/grub/themes/grubmgr/") || fsx.SafePath(strings.TrimPrefix(theme, "/")) != nil || strings.ContainsAny(theme, "'$`")) {
		return nil, fmt.Errorf("invalid derived theme location")
	}
	var out strings.Builder
	for _, line := range strings.SplitAfter(string(before), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "GRUB_THEME=") {
			continue
		}
		out.WriteString(line)
	}
	if theme != "" {
		if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
			out.WriteByte('\n')
		}
		fmt.Fprintf(&out, "GRUB_THEME=\"%s\"\n", theme)
	}
	return []byte(out.String()), nil
}
