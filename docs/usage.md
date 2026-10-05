# Usage

Run `grubmgr help` for command syntax and `grubmgr doctor` to inspect your system. Use your ordinary account; Polkit handles administrator authentication when needed.

## Import a theme

```sh
grubmgr search
grubmgr info cyberpunk-demo
grubmgr fetch cyberpunk-demo
grubmgr validate cyberpunk-demo
grubmgr list
```

For local themes:

```sh
grubmgr fetch ./my-theme
grubmgr fetch ./theme.zip --recipe ./recipe.json
```

Directories and archives without a recipe must contain exactly one `theme.txt`. A [recipe](package-format.md) supplies the theme identity, entry point, variants and license information. HTTPS imports also require the exact URL and SHA-256 in that recipe.

`list` shows your imported revisions. `status` shows packages installed through the system helper. If an ID has multiple revisions, use `namespace/name@FULL_REVISION`. After a catalog recipe changes, fetch the package again and select the new revision explicitly.

### Starfield on Debian 13

The Starfield recipe imports the installed `grub-theme-starfield` package, version `2.12-9+deb13u2`:

```sh
sudo apt install grub-theme-starfield=2.12-9+deb13u2
grubmgr fetch starfield
grubmgr validate starfield
```

Its asset and font notices are retained with the imported files. See [third-party notices](../THIRD_PARTY_NOTICES.md#theme-assets).

## Preview

After [setting up the preview tools](installation.md#preview-setup):

```sh
grubmgr preview cyberpunk-demo
grubmgr preview cyberpunk-demo --variant hd
```

The command renders a GRUB menu in QEMU and returns the PNG path. On failure, it reports the log path. Preview uses its own menu and temporary boot image.

## Install and switch

```sh
grubmgr plan install cyberpunk-demo
grubmgr apply PLAN_TOKEN
grubmgr plan switch cyberpunk-demo --variant hd
grubmgr apply SWITCH_TOKEN
```

Replace each placeholder with the complete `plan_id` returned by its preceding command. A plan shows the revision, destination, selected theme and configuration generator. If the package or system changes before application, create a new plan.

Installation copies assets. Switching updates `GRUB_THEME` and regenerates the menu with the current kernels. The helper accepts the catalog recipes compiled into its build.

## Rollback and recovery

```sh
grubmgr history
grubmgr plan rollback TRANSACTION_ID
grubmgr apply PLAN_TOKEN
```

Rollback restores the managed theme selected before that transaction and keeps current kernel entries. It requires the earlier revision's retained files; an unmanaged theme is not a rollback target.

After an interrupted operation:

```sh
grubmgr recover
```

Recovery restores the recorded configuration when its hashes match the expected state. If another process edited the files, the command reports a conflict and retains the journal for inspection. Resolve the competing edit before retrying.

## Remove a theme

```sh
grubmgr plan remove cyberpunk-demo
grubmgr apply PLAN_TOKEN
```

Removal deactivates the theme if selected and clears its installed status. Its files remain available for rollback.

## Scripting

Use `--json` for machine-readable results and `--log-json` for logs on stderr:

```sh
grubmgr --json --log-json doctor
```
