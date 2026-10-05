# Disposable VM tests

Real boot tests run in harness-created QEMU guests. The ordinary Go suite uses synthetic filesystem fixtures. Ubuntu, Kali and Arch use the [Linux matrix harness](linux-support.md#build-and-test); the Debian workflow is described below.

## Run

Host requirements: Linux, Go 1.27.1, Python 3.11+, OpenSSH, QEMU system-x86 and qemu-img, OVMF, SeaBIOS, genisoimage, Git and dpkg-deb.

```sh
python3 scripts/vm/run.py
```

Run as an ordinary user. `--go PATH` selects the compiler, `--tools-root DIRECTORY` selects extracted QEMU tools, and `--deb PATH` selects an existing package with its adjacent `.manifest.json`. Otherwise, the harness builds a package before boot.

Each run creates a directory under `work/vm-runs/debian13-*` containing a fresh disk overlay, seed data, private firmware variables and evidence. The backing image is read-only and must match the committed SHA-512. A changed upstream image requires a reviewed pin update or the matching cached image.

SSH credentials use temporary Linux storage. The guest host key is obtained from the QEMU console or from the first connection after verifying that QEMU owns the loopback listener. Later connections enforce that key. Host disks and shared writable directories are excluded; the forwarded SSH port binds to loopback.

## Workflow

The package, test binary and guest scripts are frozen and hashed before boot. The harness then:

1. Installs the package and verifies installed hashes, unchanged boot files and the absence of an activation marker.
2. Provisions the test identity and uses a local virtual-console login to test Polkit cancellation, a wrong password and successful authentication.
3. Removes the temporary password and console autologin, then enables the tester-only rule for unattended transaction tests.
4. Installs and activates the demo theme, reboots, switches variants, adds a kernel, rolls back and reboots again.
5. Runs transaction failure and recovery tests.
6. Imports, previews and activates Starfield, then reboots.
7. Reinstalls, removes, purges and reinstalls the application, checking that boot files, assets and receipts are retained.

`PASS` is written after all stages and the final input-hash check succeed. The test authorization rule is confined to the guest and excluded from packages. Authentication coverage uses the terminal agent; graphical agents require separate testing.

## Failure coverage

The harness builds the test executable with `go test -c -tags grubmgr_vmtest ./internal/debian`. Its VM guard is required before mutations; failure hooks are excluded from shipped binaries.

- Failure after each transaction phase, including the receipt commit boundary.
- Interruption at every pre-commit journal boundary, followed by exact-byte recovery.
- SIGKILL after settings staging and activation, with lock recovery.
- Generator failure, stale plans, external edits and ownership/permission conflicts.
- Disk-full replacement on a private 1 MiB tmpfs, preserving the old file.
- Rollback after a kernel update, retaining both entries in generated configuration.

These tests cover the specified VM workflows and failure points. Physical hardware, arbitrary power loss and exhaustion of the entire boot filesystem are outside the executed scope. See [Debian package verification](debian-package-verification.md) and [Linux VM verification](linux-verification.md).

## Cleanup

The harness stops its guest and removes runtime SSH credentials on exit. Run directories and cached images are retained for investigation. After exporting the reports and screenshots and confirming QEMU has stopped, remove the run's disk overlay, seed image and firmware-variable copy. Cached base images can also be removed when no retained overlay depends on them.
