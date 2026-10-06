# Usage

Run `grubmgr help` for command syntax and `grubmgr doctor` to inspect your system. Use your ordinary account; Polkit handles administrator authentication.

## Install and select a theme

```sh
grubmgr search
grubmgr info cyberpunk-demo
grubmgr install cyberpunk-demo
grubmgr switch cyberpunk-demo --variant hd
grubmgr status
```

Installation copies assets into `/boot/grub/themes/grubmgr`. Switching changes `GRUB_THEME` and generates a checked menu with the installed kernels. The command shows the proposed change before asking for confirmation.

To import without installing:

```sh
grubmgr fetch cyberpunk-demo
grubmgr validate cyberpunk-demo
grubmgr list
```

`list` shows imports in your account. `status` shows system-installed revisions. If an ID has multiple revisions, select `namespace/name@REVISION`. The displayed 12-digit prefix works when unique; use more digits if it is ambiguous.

## Local themes

```sh
grubmgr install ./my-theme
grubmgr install ./theme.zip --recipe ./recipe.json
```

Without a recipe, the importer requires exactly one `theme.txt`. Use a [recipe](package-format.md) for multiple themes, variants, attribution or an archive containing installer scripts. Its `include` field selects the assets; declared license notices are retained automatically.

Local imports with unknown license metadata can be installed for personal use. Check the author's terms before redistributing their files.

### Starfield on Debian 13

The built-in Starfield recipe imports Debian's installed `grub-theme-starfield` package, version `2.12-9+deb13u2`:

```sh
sudo apt install grub-theme-starfield=2.12-9+deb13u2
grubmgr install starfield
grubmgr switch starfield
```

Its asset and font notices are retained with the imported files. See [third-party notices](../THIRD_PARTY_NOTICES.md#theme-assets).

## Community catalogs

Add a local catalog index or a pinned HTTPS index:

```sh
grubmgr catalog add community ./catalog.json
grubmgr catalog add collection HTTPS_INDEX_URL --sha256 INDEX_SHA256
grubmgr catalog list
grubmgr search
grubmgr info namespace/theme
grubmgr install namespace/theme
```

Replace the URL and digest with those supplied by the catalog publisher. A catalog is a [JSON recipe index](package-format.md#catalog-index). Each downloadable theme needs an exact HTTPS archive URL, SHA-256 digest and reviewed recipe. Adding a catalog changes your account's listings; it does not install themes.

Catalogs cannot replace built-in IDs or IDs from another catalog. Use full namespaced IDs when short names are ambiguous.

To replace an index:

```sh
grubmgr catalog remove community
grubmgr catalog add community ./new-catalog.json
```

Existing imported and installed revisions remain available. After an upstream update, run `grubmgr fetch namespace/theme`, then install and switch to the new full revision explicitly. Updates do not activate themselves.

## Preview

After [setting up the preview tools](installation.md#preview-setup):

```sh
grubmgr preview cyberpunk-demo
grubmgr preview cyberpunk-demo --variant hd
```

The command renders a GRUB menu in QEMU and returns the PNG path. On failure, it reports the log path. Preview uses its own menu and temporary boot image.

## Rollback and recovery

```sh
grubmgr history
grubmgr rollback TRANSACTION_ID
```

Rollback selects the theme that preceded that transaction and regenerates the menu with current kernels. Managed revisions remain on disk. Before switching away from a theme installed outside GRUB Manager, the helper copies and verifies its assets in a retained backup. Rollback uses that copy even if the original directory has been removed.

An existing theme needs a canonical absolute entry path and root-owned assets. System file links are copied into regular backup files. Writable assets, links to directories and unsupported theme data prevent the change.

After an interrupted operation:

```sh
grubmgr recover
```

Recovery restores the recorded configuration when its hashes match the expected state. A competing edit produces a conflict and retains the journal. Resolve that edit before retrying recovery.

## Remove a theme

```sh
grubmgr remove namespace/theme
```

Removal deactivates the theme if selected and clears its installed status. Its files remain available for rollback.

## Plans and scripting

To save a plan before applying it:

```sh
grubmgr plan switch cyberpunk-demo --variant hd
grubmgr apply PLAN_TOKEN
```

Use the complete token printed by `plan`, or its `plan_id` field in JSON output. If the package, configuration or boot inventory changes, create a new plan.

Direct commands accept `--yes` to confirm the change. Combine it with `--json` for machine-readable output; Polkit authentication still applies. `--log-json` writes logs to stderr.

```sh
grubmgr --json --log-json doctor
grubmgr --json --yes switch cyberpunk-demo
```
