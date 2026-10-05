# Package and recipe format

Schema 1 describes theme data, provenance and compatibility. JSON decoding rejects unknown fields. Use [synthetic-recipe.json](../examples/synthetic-recipe.json) as a local fixture example; the [embedded catalog](../internal/catalog/catalog.json) contains the Linux helper recipes.

## Recipe fields

| Field | Meaning |
| --- | --- |
| `schema_version` | Exactly `1` |
| `id` | Lowercase `namespace/name`, using letters, digits, dots and hyphens |
| `name`, `author`, `project_url`, `version` | Display identity; synthetic or local packages may have an empty project URL |
| `source.provider`, `source.url`, `source.upstream_revision` | Provider, canonical source and upstream revision |
| `artifact_sha256` | Reviewed digest required for remote artifacts; optional expected digest for local imports |
| `theme_root`, `entry` | Relative root and entry file; root may be `.` |
| `variants` | Additional `{id, root, entry, resolutions}` records; `default` selects the primary root and entry |
| `license` | `{spdx, notices, verified, evidence}`; notice paths are relative to the package |
| `compatibility` | `{backends, architectures, firmware, requires, notes}` |
| `preview` | `{kind, source, attribution}` describing preview media |
| `recipe_revision` | Packaging revision, separate from the upstream theme version |
| `reviewed` | Curator assertion that the recipe has been reviewed |

`compatibility.backends` names an explicit target:

| Backend | Target |
| --- | --- |
| `fixture-debian` | Synthetic Debian root |
| `debian13-uefi-vm` | Debian 13 VM |
| `ubuntu2404-uefi-vm` | Ubuntu 24.04 VM |
| `kali-rolling-uefi-vm` | Kali VM |
| `arch-uefi-vm` | Arch VM |

The [supported configurations](installation.md#supported-configurations) define the exact GRUB versions and layouts. The fixture planner also checks any declared architecture and firmware values. The Linux helper checks its compiled recipe and profile independently; adding a backend name or setting `reviewed` in a local recipe does not authorize activation. Required capabilities such as `gfxterm` are descriptive metadata checked during integration testing.

## Package identity

Import adds `revision`, `tree_sha256` and a sorted `files` inventory of `{path, size, sha256}` records, including license notices.

- `artifact_sha256` hashes the original archive bytes. For directory, built-in and installed-package imports, it equals the normalized tree digest.
- `tree_sha256` hashes the compact Go JSON encoding of the sorted inventory.
- `revision` hashes the compact Go JSON encoding of the complete manifest with its `revision` field empty.

Recipe metadata, notices, variants and file content therefore affect revision identity. These encodings belong to schema 1. Empty directories, timestamps and executable bits are excluded; imported files become ordinary data copies.

Paths use portable ASCII names. Absolute paths, traversal, Windows device names, trailing spaces or dots, and case aliases are rejected on every platform.

## Imports

```sh
grubmgr fetch ./my-theme
grubmgr fetch ./theme.zip --recipe ./recipe.json
grubmgr fetch HTTPS_ARCHIVE_URL --recipe ./recipe.json
grubmgr validate ./my-theme --recipe ./recipe.json
```

Replace `HTTPS_ARCHIVE_URL` with the exact URL recorded in the recipe. Remote imports require HTTPS and a reviewed SHA-256 digest. Redirects are bounded and must retain HTTPS. Supported archive suffixes are `.zip`, `.tar`, `.tar.gz` and `.tgz`; download other endpoint names to a local file with the appropriate suffix before importing. Git refs are not fetched.

Without a recipe, the importer requires exactly one `theme.txt`. Zero or multiple matches return `AMBIGUOUS_THEME_ROOT`. An inferred local recipe has unknown provenance and is unavailable for activation planning. An explicit recipe selects its entry and variants; other `theme.txt` files are inventoried but not selected.

Explicit local input paths may be outside a fixture's `--root`. System inspection and destination writes remain inside the fixture.

| Resource | Limit |
| --- | --- |
| Archive or download | 128 MiB |
| Expanded file contents | 128 MiB total; 16 MiB per file |
| Members | 4,096 |
| Path | 240 characters; at most 32 separators |
| Theme entry | 1 MiB |
| Decoded image | 16 megapixels |

ZIP and TAR use the same extraction policy. It rejects duplicate or conflicting paths, symlinks, hardlinks, devices, FIFOs and unsupported extended TAR metadata. Limits apply before and during expansion.

## Validation

The parser accepts line-based `key: value`, `key = value`, `+ component {` and closing `}` forms, including quoted values and property comments. Unsupported inline components or multiline properties return `THEME_SYNTAX`.

Asset references resolve relative to the entry directory. Nested paths and pixmap patterns are supported; each wildcard must match at least one file. Validation checks pattern matches but leaves nine-slice completeness to rendering. Fonts resolve by embedded PF2 name, including GRUB's customary `Unifont Regular 16`.

PNG and JPEG files are decoded within resource limits. The PF2 inspector checks section framing, signature, `NAME`, `CHIX` record size and `DATA` presence. Glyph rendering and other image formats require preview or additional validator support. Executable signatures and script extensions are rejected.

`BOOT_UNVERIFIED` reports that no boot test was performed. Use preview to check rendering.

Receipts retain the selected variant, validation report, installed and active flags, and a pin flag reserved for future update policy. Multiple immutable revisions can coexist. Select a full revision when an ID is ambiguous.
