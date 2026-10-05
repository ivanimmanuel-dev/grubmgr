# Installation

## Download and install

Download the package for your system from [GRUB Manager 0.3.0-rc.1](https://github.com/ivanimmanuel-dev/grubmgr/releases/tag/v0.3.0-rc.1):

| System | File |
| --- | --- |
| Debian, Ubuntu or Kali · x86-64 | `grubmgr_0.3.0~rc.1-1_amd64.deb` |
| Arch · x86-64 | `grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz` |

Download its matching `.sha256` file into the same directory. Check the download before installing:

```sh
sha256sum -c grubmgr_0.3.0~rc.1-1_amd64.deb.sha256
sudo apt install ./grubmgr_0.3.0~rc.1-1_amd64.deb
```

On Arch:

```sh
sha256sum -c grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz.sha256
sudo pacman -U ./grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz
```

Then run `grubmgr doctor`. Use the CLI as your ordinary user; Polkit prompts for administrator authentication when needed. Installing the package leaves the selected boot theme unchanged.

## Supported configurations

This version activates themes in x86-64 QEMU VMs with UEFI and Secure Boot disabled:

| Distribution | GRUB package version | Root and boot layout | EFI mount |
| --- | --- | --- | --- |
| Debian 13 | `2.12-9+deb13u2` | ext4 root including `/boot` | `/boot/efi` |
| Ubuntu 24.04 | `2.12-1ubuntu7.3` | ext4 root, separate ext4 `/boot` | `/boot/efi` |
| Kali rolling | `2.14-2+kali1` | ext4 root including `/boot` | `/boot/efi` |
| Arch rolling | `2:2.16-1` | Btrfs root, subvolume ID 5, including `/boot` | `/efi` |

Activation requires a guest created by the included [VM setup scripts](https://github.com/ivanimmanuel-dev/grubmgr/tree/v0.3.0-rc.1/scripts/vm). Installing the package alone does not enable activation on an existing VM. Physical-machine and WSL activation are disabled.

Import and validation work independently of activation. Preview requires Linux and the tools below.

## Preview setup

Install the system tools on Debian, Ubuntu or Kali:

```sh
sudo apt install grub-efi-amd64-bin grub2-common bubblewrap qemu-system-x86 ovmf xorriso mtools python3-venv
```

On Arch:

```sh
sudo pacman -S --needed grub bubblewrap qemu-system-x86 edk2-ovmf libisoburn mtools python python-pip
```

Install the preview renderer in its own Python environment:

```sh
sudo python3 -m venv /usr/local/lib/grub2-theme-preview
sudo /usr/local/lib/grub2-theme-preview/bin/python -m pip install 'grub2-theme-preview==2.10.0'
sudo ln -s /usr/local/lib/grub2-theme-preview/bin/grub2-theme-preview /usr/local/bin/grub2-theme-preview
```

If `/usr/local/bin/grub2-theme-preview` already exists, check that it points to this environment before changing it. Both the renderer and its Python environment must live under `/usr/local` so the preview can access them.

### Ubuntu 24.04

Enable the supplied AppArmor profile to allow the application's user namespaces. Check for an existing `/etc/apparmor.d/usr.bin.grubmgr` before replacing it:

```sh
sudo install -o root -g root -m 0644 /usr/share/grubmgr/apparmor/usr.bin.grubmgr /etc/apparmor.d/usr.bin.grubmgr
sudo apparmor_parser -r /etc/apparmor.d/usr.bin.grubmgr
```

The profile grants user namespaces to `/usr/bin/grubmgr` and its children. Other applications retain Ubuntu's namespace restrictions.

### Try a preview

```sh
grubmgr fetch cyberpunk-demo
grubmgr preview cyberpunk-demo
```

The command prints the image path. If it fails, open the reported `preview.log` for the missing tool or permission error.

## Uninstall

Debian, Ubuntu or Kali:

```sh
sudo apt remove grubmgr
```

Arch:

```sh
sudo pacman -R grubmgr
```

Theme files and application records remain on disk so GRUB can read the selected theme. Reinstallation uses those records. The separately installed preview tools and optional AppArmor profile remain until you remove them.
