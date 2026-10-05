# Linux targets

The activation service shares one transaction engine across Debian, Ubuntu, Kali and Arch. Each distro has an explicit profile with fixed tool paths, package versions and a tested boot layout. Physical machines and WSL remain disabled. This is a VM beta, not general workstation support.

| Profile | GRUB package | Root and boot | EFI mount | Package |
| --- | --- | --- | --- | --- |
| Debian 13 | `2.12-9+deb13u2` | ext4 root including `/boot` | `/boot/efi` | `.deb` |
| Ubuntu 24.04 | `2.12-1ubuntu7.3` | ext4 root, separate ext4 `/boot` | `/boot/efi` | `.deb` |
| Kali rolling | `2.14-2+kali1` | ext4 root including `/boot` | `/boot/efi` | `.deb` |
| Arch rolling | `2:2.16-1` | flat Btrfs root, subvolume ID 5, including `/boot` | `/efi` | `.pkg.tar.xz` |

All targets are x86-64, UEFI, Secure Boot disabled and conventional GRUB. The root-owned test marker and QEMU SMBIOS guard remain required. A derivative's `ID_LIKE` does not grant support. Untested package versions, layouts, snapshot integrations and conflicting bootloader configurations are refused.

## Distro differences

Debian, Ubuntu and Kali use `/usr/sbin/grub-mkconfig`; Arch uses `/usr/bin/grub-mkconfig`. All use `/usr/bin/grub-script-check`, `/etc/default/grub`, `/boot/grub/grub.cfg` and retained assets under `/boot/grub/themes/grubmgr`. Profiles select the generator; theme packages cannot supply a command or destination.

Inspection opens the fixed EFI directory before reading mount information. This lets Arch's existing automount activate when idle; grubmgr does not write EFI files or change mount configuration.

The helper takes dpkg's POSIX locks on apt-based systems. On Arch it takes pacman's exclusive `db.lck` using a hard link to a protected ownership file. After a killed helper, recovery removes that lock only when both paths still identify the same owned inode. An unknown lock is left untouched. No package-manager command runs from the helper.

Only `GRUB_THEME` changes during activation. Kernel options, default OS, timeout, firmware files and EFI variables stay under distro/administrator control. An earlier unmanaged theme cannot currently be a rollback target. Retained managed revisions support later rollback with current kernel entries.

Terminal commands register the distro's unprivileged `pkttyagent` before calling the fixed helper through `pkexec`. It waits for registration, preserves an existing authentication agent and sends passwords directly through the terminal. This avoids the built-in text-agent failure observed with Ubuntu's Polkit 124. The shipped policy still requires an administrator password; cancelled or rejected authentication cannot activate a theme.

Arch's Polkit 127 uses its packaged `polkit-agent-helper.socket`. The VM harness reboots after the initial full system update and checks that socket before testing the application. It does not add setuid permissions or change Polkit's service restrictions.

Cloud images normally use a text console and may hide the boot menu. The VM harness prepares a graphical test menu before installing grubmgr and records those fixture changes. The application does not silently change console or timeout settings.

## Themes and preview

Cyberpunk Demo is approved for all four profiles. The pinned Debian Starfield provider remains Debian-specific; expanding OS support does not approve arbitrary community assets. Imports, validation, immutable revisions, plans and history use the same interfaces on each distro.

Preview uses the separately installed `grub2-theme-preview` 2.10.0, Bubblewrap, QEMU, xorriso and mtools. Debian-family OVMF and Arch's edk2 firmware paths map into a private preview namespace. Missing dependencies or unavailable user namespaces produce a refusal; no unsandboxed fallback is used.

Ubuntu 24.04 also requires an AppArmor permission for grubmgr's user namespaces. The package provides an optional profile; it does not load it during installation. An administrator can enable it as part of preview dependency setup:

```sh
sudo install -o root -g root -m 0644 /usr/share/grubmgr/apparmor/usr.bin.grubmgr /etc/apparmor.d/usr.bin.grubmgr
sudo apparmor_parser -r /etc/apparmor.d/usr.bin.grubmgr
```

Inspect an existing profile before replacing it. The supplied profile attaches only to `/usr/bin/grubmgr`; it grants user namespaces to that application and its children. It is not a complete AppArmor confinement policy. Bubblewrap's mount, PID and network isolation still apply. No system-wide namespace setting or Bubblewrap profile changes. The VM test checks that the global restriction remains enabled and an unrelated direct Bubblewrap invocation still fails. See [Ubuntu's namespace policy](https://documentation.ubuntu.com/release-notes/24.04/#unprivileged-user-namespace-restrictions).

## Build and test

Build as an ordinary Linux user:

```sh
python3 scripts/build-deb.py --output outputs/linux
python3 scripts/check-deb.py outputs/linux/*.deb
python3 scripts/build-arch.py --deb outputs/linux/*.deb --output outputs/arch
python3 scripts/check-arch.py outputs/arch/*.pkg.tar.xz
```

The Arch archive reuses the identical verified Linux executables and notices from the Debian payload. It has pacman metadata, depends on `polkit`, and contains no installation scripts or bootloader hooks. Neither package installs an activation marker. Removing a package retains nonempty receipt and theme directories.

Run a fresh VM for each target with `scripts/vm/run-matrix.py TARGET --package PACKAGE --go /path/to/go --tools-root /path/to/qemu-tools`. The official images and SHA-256 pins are in `scripts/vm/targets.json`. Each run freezes its installer, guest scripts and test executable before boot. The tests cover administrator authentication, activation and reboot, rollback after adding a kernel and reboot, transaction failures, package locks, preview and package lifecycle. A `PASS` file is written only after every stage succeeds; inspection probes are not passing runs.

See the [verification report](linux-verification.md) for the executed runs and their limits. VM disks can be deleted after saving the reports and screenshots; they are not needed to build or use grubmgr.

## Sources

- [Ubuntu GRUB packages](https://packages.ubuntu.com/grub-common) and [official cloud images](https://cloud-images.ubuntu.com/noble/).
- [Kali GRUB package history](https://pkg.kali.org/news/688531/grub2-214-2kali1-migrated-to-kali-rolling/) and [official cloud images](https://kali.download/cloud-images/current/).
- [Arch GRUB documentation](https://wiki.archlinux.org/title/GRUB), [GRUB package files](https://archlinux.org/packages/core/x86_64/grub/files/), [OVMF package files](https://archlinux.org/packages/extra/any/edk2-ovmf/files/) and [official cloud images](https://geo.mirror.pkgbuild.com/images/latest/).
- [Polkit terminal-agent interface](https://github.com/polkit-org/polkit/blob/main/docs/man/pkttyagent.xml) and the [upstream built-in-agent fix](https://github.com/polkit-org/polkit/pull/423). No Polkit source is copied into grubmgr.
