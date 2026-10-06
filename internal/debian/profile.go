package debian

import (
	"fmt"
	"grubmgr/internal/fsx"
	"os"
	"path"
	"strings"
)

// A profile selects distro tools and records the detected GRUB and boot layout.
type profile struct {
	ID, Version, Backend, PackageManager string
	Generator, Probe, Library            string
	SeparateBoot                         bool
	RootFS                               string
}

func installedVersions(r *os.Root, p profile) (map[string]string, []string, error) {
	if p.PackageManager == "dpkg" {
		b, err := fsx.Read(r, "var/lib/dpkg/status", 32<<20)
		return dpkgVersions(b), []string{"var/lib/dpkg/status"}, err
	}
	versions := map[string]string{}
	files := []string{"etc/pacman.conf"}
	conf, err := fsx.Read(r, "etc/pacman.conf", 1<<20)
	if err != nil {
		return nil, nil, err
	}
	if err = standardPacmanConfig(conf); err != nil {
		return nil, nil, err
	}
	db, err := r.Open("var/lib/pacman/local")
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	entries, err := db.ReadDir(-1)
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range entries {
		if entry.Name() == "ALPM_DB_VERSION" {
			continue
		}
		if !entry.IsDir() {
			return nil, nil, fmt.Errorf("unexpected pacman database entry")
		}
		name := path.Join("var/lib/pacman/local", entry.Name(), "desc")
		data, err := fsx.Read(r, name, 4<<20)
		if err != nil {
			return nil, nil, err
		}
		pkg, version, err := pacmanFields(data)
		if err != nil {
			return nil, nil, err
		}
		if versions[pkg] != "" {
			return nil, nil, fmt.Errorf("duplicate installed package: %s", pkg)
		}
		versions[pkg] = version
		files = append(files, name)
	}
	return versions, files, nil
}

func standardPacmanConfig(conf []byte) error {
	section := ""
	for _, line := range strings.Split(string(conf), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(kv[0])
		if key == "RootDir" || key == "DBPath" {
			return fmt.Errorf("custom pacman RootDir/DBPath is unsupported")
		}
		if key == "Include" && (section == "[options]" || len(kv) != 2 || strings.TrimSpace(kv[1]) != "/etc/pacman.d/mirrorlist") {
			return fmt.Errorf("unsupported pacman configuration include")
		}
	}
	return nil
}

var profiles = []profile{
	{ID: "debian", Version: "13", Backend: "debian-grub", PackageManager: "dpkg", Generator: "usr/sbin/grub-mkconfig", Probe: "usr/sbin/grub-probe", Library: "usr/share/grub/grub-mkconfig_lib"},
	{ID: "ubuntu", Version: "24.04", Backend: "ubuntu-grub", PackageManager: "dpkg", Generator: "usr/sbin/grub-mkconfig", Probe: "usr/sbin/grub-probe", Library: "usr/share/grub/grub-mkconfig_lib"},
	{ID: "arch", Backend: "arch-grub", PackageManager: "pacman", Generator: "usr/bin/grub-mkconfig", Probe: "usr/bin/grub-probe", Library: "usr/share/grub/grub-mkconfig_lib"},
	{ID: "kali", Backend: "kali-grub", PackageManager: "dpkg", Generator: "usr/sbin/grub-mkconfig", Probe: "usr/sbin/grub-probe", Library: "usr/share/grub/grub-mkconfig_lib"},
}

func grub2Version(version string) bool {
	if _, rest, found := strings.Cut(version, ":"); found {
		version = rest
	}
	return strings.HasPrefix(version, "2.") && len(version) > 2 && version[2] >= '0' && version[2] <= '9'
}

// bootLayout derives mount information instead of requiring a test image layout.
func bootLayout(mounts string, p profile) (profile, error) {
	rootOK := false
	for _, line := range strings.Split(mounts, "\n") {
		leftText, rightText, found := strings.Cut(line, " - ")
		if !found {
			continue
		}
		left, right := strings.Fields(leftText), strings.Fields(rightText)
		if len(left) < 6 || len(right) < 3 {
			continue
		}
		if left[4] == "/" || left[4] == "/boot" || left[4] == "/boot/grub" || strings.HasPrefix(left[4], "/boot/grub/") {
			if !strings.Contains(","+left[5]+",", ",rw,") || right[0] != "ext4" && right[0] != "btrfs" {
				return p, fmt.Errorf("/%s requires a writable ext4 or Btrfs filesystem", strings.TrimPrefix(left[4], "/"))
			}
			if left[4] == "/" {
				p.RootFS, rootOK = right[0], true
			} else if left[4] == "/boot" {
				p.SeparateBoot = true
			}
		}
	}
	if !rootOK {
		return p, fmt.Errorf("cannot establish the root filesystem")
	}
	return p, nil
}

func bootMountEvidence(mounts string) string {
	var relevant []string
	for _, line := range strings.Split(mounts, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 6 && (fields[4] == "/" || fields[4] == "/boot" || fields[4] == "/boot/grub" || strings.HasPrefix(fields[4], "/boot/grub/")) {
			relevant = append(relevant, line)
		}
	}
	return strings.Join(relevant, "\n")
}

func compatibleBackend(declared, actual string) bool {
	if declared == actual || declared == "linux-grub" {
		return true
	}
	// Receipts from 0.3 keep their immutable metadata during an upgrade.
	legacy := map[string]string{"debian13-uefi-vm": "debian-grub", "ubuntu2404-uefi-vm": "ubuntu-grub", "kali-rolling-uefi-vm": "kali-grub", "arch-uefi-vm": "arch-grub"}
	return legacy[declared] == actual
}

func selectProfile(release []byte) (profile, error) {
	fields := map[string]string{}
	for _, line := range strings.Split(string(release), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return profile{}, fmt.Errorf("invalid os-release")
		}
		if _, exists := fields[kv[0]]; exists {
			return profile{}, fmt.Errorf("duplicate os-release field")
		}
		value := kv[1]
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		fields[kv[0]] = value
	}
	for _, p := range profiles {
		if fields["ID"] == p.ID && (p.Version == "" || fields["VERSION_ID"] == p.Version) {
			return p, nil
		}
	}
	return profile{}, fmt.Errorf("untested distribution or release: %s %s", fields["ID"], fields["VERSION_ID"])
}

func dpkgVersions(status []byte) map[string]string {
	versions := map[string]string{}
	for _, stanza := range strings.Split(string(status), "\n\n") {
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
	return versions
}

func pacmanFields(data []byte) (string, string, error) {
	fields := map[string]string{}
	for _, block := range strings.Split(string(data), "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) == 2 && (lines[0] == "%NAME%" || lines[0] == "%VERSION%") {
			if fields[lines[0]] != "" {
				return "", "", fmt.Errorf("duplicate pacman field")
			}
			fields[lines[0]] = lines[1]
		}
	}
	if fields["%NAME%"] == "" || fields["%VERSION%"] == "" {
		return "", "", fmt.Errorf("incomplete pacman record")
	}
	return fields["%NAME%"], fields["%VERSION%"], nil
}
