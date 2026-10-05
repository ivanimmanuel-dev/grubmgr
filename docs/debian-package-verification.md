# Debian package verification

Completed 2026-10-05. The installer is ready for the documented disposable Debian VM. Physical-machine activation remains disabled.

## Tested artifact

- File: `grubmgr_0.2.0~rc.1-1_amd64.deb` (CLI version `0.2.0-rc.1`).
- SHA-256: `cbf4725f1275a49581ac3334b4221bd415e6753e8fd21a027131044548d3a427`.
- Two builds from the same inputs produced identical bytes.
- Archive inspection verified 40 installed files, their hashes, ownership/modes, amd64 executables and the strict Polkit policy. No maintainer scripts, VM marker, authorization bypass or boot settings are packaged.

This record applies to that exact archive. CI rebuilds are separate artifacts; their checksums and manifests accompany the download.

## Fresh VM run

Run `debian13-862yc7dh` used a new overlay from the pinned Debian image. The package, VM test binary and guest scripts were frozen before boot. No binaries or scripts were replaced during the run. The final input-hash check passed, evidence was exported, `PASS` was written and the VM powered off. The harness exited successfully.

Environment: Debian 13.7 amd64, GRUB 2.12-9+deb13u2, Q35/OVMF UEFI with Secure Boot disabled, writable ext4 root including `/boot`, FAT EFI partition. Host and Windows/WSL boot configuration were untouched.

| Check | Result |
| --- | --- |
| Package installation | All 40 file hashes matched; boot files unchanged; no activation marker created |
| Production authorization | Real local login on `seat0`, `Remote=no`, `Active=yes`; shipped `auth_admin` policy |
| Cancel password prompt | Interrupted, exit -2; boot files unchanged |
| Wrong password | Rejected, exit 6; boot files unchanged |
| Correct password | Helper authorized, exit 0; supported-VM inspection returned |
| Synthetic theme | Installed, activated, rebooted, switched to HD variant |
| Later rollback | Regenerated after installing another kernel; both kernel entries retained; reboot succeeded |
| Failure tests | All 26 named cases passed, including subtests |
| Starfield | Imported, validated, rendered in QEMU, installed, activated and rebooted |
| Package lifecycle | Reinstall, remove, purge and reinstall preserved boot files, theme assets and receipts |
| Final state | Reinstalled app reported Starfield active; all installed file hashes matched again |

The password test uses Polkit's terminal agent, not a graphical desktop agent. Its temporary password and console autologin were removed before the transaction suite. The remaining unattended commands used the scoped VM-only authorization rule, which is not in the package.

Earlier harness attempts exposed a permissions check in the wrong user context and a serial session that the production policy correctly refused. Both test issues were resolved before this fresh run; the application policy was not relaxed.

## Boot evidence

| State | Linux boot ID |
| --- | --- |
| Initial activation | `00d741d7-9df7-4a45-add6-081d6a2247d9` |
| After activation reboot | `22aff836-84d5-40ba-9e84-698837b7a493` |
| After rollback reboot | `f5158764-6102-472c-b532-f666cd63e78f` |
| After Starfield reboot | `1aaedc95-af3c-4c37-8440-1d075d261755` |

Rollback retained `vmlinuz-6.12.111+deb13-amd64` and `vmlinuz-6.12.111+deb13-cloud-amd64`. As in the earlier run, the VM booted the cloud kernel; the added kernel was verified as a menu entry. The final 1024×768 Starfield preview was visually inspected.

Evidence is retained locally in `outputs/debian/verification`: authentication transcripts, package command logs, installed-file checks, boot snapshots, failure log, preview PNG, console log, frozen-input hashes and `summary.json`. VM disks and credentials are not uploaded. See [package instructions](debian-package.md), [the VM harness](vm-testing.md) and [earlier backend verification](phase2-verification.md) for commands and remaining support limits.
