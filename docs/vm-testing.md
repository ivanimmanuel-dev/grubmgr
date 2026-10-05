# Disposable VM tests

For Ubuntu, Kali and Arch, use the [Linux matrix harness](linux-support.md). The Debian harness below remains available for regression tests.

The ordinary Go suite uses temporary fixtures. Real boot tests are separate and must run in the harness-created guest.

## Run

On a Linux development machine with Go 1.27.1, Python 3.11+, OpenSSH, QEMU system-x86/qemu-img, OVMF, SeaBIOS and genisoimage installed:

```sh
python3 scripts/vm/run.py
```

The harness is unprivileged on the host. `--go /path/to/go` selects a build tool; `--tools-root DIRECTORY` supports tools extracted beneath an alternate root. All guest disks, seed data and evidence are under a newly created `work/vm-runs/debian13-*` directory. The backing Debian image is read-only. No host device or shared writable host directory is attached.

The downloaded Debian image must match the committed SHA-512. If the upstream `latest` image changes, the harness refuses it. Review and update the pin or supply the matching cached image; never skip the check. Dependency packages in the guest come from Debian repositories, while the backend refuses an untested GRUB version.

Each run creates a new overlay and private firmware variables. This is the clean-state/reset mechanism; previous runs remain available for investigation. SSH keys live in a temporary Linux directory. The guest host key comes from the QEMU console, or from the first SSH connection after verifying that this QEMU process owns its loopback listener. Subsequent connections enforce that key. The only forwarded port binds to loopback.

The harness builds one [Debian package](debian-package.md) and a separate test binary before boot. Alternatively, pass `--deb PATH` with its adjacent `.manifest.json`. The package, test binary and guest scripts are frozen in the run directory and hashed; they are never rebuilt or refreshed during the run.

It installs the package **inside the VM**, verifies installed hashes and confirms installation alone does not enable activation. A local virtual-console login on seat0 exercises the shipped Polkit password policy with cancellation, wrong and correct passwords. Then it runs the synthetic workflow, reboots, installs an additional kernel, rolls back, reboots, injects failures, imports/renders/activates Starfield and reboots again. Finally, package reinstall/remove/purge/reinstall must preserve boot files, assets and receipts. `PASS` is written only after all stages and the final input-hash check succeed.

The guest test authorization rule is created only after the production-policy password checks. It is scoped to its tester account and fixed helper action and is never included in the package. The temporary test password and console autologin are removed before the transaction suite. Graphical desktop agents remain untested.

## Failure coverage

The VM-only binary is built with `go test -c -tags grubmgr_vmtest ./internal/debian`. Never run it on a workstation. It checks the disposable-VM guard before mutations. No shipped CLI/helper flag enables its failure hooks.

- Injected failure after each transaction phase, including the receipt commit boundary.
- Interrupted journals at every pre-commit boundary, followed by exact-byte recovery.
- Actual SIGKILL after settings staging and activation, with kernel-released locks and recovery.
- A real distro-generator error, stale plans and a conflicting manual defaults edit.
- Disk-full replacement on a private 1 MiB guest tmpfs: the old file remains intact. This is an atomic-write test, not a claim of full-system ENOSPC recovery coverage.
- A new installed kernel between activation and later rollback; both entries must remain in regenerated GRUB.

Power loss, every syscall interruption, physical hardware and broad distribution compatibility are outside this test claim. See [Phase 2 verification](phase2-verification.md) for the actual run and any outstanding gates.
