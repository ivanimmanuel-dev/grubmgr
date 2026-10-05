# Phase 2 verification

Verified on 2026-10-05. `v0.2.0-rc.1` is ready for experimental use in the specified disposable VM. It is not a physical-machine release.

A subsequent [Debian package run](debian-package-verification.md) completed with frozen inputs from start to finish and tested the production Polkit terminal password prompt. The earlier development-run details below remain as historical evidence.

## Implementation and environment

The fixture engine remains intact. A separate Debian backend follows the same seven transaction phases, using a fixed Polkit helper, root-owned state, package-manager locks, independently checked package pins and plans, and ownership/path checks. Only `GRUB_THEME` changes in defaults. Debian generates a separate candidate, checks it, then activates it by synced replacement. Immediate recovery restores verified snapshots; later rollback regenerates with current kernels. Failed operations retain assets and recovery material.

| Component | Tested configuration |
| --- | --- |
| Guest | Debian 13.7 amd64; QEMU Q35, TCG, 2 CPUs, 2 GiB RAM |
| GRUB | grub-common, grub2-common and grub-theme-starfield 2.12-9+deb13u2 |
| Firmware | OVMF UEFI, private variables, Secure Boot reported disabled |
| Disk | 16 GiB disposable overlay; ext4 root includes `/boot`; FAT `/boot/efi` |
| Defaults / generated config | `/etc/default/grub`, `/boot/grub/grub.cfg` |
| Managed assets | `/boot/grub/themes/grubmgr/NAMESPACE/NAME/REVISION` |
| Host virtualization | QEMU 10.2.1; OVMF 2025.11-3ubuntu7.2 |
| Guest preview | QEMU 10.0.13+ds-0+deb13u1; OVMF 2025.02-8+deb13u1 |
| Preview tools | grub2-theme-preview 2.10.0; Bubblewrap 0.12.0-1~deb13u1; xorriso 1.5.6-1.2+b1; mtools 4.0.48-1 |
| Authorization | pkexec/polkitd 126-2; test-only rule for the guest user and fixed action |
| Builds | Go 1.27.1; Windows amd64 and Linux amd64 |

The Debian image SHA-512 is pinned in `scripts/vm/run.py` and was checked against Debian's published checksums. All GRUB generation and boot changes occurred inside the disposable guest. Windows and WSL boot configuration were not modified. Host tools were extracted into ignored `work/` directories.

## Commands and ordinary tests

The unchanged baseline passed 83 named Windows cases and 86 Linux cases. The final ordinary suite passed 91 Windows cases, with three filesystem-capability skips, and 94 Linux cases including race detection. Counts include named subtests. Windows skipped case-collision and two symlink checks; Linux exercised them.

Executed with the workspace toolchains:

```sh
go test -json ./...
go test -race -json ./...  # Linux
go vet ./...
go mod verify
go run ./tools/checklicenses
go build ./cmd/...
gofmt -l cmd internal tools
python3 -m py_compile scripts/vm/run.py scripts/vm/guest-flow.py
```

Formatting, vet, module verification, license checks, builds and Python compilation passed. Normal Actions retains unit/fixture tests and builds all three commands; it does not run real boot tests. This report records local checks, not a remote Actions result.

The separate harness ran as an ordinary WSL user:

```sh
python3 -u scripts/vm/run.py \
  --go "$PWD/work/linux-toolchain/go/bin/go" \
  --tools-root "$PWD/work/vm-tools/root"
```

Guest commands included the following, using `--json` and returned identifiers:

```text
grubmgr doctor
grubmgr fetch cyberpunk-demo
grubmgr validate cyberpunk-demo
grubmgr plan install grubmgr/cyberpunk-demo
grubmgr apply <plan>
grubmgr plan switch grubmgr/cyberpunk-demo
grubmgr apply <plan>
grubmgr plan switch grubmgr/cyberpunk-demo --variant hd
grubmgr apply <plan>
sudo apt-get install -y --no-install-recommends linux-image-amd64
grubmgr plan rollback <HD-switch-transaction>
grubmgr apply <plan>
grubmgr info starfield
grubmgr fetch starfield
grubmgr validate starfield
grubmgr preview starfield
grubmgr plan install debian/starfield
grubmgr apply <plan>
grubmgr plan switch debian/starfield
grubmgr apply <plan>
```

Inside the guest, the helper ran `/usr/sbin/grub-mkconfig -o /boot/grub/.grubmgr-TRANSACTION.cfg` and `/usr/bin/grub-script-check CANDIDATE`. No `grub-install` was run. Apt and test setup were separate provisioning operations, not helper features.

## Reboots and rollback

