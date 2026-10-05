package debian

import (
	"fmt"
	"grubmgr/internal/fsx"
	"os"
	"path"
	"strings"
)

// A profile fixes the distro identity, GRUB version, tools and boot layout.
type profile struct {
	ID, Version, Backend, PackageManager, GRUBVersion string
	Generator, Probe, Library                         string
	SeparateBoot                                      bool
	RootFS, EFIMount                                  string
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
	{"debian", "13", "debian13-uefi-vm", "dpkg", "2.12-9+deb13u2", "usr/sbin/grub-mkconfig", "usr/sbin/grub-probe", "usr/share/grub/grub-mkconfig_lib", false, "ext4", "/boot/efi"},
	{"ubuntu", "24.04", "ubuntu2404-uefi-vm", "dpkg", "2.12-1ubuntu7.3", "usr/sbin/grub-mkconfig", "usr/sbin/grub-probe", "usr/share/grub/grub-mkconfig_lib", true, "ext4", "/boot/efi"},
	{"arch", "", "arch-uefi-vm", "pacman", "2:2.16-1", "usr/bin/grub-mkconfig", "usr/bin/grub-probe", "usr/share/grub/grub-mkconfig_lib", false, "btrfs", "/efi"},
	{"kali", "", "kali-rolling-uefi-vm", "dpkg", "2.14-2+kali1", "usr/sbin/grub-mkconfig", "usr/sbin/grub-probe", "usr/share/grub/grub-mkconfig_lib", false, "ext4", "/boot/efi"},
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
