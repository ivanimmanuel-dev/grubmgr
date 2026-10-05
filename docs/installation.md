# Installation

Linux packages include the CLI, activation helper, preview adapter and Polkit policy. Installation leaves the selected boot theme unchanged.

## Supported configurations

Activation requires an x86-64 QEMU VM with UEFI, Secure Boot disabled, and one of these configurations:

| Distribution | GRUB package version | Root and boot layout | EFI mount |
| --- | --- | --- | --- |
| Debian 13 | `2.12-9+deb13u2` | ext4 root including `/boot` | `/boot/efi` |
| Ubuntu 24.04 | `2.12-1ubuntu7.3` | ext4 root, separate ext4 `/boot` | `/boot/efi` |
| Kali rolling | `2.14-2+kali1` | ext4 root including `/boot` | `/boot/efi` |
| Arch rolling | `2:2.16-1` | Btrfs root, subvolume ID 5, including `/boot` | `/efi` |

The [VM test harness](testing.md#vm-tests) provisions the required machine identity and activation marker. Installing a package alone does not enable activation on an existing VM. `grubmgr doctor` reports the detected profile and any configuration mismatch.

Imports and validation also work on Windows and other Linux installations. Preview requires Linux. Physical-machine and WSL activation are disabled.

## Packages

Download and extract an artifact from a successful [GitHub Actions run](https://github.com/ivanimmanuel-dev/grubmgr/actions/workflows/ci.yml):

| Distribution | Artifact | Install |
| --- | --- | --- |
| Debian, Ubuntu, Kali | `grubmgr-debian-amd64` | `sudo apt install ./grubmgr_0.3.0~rc.1-1_amd64.deb` |
| Arch | `grubmgr-arch-x86_64` | `sudo pacman -U ./grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz` |

Each artifact contains a package, SHA-256 checksum and file manifest. Run `grubmgr doctor` after installation. Use the CLI as your ordinary user; operations that access managed system state prompt through Polkit.

## Build from source

With Go 1.27.1 installed:

```sh
go build -o grubmgr ./cmd/grubmgr
./grubmgr doctor
```

On Windows, use `go build -o grubmgr.exe ./cmd/grubmgr` and run `./grubmgr.exe`. The standalone binary supports import, validation and [fixture testing](testing.md#fixture-walkthrough). Activation and preview also need the packaged helpers. See [package builds](testing.md#package-builds).

## Preview setup

Preview requires `grub2-theme-preview` 2.10.0 at `/usr/local/bin/grub2-theme-preview`, Bubblewrap, QEMU, xorriso, mtools and UEFI firmware. The renderer's [installation instructions](https://github.com/hartwork/grub2-theme-preview#installation) cover its Python package.

| System | Distro packages |
| --- | --- |
| Debian, Ubuntu, Kali | `grub-efi-amd64-bin`, `grub2-common`, `bubblewrap`, `qemu-system-x86`, `ovmf`, `xorriso`, `mtools` |
| Arch | `grub`, `bubblewrap`, `qemu-system-x86`, `edk2-ovmf`, `libisoburn`, `mtools` |

Ubuntu 24.04 requires permission for the application's user namespaces. To enable the supplied AppArmor profile, first check for an existing `/etc/apparmor.d/usr.bin.grubmgr`, then install and load the profile:

```sh
sudo install -o root -g root -m 0644 /usr/share/grubmgr/apparmor/usr.bin.grubmgr /etc/apparmor.d/usr.bin.grubmgr
sudo apparmor_parser -r /etc/apparmor.d/usr.bin.grubmgr
```

The profile grants user namespaces to `/usr/bin/grubmgr` and its children. Bubblewrap isolates the preview's files, processes and network. Other applications remain subject to [Ubuntu's namespace restrictions](https://documentation.ubuntu.com/release-notes/24.04/#unprivileged-user-namespace-restrictions).

## Uninstall

```sh
# Debian, Ubuntu or Kali
sudo apt remove grubmgr

# Arch
sudo pacman -R grubmgr
```

Theme files and nonempty application state remain on disk so GRUB can still read the selected theme. Reinstallation uses those existing records. If you enabled the optional AppArmor profile, it remains in `/etc/apparmor.d` until you unload and remove it.