Run `debian13-b7ual0_p` completed all stages, exported evidence, wrote `PASS`, powered off and exited successfully. Each reboot reached SSH and completed its post-boot assertions:

| State | Linux boot ID |
| --- | --- |
| Initial activation | `9d64827c-a651-4cf1-9964-23dd768f2424` |
| Synthetic-theme reboot | `60a9202e-0d01-47db-8adb-29d7284d6f4b` |
| Later rollback reboot | `84a1fa0a-56ce-4c10-b52b-b4c2972c477f` |
| Starfield reboot | `0362c232-ab47-4a9c-906b-3906e627dfeb` |

The kernel update added `6.12.111+deb13-amd64` alongside `6.12.111+deb13-cloud-amd64`. Later rollback selected the default theme after the HD variant and asserted that both kernel filenames remained in the newly generated configuration. The VM continued booting the cloud kernel; both kernels were not individually boot-tested.

Rollback generated configuration SHA-256 `fe59699b21627f4d22896bfb653940b2c586385270d93daaa9fe9b98a1cf1cc7`, distinct from the initial `d7ba72e9d57ace53d6ed9c7be638a32e02d0c7b0512f185107d91d06adabd5c4`. It remained unchanged across reboot. Starfield's hash `a51948c997000dee8e54e5e69d1bc59fdd6c6b9af9d874977fde4ac95d03668d` also matched before and after reboot.

This run began with a fresh overlay. During development its initial SSH bootstrap was repaired, and binaries were refreshed before the rollback/failure and Starfield stages. The committed harness contains that fix. This is completed development-run evidence, not a claim that one frozen archive completed every stage without intervention.

## Failure checks

The guest test binary passed 26 named cases, including subtests:

- Strict requests, content pins, settings preservation and token parsing.
- Failures after all seven phases; journal interruption/recovery at all six pre-commit phases.
- Actual SIGKILL after settings staging and activation; an actual Debian generator error.
- Stale plans and manual defaults edits. Conflicting recovery retained `recovery_required` until the conflict was resolved.
- A failed later rollback restored its immediate predecessor byte-for-byte; a fresh rollback then succeeded.
- Unsafe destination-parent and individual asset permissions were refused.
- ENOSPC during replacement on a private 1 MiB guest tmpfs preserved the original file.

The disk-full test covers atomic replacement, not exhaustion of the entire boot/state filesystem. Phase hooks are not power-loss certification. Failure injection exists only in the VM test build.

## Real theme and preview

Starfield passed import, validation, preview, installation, activation and reboot. Its normalized tree including notices is pinned to `b85ca5d4edb4af357c3ffa67bdf4dc74e43c436d94354ea2c714c63970cc4739`; package revision is `fa3fb6f82730aa0c96483ab9170def95ecd20103e4b4ef3509dd4f56cae551fc`. All 33 files, including five PF2 fonts and notices, were retained.

The final 1024×768 PNG was visually inspected: Starfield's background and generated Debian/Recovery menu rendered in GRUB under QEMU. This is a real GRUB preview of a generated menu. Separate reboot evidence establishes that the configured VM booted; neither proves physical-machine rendering.

An early capture was blank. The adapter now waits for both a serial readiness marker and the menu entry, allows the first paint to finish and rejects a blank frame. Preview has a six-minute overall limit, five-minute QEMU limit, and memory/file/descriptor bounds. Its namespace exposes no host boot disk, home directory or network. The external GPL renderer and theme assets are not bundled.

Grub of Tsushima stays browse-only: the user could not establish permissions for its Sony artwork and third-party fonts. The canonical repository is recorded, but no approved recipe, assets or installer were imported.

## Evidence and remaining limits

Local evidence is in `outputs/phase2-evidence/clean-vm`: command records, state snapshots, package versions, kernel-install log, failure log, preview PNG, console log and `PASS`. Ordinary logs are `work/phase2-windows.jsonl` and `work/phase2-linux-race.jsonl`. Ignored outputs, VM images and credentials are not uploaded as source. The harness deletes its runtime SSH key on exit.

The shipped Polkit policy requires administrator authorization. Tests used a scoped guest rule, so the graphical password prompt and desktop-agent behavior remain untested. The VM marker/SMBIOS check is a deployment guard, not protection against an existing root administrator.

Physical machines, WSL activation, Ubuntu, Arch, Fedora, BIOS, Secure Boot enabled, separate `/boot`, non-ext4 roots, GRUB Customizer and snapshot integrations are unsupported. GUI work, online catalog refresh, general theme upgrades and garbage collection remain deferred. This candidate is suitable for further disposable-VM testing within those limits.
