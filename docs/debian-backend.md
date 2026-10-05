# Debian profile and recovery

The `debian13-uefi-vm` profile supports this disposable VM configuration:

| Item | Tested value |
| --- | --- |
| OS and architecture | Debian 13.7, amd64 |
| GRUB packages | `grub-common` and `grub2-common` 2.12-9+deb13u2 |
| Machine | QEMU Q35, TCG, 2 CPUs, 2 GiB RAM |
| Firmware | OVMF UEFI; Secure Boot disabled |
| Storage | Writable ext4 root including `/boot`; FAT `/boot/efi` |
| Defaults | `/etc/default/grub`; fragments must not override `GRUB_THEME` |
| Generated configuration | `/boot/grub/grub.cfg` |
| Theme assets | `/boot/grub/themes/grubmgr/NAMESPACE/NAME/REVISION` |
| Candidate generation | `/usr/sbin/grub-mkconfig -o /boot/grub/.grubmgr-TRANSACTION.cfg` |
| Candidate check | `/usr/bin/grub-script-check CANDIDATE` |
| Helper | `/usr/libexec/grubmgr-helper`, authorized by Polkit |
| State | `/var/lib/grubmgr/{data,state,cache}` |

`doctor` reports `SUPPORTED WITH WARNINGS` for this experimental profile, `UNSUPPORTED` for an untested configuration, or `AMBIGUOUS` for conflicting boot layouts. The helper checks the VM identity before mutation.

Activation preserves unrelated defaults byte-for-byte. Debian's generator reads the staged theme assignment, installed GRUB scripts and current kernel/initrd files, then writes a separate candidate using its documented [output option](https://manpages.debian.org/trixie/grub-common/grub-mkconfig.8.en.html).

## Starfield import

`debian/starfield` imports the installed [grub-theme-starfield](https://packages.debian.org/trixie/grub-theme-starfield) 2.12-9+deb13u2 data package. The installed theme directory is the explicit root. Each imported revision preserves `theme.txt`, `README`, `DEBIAN-COPYRIGHT`, `FONT-COPYRIGHT` and `GPL-3`.

The catalog and helper pin the normalized file-inventory SHA-256 to `b85ca5d4edb4af357c3ffa67bdf4dc74e43c436d94354ea2c714c63970cc4739`. For this provider, the artifact digest covers the imported directory and notices. A changed package or notice file requires a recipe review.

The preserved notices identify the layout as MIT, artwork as CC-BY-SA-3.0, and DejaVu font data as Bitstream-Vera with public-domain changes. Starfield assets are imported from Debian and retain those terms.

## Recovery

After an interrupted operation, run `grubmgr recover` as the ordinary guest user. The helper reacquires locks and verifies recovery hashes. If either target has an external edit, retain the journal, inspect the competing files and resolve the conflict before retrying.

For a later rollback, run `grubmgr plan rollback TRANSACTION_ID`, review the plan, then run `grubmgr apply TOKEN`. This restores the previous managed theme selection and regenerates configuration with the current kernels. Immediate transaction recovery uses the recorded snapshots.

Physical Debian machines, WSL, BIOS, Secure Boot enabled, separate `/boot`, non-ext4 roots and customizer/snapshot integrations are outside this profile. Ubuntu, Kali and Arch have distinct [Linux profiles](linux-support.md).
