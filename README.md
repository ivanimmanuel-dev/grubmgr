# GRUB Manager

`grubmgr` is a command-line manager for GRUB theme packages. It imports and validates themes, records immutable revisions, previews them in QEMU, and applies changes through an inspectable plan. Previous themes remain available for rollback.

**Status: experimental VM release, `0.3.0-rc.1`.** Activation supports specific Debian 13, Ubuntu 24.04, Kali and Arch test profiles. Physical-machine and WSL activation are disabled. See [Linux support](docs/linux-support.md) for exact versions and layouts.

## Build

Requires Go 1.27.1:

```sh
go build -o grubmgr ./cmd/grubmgr
./grubmgr doctor
./grubmgr search
```

On Windows, build with `-o grubmgr.exe` and run `./grubmgr.exe`. The standalone binary supports browsing, imports, validation and fixture testing. Linux activation also requires the packaged helper and Polkit policy.

Linux CI produces two package artifacts: `grubmgr-debian-amd64` for Debian, Ubuntu and Kali, and `grubmgr-arch-x86_64` for Arch. Each includes the package, checksum and file manifest. See [package instructions](docs/debian-package.md) and [Arch packaging](docs/linux-support.md#build-and-test).

## Try it with a fixture

Copy the synthetic Debian root into a new working directory:

```sh
mkdir -p work
cp -R fixtures/debian work/demo-debian
./grubmgr --root work/demo-debian fetch cyberpunk-demo
./grubmgr --root work/demo-debian validate cyberpunk-demo
./grubmgr --root work/demo-debian plan install grubmgr/cyberpunk-demo
```

In PowerShell, use `Copy-Item -Recurse fixtures/debian work/demo-debian` for the copy. Keep the same `--root` for each command.

Review the plan, then replace `PLAN_TOKEN` below with its complete `plan_id`:

```sh
./grubmgr --root work/demo-debian apply PLAN_TOKEN
./grubmgr --root work/demo-debian plan switch grubmgr/cyberpunk-demo
```

Apply the new switch token to select the theme. `install` copies assets; `switch` activates an installed revision. Plans expire when the relevant state changes.

To restore the theme selection that preceded a transaction, find its ID in `history`, run `plan rollback TRANSACTION_ID`, then apply the returned token. Rollback preserves the current boot entries.

## Commands

| Command | Purpose |
| --- | --- |
| `doctor` | Inspect the system and report activation support |
| `search [QUERY]`, `info ID` | Browse the catalog and package details |
| `fetch SOURCE [--recipe FILE]` | Import a catalog entry, directory, archive or reviewed HTTPS artifact |
| `validate PATH_OR_ID [--recipe FILE]` | Check theme syntax, references, images, fonts and metadata |
| `list` | Show the user's imported packages |
| `status`, `history` | Show managed package state and transactions |
| `plan install\|switch\|remove ID` | Inspect a proposed change |
| `plan rollback TRANSACTION_ID` | Select the theme used before a committed transaction |
| `apply TOKEN`, `recover` | Apply a plan or recover an interrupted transaction |
| `preview ID [--variant ID]` | Render a generated menu using GRUB and QEMU |

Use `--variant ID` with install or switch plans, `--json` for structured output, and `--log-json` for logs on stderr. If several revisions are cached, select `namespace/name@FULL_REVISION`. On supported VMs, `doctor`, `plan`, `status`, `history`, `apply` and `recover` use the Polkit helper; `list` reads the user store.

## Themes

The catalog ships with Cyberpunk Demo, an original test theme approved for all four VM profiles, and a Debian-only Starfield import that preserves the upstream asset and font notices. Minegrub and Grub of Tsushima are discovery links pending recipe and asset-license review.

Theme packages contain data. The Linux helper accepts the recipes and content pins compiled into its build. General community-theme updates, catalog refresh, garbage collection and a graphical interface are planned work.

## Development

```sh
go test ./...
go vet ./...
go mod verify
go run ./tools/checklicenses
```

CI runs Windows and Linux tests, Linux race detection, and package checks. Boot tests use a separate [disposable VM harness](docs/vm-testing.md). The [verification report](docs/linux-verification.md) records the tested archives, reboots and recovery cases.

## Documentation

- [Linux support and preview setup](docs/linux-support.md)
- [Architecture](docs/architecture.md)
- [Package and recipe format](docs/package-format.md)
- [Security model](docs/security-model.md)
- [Testing](docs/testing.md)
- [Ecosystem audit](docs/ecosystem-audit.md) and [reuse decisions](docs/reuse-plan.md)
- [Release notes](docs/releases/v0.3.0-rc.1.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)

Original code and demo assets are [MIT licensed](LICENSE). Dependencies and imported themes retain their own licenses.
