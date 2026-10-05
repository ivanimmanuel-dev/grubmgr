GRUB Manager (`grubmgr`) is a package manager for community GRUB themes.

Version **0.1.0** provides a Go CLI, immutable package imports, provenance and hashes, validation, a shipped catalogue, SQLite receipts, read-only plans, and a working **synthetic-root transaction backend** with recovery tests.

**Real bootloader modification is disabled.** This build never runs a GRUB utility, QEMU, a theme installer, or a privileged helper. Linux is the product target; Windows is supported for developing and demonstrating the fixture workflow.

## Build

Use Go **1.27.1** (the stable toolchain used to verify this release), or a compatible newer Go 1.27 toolchain:

```sh
go mod download
go build -o grubmgr ./cmd/grubmgr
./grubmgr version
./grubmgr doctor
./grubmgr search cyberpunk
./grubmgr info grubmgr/cyberpunk-demo
```

Windows: `go build -o grubmgr.exe ./cmd/grubmgr`, then use `./grubmgr.exe`.
No host-wide installation is required. Run ordinary package imports as your normal user.

## Working end-to-end demo

Copy a fixture so the checked-in examples remain pristine. The following commands work in a POSIX shell; Python 3 is used only to select the plan token from JSON:

```sh
mkdir -p work
cp -R fixtures/debian work/demo-debian
./grubmgr --root work/demo-debian doctor
./grubmgr search cyberpunk
./grubmgr info grubmgr/cyberpunk-demo
./grubmgr --root work/demo-debian fetch grubmgr/cyberpunk-demo
./grubmgr --root work/demo-debian validate grubmgr/cyberpunk-demo
./grubmgr --root work/demo-debian list
./grubmgr --root work/demo-debian plan install grubmgr/cyberpunk-demo

PLAN=$(./grubmgr --root work/demo-debian --json plan install grubmgr/cyberpunk-demo | python3 -c 'import json,sys; print(json.load(sys.stdin)["plan_id"])')
./grubmgr --root work/demo-debian apply "$PLAN"

PLAN=$(./grubmgr --root work/demo-debian --json plan switch grubmgr/cyberpunk-demo | python3 -c 'import json,sys; print(json.load(sys.stdin)["plan_id"])')
./grubmgr --root work/demo-debian apply "$PLAN"
./grubmgr --root work/demo-debian status
./grubmgr --root work/demo-debian history
```

PowerShell equivalent:

```powershell
New-Item -ItemType Directory -Force work | Out-Null
Copy-Item -Recurse fixtures/debian work/demo-debian
./grubmgr.exe --root work/demo-debian doctor
./grubmgr.exe search cyberpunk
./grubmgr.exe info grubmgr/cyberpunk-demo
./grubmgr.exe --root work/demo-debian fetch grubmgr/cyberpunk-demo
./grubmgr.exe --root work/demo-debian validate grubmgr/cyberpunk-demo
./grubmgr.exe --root work/demo-debian list
$plan = ./grubmgr.exe --root work/demo-debian --json plan install grubmgr/cyberpunk-demo | ConvertFrom-Json
$plan
./grubmgr.exe --root work/demo-debian apply $plan.plan_id
$plan = ./grubmgr.exe --root work/demo-debian --json plan switch grubmgr/cyberpunk-demo | ConvertFrom-Json
./grubmgr.exe --root work/demo-debian apply $plan.plan_id
./grubmgr.exe --root work/demo-debian history
```

Use a fresh copy for another demo. **Use the same `--root` on every stateful command.** Each root owns its own `.grubmgr` store. Planning on one root and applying on another is refused.

`install` stages a revision without changing the selected theme. `switch` activates an installed revision. The synthetic backend changes only files beneath the supplied, marked fixture root. Its generated configuration is a fixture demonstration, not output from real GRUB.

