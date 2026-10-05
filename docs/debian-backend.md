# Debian backend

Activation is experimental and restricted to the disposable test VM. Do not install the VM marker or test authorization rule on a workstation.

| Item | Tested value |
| --- | --- |
| OS / architecture | Debian 13.7, amd64 |
| GRUB | grub-common and grub2-common 2.12-9+deb13u2 |
| Machine | QEMU Q35, TCG, 2 CPUs, 2 GiB RAM |
| Firmware | OVMF UEFI; Secure Boot variable reports disabled |
| Storage | Writable ext4 root including `/boot`; FAT `/boot/efi` |
| Defaults | `/etc/default/grub`; root-owned fragments may not override GRUB_THEME |
| Generated configuration | `/boot/grub/grub.cfg` |
| Theme assets | `/boot/grub/themes/grubmgr/NAMESPACE/NAME/REVISION` |
| Candidate command | `/usr/sbin/grub-mkconfig -o /boot/grub/.grubmgr-TRANSACTION.cfg` |
| Candidate check | `/usr/bin/grub-script-check CANDIDATE` |
| Helper | `/usr/libexec/grubmgr-helper`, invoked through Polkit |
| State | `/var/lib/grubmgr/{data,state,cache}` |

`doctor` reports SUPPORTED, SUPPORTED WITH WARNINGS, UNSUPPORTED or AMBIGUOUS. This experimental target always carries a warning. Conflicting boot layouts, unsupported versions and uncertain theme assignments cause refusal. The VM check precedes real mutation.

The backend preserves unrelated defaults byte-for-byte. Candidate generation consumes the installed Debian scripts and current kernel/initrd files. It never uses an invented alternate-defaults flag. Debian documents the supported output option in [grub-mkconfig(8)](https://manpages.debian.org/trixie/grub-common/grub-mkconfig.8.en.html).

## Real theme

`debian/starfield` imports the installed `grub-theme-starfield` 2.12-9+deb13u2 data package. It does not invoke apt or package scripts. The canonical package is [Debian Starfield](https://packages.debian.org/trixie/grub-theme-starfield). Its installed directory is the explicit theme root; the upstream theme.txt, README, Debian copyright, font copyright and GPL text are preserved.

The catalog pins the normalized file-inventory SHA-256 to `b85ca5d4edb4af357c3ffa67bdf4dc74e43c436d94354ea2c714c63970cc4739`. For this provider, artifact identity means the imported directory plus notices, not the hash of a .deb archive. The helper pins the same tree. A changed Debian build or notice file requires a reviewed recipe update.

Theme layout is MIT, artwork CC-BY-SA-3.0 and bundled DejaVu font data Bitstream-Vera/public-domain changes, according to the preserved package notices. Assets are not relicensed as application code or bundled in this repository. Grub of Tsushima remains deferred because its Sony artwork and third-party font permissions are unconfirmed.

## Recovery

After an interrupted operation, run `grubmgr recover` as the ordinary guest user. The helper reacquires locks and verifies recovery hashes. If an administrator changed either target, keep the journal, inspect the competing files and resolve the conflict before retrying. Do not delete receipts to suppress a recovery error.

Later user rollback uses `grubmgr plan rollback TRANSACTION`, followed by `grubmgr apply TOKEN`. It selects the previous retained theme and regenerates against current kernels. Old snapshots are used only for immediate transaction recovery.

Ubuntu, Arch, Fedora, physical Debian machines, WSL activation, BIOS, Secure Boot enabled, separate boot mounts, non-ext4 roots, customizer and snapshot integrations are not supported.
