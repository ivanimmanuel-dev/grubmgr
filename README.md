# GRUB Manager

A command-line package manager for GRUB themes. Import community themes, validate their assets, preview them in QEMU and switch your boot menu. Each revision retains its source, file hashes and license notices. Rollback restores a previous theme while keeping current kernel entries.

**0.4.0-rc.1** supports conventional x86-64 UEFI GRUB installations on Debian 13, Ubuntu 24.04, Kali rolling and Arch rolling. Secure Boot must be disabled. Run `grubmgr doctor` to check your [configuration](docs/installation.md#supported-configurations).

## Download

| Your system | Package |
| --- | --- |
| Debian, Ubuntu or Kali · x86-64 | [Download .deb](https://github.com/ivanimmanuel-dev/grubmgr/releases/download/v0.4.0-rc.1/grubmgr_0.4.0rc1-1_amd64.deb) |
| Arch · x86-64 | [Download Arch package](https://github.com/ivanimmanuel-dev/grubmgr/releases/download/v0.4.0-rc.1/grubmgr-0.4.0rc1-1-x86_64.pkg.tar.xz) |

[Release details and checksums](https://github.com/ivanimmanuel-dev/grubmgr/releases/tag/v0.4.0-rc.1)

```sh
# Debian, Ubuntu or Kali
sudo apt install ./grubmgr_0.4.0rc1-1_amd64.deb

# Arch
sudo pacman -U ./grubmgr-0.4.0rc1-1-x86_64.pkg.tar.xz
```

Installing the package leaves your boot menu unchanged. Use `grubmgr` as your ordinary user; Polkit requests administrator authentication for system changes.

## Try a theme

```sh
grubmgr doctor
grubmgr search
grubmgr install cyberpunk-demo
grubmgr switch cyberpunk-demo
grubmgr history
```

Each change shows its destination and asks for confirmation. To restore the theme selected before a transaction, run `grubmgr rollback TRANSACTION_ID`.

Local themes work too:

```sh
grubmgr install ./my-theme
grubmgr status
grubmgr switch THEME_ID
```

Use the ID printed by `install` or listed by `status`. Directories and archives with multiple themes need a [recipe](docs/package-format.md) to select the entry and variants.

## Catalogs and previews

The built-in catalog contains Cyberpunk Demo and Debian's Starfield. You can [add a community catalog](docs/usage.md#community-catalogs) without rebuilding the application. Catalog recipes pin each archive by SHA-256 and retain the author's source links and license notices.

After [setting up the preview tools](docs/installation.md#preview-setup), run:

```sh
grubmgr preview cyberpunk-demo
```

## Help

- [Installation, requirements and removal](docs/installation.md)
- [Themes, variants, catalogs and rollback](docs/usage.md)
- [Recipes and catalog format](docs/package-format.md)

## License

GRUB Manager and its demo assets use the [MIT license](LICENSE). Dependencies and imported themes retain their [third-party notices](THIRD_PARTY_NOTICES.md).
