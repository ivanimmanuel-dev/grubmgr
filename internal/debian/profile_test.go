package debian

import "testing"

func TestProfileRejectsUnknownAndAmbiguousOS(t *testing.T) {
	for _, release := range []string{
		"ID=mint\nID_LIKE=ubuntu\nVERSION_ID=24.04\n",
		"ID=debian\nVERSION_ID=12\n", "ID=debian\nID=debian\nVERSION_ID=13\n",
		"ID=$(debian)\nVERSION_ID=13\n", "ID=debian\nVERSION_ID=\"13\n",
	} {
		if _, err := selectProfile([]byte(release)); err == nil {
			t.Fatal("accepted untested OS", release)
		}
	}
	for _, target := range profiles {
		release := "ID=\"" + target.ID + "\"\nVERSION_ID='" + target.Version + "'\n"
		got, err := selectProfile([]byte(release))
		if err != nil || got.Backend != target.Backend {
			t.Fatal(target, got, err)
		}
	}
}

func TestPackageDatabaseParsers(t *testing.T) {
	versions := dpkgVersions([]byte("Package: grub-common\nStatus: install ok installed\nVersion: 1\n\nPackage: removed\nStatus: deinstall ok config-files\nVersion: 2\n"))
	if versions["grub-common"] != "1" || versions["removed"] != "" {
		t.Fatal(versions)
	}
	name, version, err := pacmanFields([]byte("%NAME%\ngrub\n\n%VERSION%\n2:2.16-1\n\n%DEPENDS%\na\nb\n"))
	if err != nil || name != "grub" || version != "2:2.16-1" {
		t.Fatal(name, version, err)
	}
	for _, record := range []string{"%NAME%\ngrub\n", "%NAME%\ngrub\n\n%NAME%\nother\n\n%VERSION%\n1\n"} {
		if _, _, err = pacmanFields([]byte(record)); err == nil {
			t.Fatal("accepted incomplete or duplicate record")
		}
	}
	for _, conf := range []string{"[options]\nDBPath = /other\n", "[options]\nInclude = /etc/custom\n", "[core]\nInclude = /other\n"} {
		if standardPacmanConfig([]byte(conf)) == nil {
			t.Fatal("accepted custom package database configuration", conf)
		}
	}
	if err = standardPacmanConfig([]byte("[options]\n#DBPath = /var/lib/pacman\n[core]\nInclude = /etc/pacman.d/mirrorlist\n")); err != nil {
		t.Fatal(err)
	}
}

func TestBootLayoutsAndGRUBUpdates(t *testing.T) {
	for _, version := range []string{"2.12-9+deb13u3", "2:2.16-2", "2.14-3+kali1", "2.12-1ubuntu7.4"} {
		if !grub2Version(version) {
			t.Fatal("GRUB update rejected", version)
		}
	}
	for _, version := range []string{"", "1.99", "3.0", "2.", "custom"} {
		if grub2Version(version) {
			t.Fatal("unsupported GRUB version accepted", version)
		}
	}
	for _, mounts := range []string{
		"30 1 8:2 / / rw,relatime - ext4 /dev/sda2 rw\n",
		"30 1 8:2 /@ / rw,relatime - btrfs /dev/sda2 rw,subvolid=256,subvol=/@\n",
		"30 1 8:2 / / rw,relatime - ext4 /dev/sda2 rw\n31 30 8:3 / /boot rw,relatime - ext4 /dev/sda3 rw\n",
	} {
		if _, err := bootLayout(mounts, profiles[0]); err != nil {
			t.Fatal(mounts, err)
		}
	}
	for _, mounts := range []string{
		"", "30 1 8:2 / / ro - ext4 /dev/sda2 ro\n",
		"30 1 8:2 / / rw - zfs pool/root rw\n",
		"30 1 8:2 / / rw - ext4 /dev/sda2 rw\n31 30 8:3 / /boot ro - ext4 /dev/sda3 ro\n",
	} {
		if _, err := bootLayout(mounts, profiles[0]); err == nil {
			t.Fatal("unsupported layout accepted", mounts)
		}
	}
}
