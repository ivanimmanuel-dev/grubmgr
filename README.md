# GRUB Manager

`grubmgr` imports, validates, previews and switches GRUB themes. It keeps theme revisions and transaction history so you can restore an earlier selection.

Version `0.3.0-rc.1` supports activation in specific Debian, Ubuntu, Kali and Arch QEMU VMs. Physical-machine and WSL activation are disabled. See [supported configurations](docs/installation.md#supported-configurations).

## Install

Download a package from a successful [GitHub Actions run](https://github.com/ivanimmanuel-dev/grubmgr/actions/workflows/ci.yml). Use `grubmgr-debian-amd64` for Debian, Ubuntu or Kali, and `grubmgr-arch-x86_64` for Arch.

```sh
# Debian, Ubuntu or Kali
sudo apt install ./grubmgr_0.3.0~rc.1-1_amd64.deb

# Arch
sudo pacman -U ./grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz
```

Run `grubmgr doctor` to check your configuration. [Installation](docs/installation.md) covers building from source and setting up preview tools.

## Use

```sh
grubmgr search
grubmgr fetch cyberpunk-demo
grubmgr validate cyberpunk-demo
grubmgr plan install cyberpunk-demo
```

Read the plan, then apply its complete `plan_id`:

```sh
grubmgr apply PLAN_TOKEN
grubmgr plan switch cyberpunk-demo
```

Apply the switch plan's token to activate the theme. `install` copies assets; `switch` selects them for the boot menu.

Use `grubmgr history` to find a transaction and `grubmgr plan rollback TRANSACTION_ID` to restore the theme selected before it. [Usage](docs/usage.md) covers variants, preview, removal and recovery.

## Themes

The catalog includes Cyberpunk Demo for all four VM profiles and Debian's Starfield theme for Debian 13. You can also import local directories or archives for validation and preview. Linux activation accepts the recipes bundled with the installed helper.

## Development

Requires Go 1.27.1:

```sh
go build -o grubmgr ./cmd/grubmgr
go test ./...
go vet ./...
```

On Windows, build with `-o grubmgr.exe`. The [testing guide](docs/testing.md) includes a fixture walkthrough and boot tests for contributors.

## Documentation

- [Installation and supported configurations](docs/installation.md)
- [Usage](docs/usage.md)
- [Theme package format](docs/package-format.md)
- [Architecture](docs/architecture.md)
- [Testing and package builds](docs/testing.md)

## License

GRUB Manager and its demo assets use the [MIT license](LICENSE). See [third-party notices](THIRD_PARTY_NOTICES.md) for dependencies and imported themes.
