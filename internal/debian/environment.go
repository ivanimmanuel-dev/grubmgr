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
	profile     profile
}

func paths(root string) system.Paths {
	return system.Paths{Root: root, Data: path.Join(root, Journal, "data"), State: path.Join(root, Journal, "state"), Cache: path.Join(root, Journal, "cache"), Config: path.Join(root, "etc/grubmgr")}
}

func Inspect() (Environment, error) { return inspect("/") }

func inspect(root string) (Environment, error) {
	q := Environment{Status: "UNSUPPORTED", Backend: "none", Warnings: []string{}}
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
	target, err := selectProfile(read("etc/os-release"))
	if err != nil {
		return refuse(err.Error())
	}
	q.profile, q.Backend = target, target.Backend
	sb := read("sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c")
	if _, err := r.Stat("sys/firmware/efi"); err != nil {
		return refuse("requires a UEFI installation")
	}
	if len(sb) != 5 || sb[4] != 0 {
		return refuse("theme activation requires Secure Boot reported disabled")
	}
	q.Firmware = "UEFI; Secure Boot disabled"
	for _, p := range []string{"boot/grub2/grub.cfg", "boot/loader/loader.conf", "efi/loader/loader.conf", "etc/grub.d/proxifiedScripts", "etc/grub.d/40_custom_proxy", "etc/ostree", "etc/snapper/configs/root"} {
		if _, e := r.Stat(p); e == nil {
			q.Status = "AMBIGUOUS"
			return refuse("conflicting boot configuration: /" + p)
		}
	}
	versions, packageFiles, err := installedVersions(r, target)
	if err != nil {
		return refuse(err.Error())
	}
	q.GRUBVersion = versions["grub-common"]
	if target.PackageManager == "pacman" {
		q.GRUBVersion = versions["grub"]
	}
	if !grub2Version(q.GRUBVersion) || (target.PackageManager == "dpkg" && versions["grub2-common"] != q.GRUBVersion) {
		return refuse("requires matching installed GRUB 2 packages: " + q.GRUBVersion)
	}
	if versions["grub-customizer"] != "" || versions["grub-btrfs"] != "" || versions["snapper"] != "" {
		return refuse("GRUB Customizer and snapshot integration are unsupported")
	}
	mounts := string(read("proc/self/mountinfo"))
	target, err = bootLayout(mounts, target)
	if err != nil {
		return refuse(err.Error())
	}
	q.profile = target
	q.Layout = target.RootFS + " root including /boot"
	if target.SeparateBoot {
		q.Layout = target.RootFS + " root; separate /boot"
	}
	evidence := []model.File{}
	if target.PackageManager == "dpkg" {
		link, linkErr := os.Readlink(path.Join(root, "usr/lib/grub/grub-mkconfig_lib"))
		if linkErr != nil || link != "../../share/grub/grub-mkconfig_lib" {
			return refuse("unexpected GRUB library link")
		}
		if err := secure(path.Join(root, "usr/lib/grub"), true); err != nil {
			return refuse(err.Error())
		}
		evidence = append(evidence, model.File{Path: "usr/lib/grub/grub-mkconfig_lib", Size: int64(len(link)), SHA256: model.Hash([]byte(link))})
	}
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
	for _, name := range append([]string{Defaults, Config, "etc/machine-id", target.Generator, "usr/bin/grub-script-check", target.Probe, target.Library}, packageFiles...) {
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
		if strings.HasPrefix(file.Name(), "vmlinuz-") || strings.HasPrefix(file.Name(), "initrd.img-") || strings.HasPrefix(file.Name(), "initramfs-") || strings.HasSuffix(file.Name(), "-ucode.img") {
			if err = add("boot/" + file.Name()); err != nil {
				return refuse(err.Error())
			}
		}
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].Path < evidence[j].Path })
	q.Fingerprint = model.Digest(struct {
		Profile profile
		Release string
		Mounts  string
		Files   []model.File
	}{target, string(read("etc/os-release")), bootMountEvidence(mounts), evidence})
	q.Status = "SUPPORTED"
	q.Reason = "GRUB 2 configuration and theme destination verified for " + target.ID
	q.Warnings = []string{}
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
