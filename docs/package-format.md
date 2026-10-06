# Package and recipe format

Schema 1 describes theme data, provenance and compatibility. JSON decoding rejects unknown fields. Use [local-recipe.json](../examples/local-recipe.json) as a starting point for your own theme; `grubmgr info ID` shows a themeâ€™s catalog recipe.

## Recipe fields

| Field | Meaning |
| --- | --- |
| `schema_version` | Exactly `1` |
| `id` | Lowercase `namespace/name`, using letters, digits, dots and hyphens |
| `name`, `author`, `project_url`, `version` | Display identity; local packages may have an empty project URL |
| `source.provider`, `source.url`, `source.upstream_revision` | Provider, canonical source and upstream revision |
| `artifact_sha256` | Exact digest required for remote artifacts; optional expected digest for local imports |
| `theme_root`, `entry` | Relative root and entry file; root may be `.` |
| `include` | Optional relative file or directory paths to retain from the archive; declared license notices are always retained |
| `variants` | Additional `{id, root, entry, resolutions}` records; `default` selects the primary root and entry |
| `license` | `{spdx, notices, verified, evidence}`; notice paths are relative to the package |
| `compatibility` | `{backends, architectures, firmware, requires, notes}` |
| `preview` | `{kind, source, attribution}` describing preview media |
| `recipe_revision` | Packaging revision, separate from the upstream theme version |
| `reviewed` | Curator assertion that the recipe has been reviewed |

`compatibility.backends` may be empty, `linux-grub` for every supported Linux backend, or a specific backend:

| Backend | Target |
| --- | --- |
| `debian-grub` | Debian 13 |
| `ubuntu-grub` | Ubuntu 24.04 |
| `kali-grub` | Kali rolling |
| `arch-grub` | Arch rolling |

Architecture and firmware constraints use `amd64` and `uefi`. The helper independently checks the [system configuration](installation.md#supported-configurations), package hashes and theme data. `reviewed` and `license.verified` record curator assertions; they do not bypass validation. Required capabilities such as `gfxterm` describe the theme's rendering requirements.

### Selecting files from a collection

An archive may contain several themes and an upstream installer. Select the asset directory in the recipe:

```json
"theme_root": "collection/theme",
"entry": "theme.txt",
"include": ["collection/theme"],
"license": {
  "spdx": "MIT",
  "notices": ["collection/LICENSE"],
  "verified": true,
  "evidence": "Upstream LICENSE at the pinned revision"
}
```

Replace the paths and license with the upstream values. The importer retains the selected directory and the declared notice. Every include path must exist. Without `include`, the entire archive is retained. Included scripts or executable files still fail validation.

## Catalog index

A catalog is a JSON array of records containing `recipe`, optional `status` and optional `tree_sha256`:

```json
[
  {
    "recipe": {
      "schema_version": 1,
      "id": "author/theme",
      "name": "Theme name",
      "author": "Theme author",
      "project_url": "https://example.org/theme",
      "version": "1.0.0",
      "source": {
        "provider": "https",
        "url": "https://example.org/theme-1.0.0.zip",
        "upstream_revision": "1.0.0"
      },
      "artifact_sha256": "REPLACE_WITH_64_LOWERCASE_HEX_DIGITS",
      "theme_root": "theme-1.0.0",
      "entry": "theme.txt",
      "variants": [],
      "license": {
        "spdx": "MIT",
        "notices": ["theme-1.0.0/LICENSE"],
        "verified": true,
        "evidence": "Upstream LICENSE at version 1.0.0"
      },
      "compatibility": {"backends": ["linux-grub"], "requires": ["gfxterm"]},
      "preview": {"kind": "none"},
      "recipe_revision": "1",
      "reviewed": true
    }
  }
]
```

Replace the example identity, URLs, license and digest before using the index. IDs must be unique. Online indexes require their own SHA-256 when registered. Limits are 4 MiB and 2,048 records per index, with at most 32 registered catalogs. Each theme archive is downloaded and checked independently. See [catalog commands](usage.md#community-catalogs).

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

Without a recipe, the importer requires exactly one `theme.txt`. Zero or multiple matches return `AMBIGUOUS_THEME_ROOT`. An inferred local recipe records unknown author and license metadata; validated imports can be installed. An explicit recipe selects its entry and variants; other `theme.txt` files are inventoried but not selected.

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

Receipts retain the selected variant, validation report, installed and active flags. Multiple immutable revisions can coexist. Select a full revision when an ID is ambiguous.
