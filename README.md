# GRUB Manager

`grubmgr` installs, switches, and rolls back GRUB themes.

The CLI includes a fixture backend and experimental Debian, Ubuntu, Kali and Arch VM profiles. Physical-machine activation is disabled.

## Build

Requires Go 1.27.1.

```sh
go build -o grubmgr ./cmd/grubmgr
./grubmgr doctor
./grubmgr search
```

On Windows, use `-o grubmgr.exe` and `./grubmgr.exe`.

A [Debian package](docs/debian-package.md) is available for disposable-VM testing. Linux CI publishes it as the `grubmgr-debian-amd64` workflow artifact.

The [Linux support guide](docs/linux-support.md) covers Ubuntu/Kali `.deb` testing and the Arch pacman package. Linux CI also publishes `grubmgr-arch-x86_64`.

## Try the fixture workflow

```sh
mkdir -p work
cp -R fixtures/debian work/demo-debian
./grubmgr --root work/demo-debian fetch cyberpunk-demo
./grubmgr --root work/demo-debian validate cyberpunk-demo
./grubmgr --root work/demo-debian plan install grubmgr/cyberpunk-demo
./grubmgr --root work/demo-debian apply <plan-token>
./grubmgr --root work/demo-debian plan switch grubmgr/cyberpunk-demo
./grubmgr --root work/demo-debian apply <plan-token>
./grubmgr --root work/demo-debian history
./grubmgr --root work/demo-debian plan rollback <transaction-id>
./grubmgr --root work/demo-debian apply <plan-token>
```

Use the token printed by each plan. Use the same `--root` throughout. `install` copies assets; `switch` selects an installed theme. A new plan is needed after state changes. In PowerShell, copy the fixture with `Copy-Item -Recurse fixtures/debian work/demo-debian`.

## Commands

| Command | Purpose |
| --- | --- |
| `doctor` | Inspect the system and report activation support |
| `search [query]`, `info ID` | Browse themes and package details |
| `fetch ID-or-path` | Import a catalog entry, directory, ZIP, TAR, or TAR.GZ |
| `fetch HTTPS_URL --recipe FILE` | Download an artifact pinned by a reviewed recipe |
| `validate PATH-or-ID` | Check theme syntax, paths, images, fonts, and metadata |
| `list`, `status`, `history` | Show packages and transactions |
| `plan install\|switch\|remove ID` | Inspect a change before applying it |
| `plan rollback TRANSACTION` | Restore a previous theme selection |
| `apply TOKEN`, `recover` | Apply a plan or recover an interrupted transaction |
| `preview ID` | Render a generated menu with optional GRUB/QEMU tools |

Use `--variant ID` to select a variant, `--json` for structured output, and `--log-json` for logs on stderr. If several revisions are cached, select `namespace/name@FULL_REVISION`.

## Support

| Environment | Activation |
| --- | --- |
| Marked Debian fixture | Working |
| Debian 13.7 amd64 UEFI test VM, GRUB 2.12-9+deb13u2 | Experimental; exact layout only |
| Ubuntu 24.04 amd64 UEFI test VM | Experimental; separate ext4 `/boot` profile |
| Kali amd64 UEFI test VM | Experimental; pinned GRUB and ext4 layout |
| Arch x86-64 UEFI test VM | Experimental; flat Btrfs root and `/efi` profile |
| Physical machines, WSL | Refused |
| Fedora | Detection only |
| Other systems | Read-only or refused |

Cyberpunk Demo is an original test theme. Starfield imports a pinned Debian data package and preserves its notices. Other themes need reviewed recipes, content pins and asset licenses. Grub of Tsushima is deferred until its artwork/font permissions are known. No community installer scripts run.

## Tests

```sh
go test ./...
go vet ./...
go mod verify
go run ./tools/checklicenses
```

See [testing](docs/testing.md), [Linux VM verification](docs/linux-verification.md), the earlier [Debian verification](docs/phase2-verification.md), and the [VM harness](docs/vm-testing.md). Normal CI uses temporary fixtures. Real boot tests run separately in disposable VMs.

## Documentation

- [Architecture](docs/architecture.md)
- [Debian support and recovery](docs/debian-backend.md)
- [Linux profiles, packages and VM tests](docs/linux-support.md)
- [Package format](docs/package-format.md)
- [Security model](docs/security-model.md)
- [Ecosystem audit](docs/ecosystem-audit.md)
- [Reuse plan](docs/reuse-plan.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)

MIT license. Dependency and theme licenses remain separate.
