# Package and recipe format, schema 1

Recipes are JSON data decoded with unknown fields rejected. No command, hook, plugin or executable field exists. Start with the reviewed synthetic recipe in `internal/catalog/catalog.json` when authoring a local fixture recipe; `examples/synthetic-recipe.json` is a standalone copy. Never set `reviewed` merely to bypass missing asset rights.

## Recipe fields

| Field | Meaning |
| --- | --- |
| `schema_version` | Exactly 1 |
| `id` | Stable lowercase `namespace/name`; letters, digits, dots and hyphens |
| `name`, `author`, `project_url`, `version` | Human identity; an unknown project URL may be empty for synthetic/local data |
| `source.provider`, `source.url`, `source.upstream_revision` | Origin, canonical fetch location, immutable upstream reference if supplied |
| `artifact_sha256` | Required reviewed digest for remote artifacts; optional expected digest for local imports |
| `theme_root`, `entry` | Explicit relative root and entry, normally `theme.txt`; root may be `.` |
| `variants` | Additional `{id, root, entry, resolutions}` entries; `default` is reserved for the primary root/entry |
| `license` | `{spdx, notices, verified, evidence}`; notice paths are relative to the package |
| `compatibility` | `{backends, architectures, firmware, requires, notes}` |
| `preview` | `{kind, source, attribution}` descriptive metadata; never QEMU arguments |
| `recipe_revision` | Packaging recipe identity, distinct from human theme version |
| `reviewed` | Operator/curator assertion; not a cryptographic attestation |

The only installable compatibility value today is `fixture-debian`. Optional architecture/firmware lists must match fixture evidence. Required capabilities such as `gfxterm` are descriptive; static validation cannot prove them and emits a boot-unverified warning. No real compatibility combination is certified.

Import adds `revision`, `tree_sha256`, and sorted `files` entries `{path,size,sha256}`. `artifact_sha256` is the digest of original archive/download bytes. For local directories and builtin data there is no archive, so it equals the canonical tree digest. `tree_sha256` hashes the compact JSON array of sorted file records. `revision` hashes the compact Go JSON encoding of the complete manifest with its `revision` field empty. Thus source, recipe, license, variants and file content affect identity. These schema-1 encodings are versioned implementation rules, not a general canonical-JSON standard.

The file inventory includes license notices. Empty directories and timestamps are not identity-bearing; executable bits are not preserved. Source files are normalized to ordinary data copies. The initial portable filename policy accepts ASCII paths, rejects Windows device names and trailing spaces/dots, and rejects case aliases on every target.

## Imports

```sh
grubmgr fetch ./my-theme
grubmgr fetch ./theme.zip --recipe ./recipe.json
grubmgr fetch https://example.org/releases/theme-1.zip --recipe ./recipe.json
grubmgr validate ./my-theme --recipe ./recipe.json
```

The example URL is illustrative: supply an actual reviewed URL and its digest. HTTPS is required; plaintext HTTP and redirects to plaintext are refused. The recipe URL must exactly match the requested URL. Redirects are bounded. Only ZIP/TAR/TAR.GZ/TGZ artifact names are accepted; endpoints with non-archive names currently require downloading the file explicitly first. Mutable Git refs are not fetched. Local paths outside `--root` may be supplied explicitly as input data; OS inspection and destination state still remain rooted.

Without a recipe, exactly one `theme.txt` must exist. Zero or multiple candidates yield `AMBIGUOUS_THEME_ROOT`; the importer never chooses the first. Inferred local packages have unknown license/provenance and remain unavailable for activation planning. With a recipe, all declared variants are checked and unknown sibling `theme.txt` files are simply ordinary inventory files, not selected entries.

Limits: 128 MiB artifact/download, 128 MiB total expanded files, 16 MiB per file, 4,096 members, at most 32 path separators, 240-character paths, 1 MiB theme entry, and 16 megapixels per decoded image. ZIP and TAR share the same member policy: no absolute/traversal/noncanonical paths, duplicate or case-ambiguous members, symlinks, hardlinks, devices/FIFOs or unsupported extended TAR metadata. Limits are checked before and during expansion.

## Validation coverage

The theme grammar subset accepts line-based `key: value`, `key = value`, `+ component {` and closing `}`. It rejects malformed/unsupported multiline or inline-component forms with `THEME_SYNTAX` instead of claiming complete GRUB parsing. Quoted values and property comments are supported. Asset paths are resolved relative to the entry directory, including nested assets and pixmap patterns. Wildcard patterns must match at least one file; full nine-slice completeness/rendering is not certified. Fonts are matched by embedded PF2 name (plus GRUB's customary Unifont name), not by their filename alone.

PNG (including palette/RGB/RGBA) and JPEG files are decoded with resource limits. Other image formats are reported unsupported. The PF2 inspector validates the section framing, signature, NAME, CHIX record sizing and DATA presence; it reports the font name. It is not a full glyph-renderer or an exhaustive PF2 verifier. Scripts/executable extensions produce errors and are never executed. A successful lint always includes `BOOT_UNVERIFIED`.

Receipts additionally retain selected variant, validation report, installed/active flags and a pin flag reserved for future update policy. Multiple immutable revisions coexist. No automatic version choice, update channel, remote registry trust or redistribution permission is inferred from a digest.
