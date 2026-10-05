# Testing and package builds

## Automated checks

Run from the repository root with Go 1.27.1:

```sh
go test ./...
go vet ./...
go mod verify
go run ./tools/checklicenses
test -z "$(gofmt -l cmd internal tools)"
go build ./cmd/...
```

On Linux, also run `go test -race ./...`. CI runs both platforms and builds the Linux packages. Ordinary tests use temporary fixtures and a local HTTPS server. The VM suite exercises the privileged helper and GRUB.

On Windows, use a short temporary path if tests reach the filesystem path limit:

```powershell
New-Item -ItemType Directory -Force work/test-tmp | Out-Null
$env:TEMP = Join-Path $PWD work/test-tmp
$env:TMP = $env:TEMP
go test ./...
```

Case-collision and symlink tests skip when the filesystem or account cannot create those inputs. Linux CI covers them.

## Fixture walkthrough

Copy a synthetic root before experimenting:

```sh
mkdir -p work
cp -R fixtures/debian work/demo-debian
go build -o grubmgr ./cmd/grubmgr
./grubmgr --root work/demo-debian fetch cyberpunk-demo
./grubmgr --root work/demo-debian validate cyberpunk-demo
./grubmgr --root work/demo-debian plan install cyberpunk-demo
./grubmgr --root work/demo-debian apply PLAN_TOKEN
./grubmgr --root work/demo-debian plan switch cyberpunk-demo
```

Replace `PLAN_TOKEN` with the complete `plan_id` and apply the switch plan separately. In PowerShell, use `Copy-Item -Recurse fixtures/debian work/demo-debian` and build `grubmgr.exe`. Keep the same `--root` for every command. Fixture generation edits synthetic files beneath that root.

### Fixture recovery

Focused recovery tests:

```sh
go test -v ./internal/transaction -run 'TestFailuresEveryPhase|TestSwitchFailuresRetainActiveAssets|TestInterruptedRecovery|TestRecoveryFailureAndDrift'
```

After killing a fixture process, confirm it has stopped before removing stale `.grubmgr/apply.lock` and `.grubmgr/state/operation.lock` files. Run `grubmgr --root ROOT recover`. Hash conflicts require inspection of the snapshots and competing files.

## Package builds

Use an ordinary Linux account with Go 1.27.1, Python 3.11+, Git and dpkg-deb:

```sh
python3 scripts/build-deb.py --output outputs/debian
python3 scripts/check-deb.py outputs/debian/*.deb
python3 scripts/build-arch.py --deb outputs/debian/*.deb --output outputs/arch
python3 scripts/check-arch.py outputs/arch/*.pkg.tar.xz
```

`--go PATH` selects the compiler for the Debian build. The Arch package uses the same executables and notices with pacman metadata. Both builds produce an adjacent checksum and file manifest. Archive inspection checks hashes, ownership, permissions, executable architecture and Polkit policy.

The build targets Linux amd64 with CGO disabled and uses the source commit timestamp unless `SOURCE_DATE_EPOCH` is set. It includes the user guides and dependency notices under `/usr/share/doc/grubmgr`.

## VM tests

Host requirements: Linux, Go, Python 3.11+, OpenSSH, QEMU system-x86 and qemu-img, OVMF, SeaBIOS, genisoimage, Git and dpkg-deb. Run as an ordinary user:

```sh
python3 scripts/vm/run.py
python3 scripts/vm/run-matrix.py ubuntu --package outputs/debian/grubmgr_0.3.0~rc.1-1_amd64.deb
python3 scripts/vm/run-matrix.py kali --package outputs/debian/grubmgr_0.3.0~rc.1-1_amd64.deb
python3 scripts/vm/run-matrix.py arch --package outputs/arch/grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz
```

The Debian harness accepts `--deb PATH`; otherwise it builds a package. Both harnesses accept `--go PATH` and `--tools-root DIRECTORY`. Existing packages need their adjacent `.manifest.json`. Image sources and hashes are pinned in `scripts/vm/run.py` and `scripts/vm/targets.json`.

Each run freezes its package, test binary and guest scripts, creates a fresh disk overlay, and tests:

- Installed files and administrator authentication, including cancellation and an incorrect password.
- Theme activation, reboot, variant switching and rollback after a kernel installation.
- Transaction interruptions, process termination, package locks, generator errors, stale plans and conflicting edits.
- Disk-full file replacement, preview, and package removal/reinstallation.

The Debian run also imports and activates Starfield. Tests built with `grubmgr_vmtest` use the guest identity check before mutation. A `PASS` file is written after all stages and the final input-hash check succeed. Logs and screenshots are saved under `work/vm-runs`.

The harness stops QEMU and removes runtime SSH credentials on exit. Once you have saved the logs you need, delete the stopped run's overlay, seed image and firmware-variable copy. Delete a cached base image only after removing overlays that depend on it.