The bundled `grubmgr/cyberpunk-demo` is original synthetic test data, with a `default` and an `hd` variant. It is not an upstream community theme. `info minegrub` displays a factual upstream link marked **UNREVIEWED — BROWSE ONLY**; `fetch minegrub` refuses until a reviewed recipe and asset rights are supplied. No Gorgeous-GRUB media or catalogue descriptions are bundled.

## Commands

| Command | Implemented behavior |
| --- | --- |
| `doctor` | Read-only rooted distro, architecture, firmware, layout, theme, utility and preview diagnostics |
| `search [query]`, `info ID` | Shipped catalogue and fetched-package provenance |
| `fetch ID-or-path` | Import builtin data, local directories, ZIP, TAR and TAR.GZ |
| `fetch HTTPS_URL --recipe recipe.json` | Fetch an explicitly reviewed, hash-pinned artifact; no mutable Git checkout or scripts |
| `validate directory-or-ID [--recipe file]` | Structured syntax-subset, path, asset, image, PF2 and metadata findings |
| `list`, `status`, `history` | SQLite package receipts, active selection and transaction outcomes |
| `plan install ID` | Read-only asset staging plan |
| `plan switch ID --variant hd` | Read-only activation plan for an installed revision |
| `plan remove ID` | Plan deactivation if active; retain assets for rollback |
| `plan rollback TRANSACTION_ID` | Select that transaction's prior managed theme using the current configuration |
| `apply p1.…` | Recompute and verify a plan, then apply to a Debian fixture only |
| `recover` | Restore an interrupted fixture transaction if recorded hashes still match |
| `preview ID` | Detect external dependencies and explicitly refuse unavailable/unreviewed execution |

If multiple revisions exist, select `namespace/name@FULL_REVISION`. An unreviewed local import can be fetched and validated, but activation planning requires an explicit reviewed recipe and license evidence. The `reviewed` flag is an operator assertion, not a digital signature or a legal determination.

Plans are self-contained `p1.*` tokens. They are long because **planning writes no plan cache**. Tokens carry a request and precondition digest; apply derives all destinations and changes itself. A token is not authorization or a signature. State changes invalidate old plans; obtain a fresh plan after every apply/fetch that changes receipts.

Without `--root`, stores follow `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_CACHE_HOME`, and `XDG_STATE_HOME`, with their standard home-directory fallbacks. Read-only commands do not create these directories. With `--root`, stores and OS inspection stay under that root and host utility paths are never searched.

`--json` writes structured results to stdout and structured errors to stderr. Optional `--log-json` adds structured start/finish logs to stderr (NDJSON when combined with `--json`); stdout stays machine-readable. Findings have `severity`, `code`, `file`, `line`, and `message`. Exit codes: **0** success, **2** usage, **3** invalid package/input, **4** unsupported/refused, **5** conflict/stale plan/lock, **6** I/O or successfully rolled-back operation failure, **7** recovery required. `validate` and `fetch` return 3 when their result contains validation errors; invalid imported data is retained for inspection and cannot be planned.

## Tests and limits

```sh
go test ./...
go vet ./...
go mod verify
go run ./tools/checklicenses
```

Tests exercise all transaction failure boundaries, interrupted recovery, current-kernel preservation during later rollback, stale plans, immutable receipts, archive limits, path/link refusals, nested assets, PF2 names, and all five distro fixtures. See [testing](docs/testing.md) for platform-specific instructions and the real-VM acceptance gate.

Not implemented: real activation/Polkit, authoritative GRUB syntax/rendering validation, QEMU execution, automatic updates, catalogue refresh, pin/unpin commands, garbage collection, package exports, GUI, native distro packaging, or non-Debian activation. Pins are represented in receipts for future policy. Arch/Fedora detection is read-only; ambiguous and non-GRUB systems are refused. Asset retention intentionally uses additional disk space.

See [architecture](docs/architecture.md), [package format](docs/package-format.md), [security model](docs/security-model.md), and [third-party notices](THIRD_PARTY_NOTICES.md). The completed [ecosystem audit](docs/ecosystem-audit.md) and [reuse plan](docs/reuse-plan.md) remain the design foundation.
