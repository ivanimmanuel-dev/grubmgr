# GRUB Manager

Import, validate, preview and switch GRUB themes from the command line. Keep previous theme revisions and restore a selection with rollback.

**0.3.0-rc.1 · Linux VM preview.** Activation supports the [documented Debian, Ubuntu, Kali and Arch QEMU configurations](docs/installation.md#supported-configurations). Physical-machine and WSL activation are disabled.

## Download

| Your system | Package |
| --- | --- |
| Debian, Ubuntu or Kali · x86-64 | [Download .deb](https://github.com/ivanimmanuel-dev/grubmgr/releases/download/v0.3.0-rc.1/grubmgr_0.3.0~rc.1-1_amd64.deb) |
| Arch · x86-64 | [Download Arch package](https://github.com/ivanimmanuel-dev/grubmgr/releases/download/v0.3.0-rc.1/grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz) |

[Release details and checksums](https://github.com/ivanimmanuel-dev/grubmgr/releases/tag/v0.3.0-rc.1)

## Install

Debian, Ubuntu or Kali:

```sh
sudo apt install ./grubmgr_0.3.0~rc.1-1_amd64.deb
```

Arch:

```sh
sudo pacman -U ./grubmgr-0.3.0rc1-1-x86_64.pkg.tar.xz
```

Run `grubmgr doctor` to check your configuration. See [installation and preview setup](docs/installation.md) for requirements.

## Select a theme

```sh
grubmgr search
grubmgr fetch cyberpunk-demo
grubmgr validate cyberpunk-demo
grubmgr plan install cyberpunk-demo
```

Read the plan and apply its complete token:

```sh
grubmgr apply PLAN_TOKEN
grubmgr plan switch cyberpunk-demo
```

Apply the switch plan's token to select the theme for the boot menu. Use `grubmgr history` and `grubmgr plan rollback TRANSACTION_ID` to restore a previous selection.

The catalog includes Cyberpunk Demo and Debian's Starfield theme. Local directories and archives can also be imported for validation and preview.

## Help

- [Installation, preview tools and removal](docs/installation.md)
- [Using themes, variants and rollback](docs/usage.md)
- [Importing a theme with a recipe](docs/package-format.md)

## License

GRUB Manager and its demo assets use the [MIT license](LICENSE). Dependencies and imported themes retain their [third-party notices](THIRD_PARTY_NOTICES.md).
