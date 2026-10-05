# Linux support

GRUB Manager supports the following experimental QEMU VM profiles. Each defines tool paths, package versions and a tested boot layout. Physical-machine and WSL activation are disabled.

| Profile | GRUB package | Root and boot | EFI mount | Package |
| --- | --- | --- | --- | --- |
| Debian 13 | `2.12-9+deb13u2` | ext4 root including `/boot` | `/boot/efi` | `.deb` |
| Ubuntu 24.04 | `2.12-1ubuntu7.3` | ext4 root, separate ext4 `/boot` | `/boot/efi` | `.deb` |
| Kali rolling | `2.14-2+kali1` | ext4 root including `/boot` | `/boot/efi` | `.deb` |
| Arch rolling | `2:2.16-1` | flat Btrfs root, subvolume ID 5, including `/boot` | `/efi` | `.pkg.tar.xz` |

All targets are x86-64, UEFI, Secure Boot disabled and conventional GRUB. Activation also requires the root-owned test marker and matching QEMU SMBIOS identity. Derivatives, other package versions and layouts, snapshot integrations and conflicting bootloader configurations require separate profiles.

## Distro differences

Debian, Ubuntu and Kali use `/usr/sbin/grub-mkconfig`; Arch uses `/usr/bin/grub-mkconfig`. All use `/usr/bin/grub-script-check`, `/etc/default/grub`, `/boot/grub/grub.cfg` and retained assets under `/boot/grub/themes/grubmgr`. The profile selects the generator and destinations.

Inspection opens the fixed EFI directory before reading mount information. This activates Arch's existing EFI automount when idle.

The helper takes dpkg's POSIX locks on apt-based systems. On Arch it takes pacman's exclusive `db.lck` using a hard link to a protected ownership file. After a killed helper, recovery removes that lock only when both paths still identify the same owned inode. Foreign locks remain untouched.

Only `GRUB_THEME` changes during activation. Kernel options, default OS, timeout, firmware files and EFI variables stay under distro/administrator control. An earlier unmanaged theme cannot currently be a rollback target. Retained managed revisions support later rollback with current kernel entries.

Terminal commands register the distro's unprivileged `pkttyagent` before calling the fixed helper through `pkexec`. It waits for registration, preserves an existing authentication agent and sends passwords directly through the terminal. This avoids the built-in text-agent failure observed with Ubuntu's Polkit 124. The shipped policy requires an administrator password.

Arch's Polkit 127 uses its packaged `polkit-agent-helper.socket`. The VM harness reboots after the initial full system update and checks that socket before testing the application.

Cloud images normally use a text console and may hide the boot menu. The VM harness prepares a graphical test menu before installing grubmgr and records those fixture changes.

## Themes and preview

Cyberpunk Demo is approved for all four profiles. The pinned Starfield import is Debian-specific. Other community themes need individual recipes and asset review. Imports, validation, immutable revisions, plans and history use the same interfaces on each distro.

Preview uses the separately installed `grub2-theme-preview` 2.10.0, Bubblewrap, QEMU, xorriso and mtools. Debian-family OVMF and Arch's edk2 firmware paths map into a private preview namespace. Preview requires all dependencies and working user namespaces.

Ubuntu 24.04 also requires an AppArmor permission for grubmgr's user namespaces. The package provides an optional profile; it does not load it during installation. An administrator can enable it as part of preview dependency setup:

```sh
sudo install -o root -g root -m 0644 /usr/share/grubmgr/apparmor/usr.bin.grubmgr /etc/apparmor.d/usr.bin.grubmgr
sudo apparmor_parser -r /etc/apparmor.d/usr.bin.grubmgr
```

Inspect any existing profile before installing the supplied file. This profile grants user namespaces to `/usr/bin/grubmgr` and its children; Bubblewrap provides mount, PID and network isolation. System-wide namespace restrictions remain enabled. The VM test also checks that an unrelated Bubblewrap invocation is still denied. See [Ubuntu's namespace policy](https://documentation.ubuntu.com/release-notes/24.04/#unprivileged-user-namespace-restrictions).

## Build and test

Build as an ordinary Linux user:

```sh
python3 scripts/build-deb.py --output outputs/linux
python3 scripts/check-deb.py outputs/linux/*.deb
python3 scripts/build-arch.py --deb outputs/linux/*.deb --output outputs/arch
python3 scripts/check-arch.py outputs/arch/*.pkg.tar.xz
```

The Arch archive uses the same Linux executables and notices as the Debian payload. It adds pacman metadata and depends on `polkit`. In a supported Arch test VM, install it with `sudo pacman -U outputs/arch/grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz`. Both formats leave boot configuration unchanged and retain nonempty receipt and theme directories on removal.

Run a fresh VM for each target with `scripts/vm/run-matrix.py TARGET --package PACKAGE --go /path/to/go --tools-root /path/to/qemu-tools`. The official images and SHA-256 pins are in `scripts/vm/targets.json`. Each run freezes its installer, guest scripts and test executable before boot. The tests cover administrator authentication, activation and reboot, rollback after adding a kernel and reboot, transaction failures, package locks, preview and package lifecycle. A `PASS` file records completion of all stages.

See [verification](linux-verification.md) for executed runs and [VM cleanup](vm-testing.md#cleanup) for removing disks after saving evidence.

## Sources

- [Ubuntu GRUB packages](https://packages.ubuntu.com/grub-common) and [official cloud images](https://cloud-images.ubuntu.com/noble/).
- [Kali GRUB package history](https://pkg.kali.org/news/688531/grub2-214-2kali1-migrated-to-kali-rolling/) and [official cloud images](https://kali.download/cloud-images/current/).
- [Arch GRUB documentation](https://wiki.archlinux.org/title/GRUB), [GRUB package files](https://archlinux.org/packages/core/x86_64/grub/files/), [OVMF package files](https://archlinux.org/packages/extra/any/edk2-ovmf/files/) and [official cloud images](https://geo.mirror.pkgbuild.com/images/latest/).
- [Polkit terminal-agent interface](https://github.com/polkit-org/polkit/blob/main/docs/man/pkttyagent.xml) and the [upstream built-in-agent fix](https://github.com/polkit-org/polkit/pull/423).
