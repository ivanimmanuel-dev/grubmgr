# Third-party notices

Original GRUB Manager code and demo assets are Copyright (c) 2026 GRUB Manager contributors, under the [MIT license](LICENSE).

## Linked dependencies

The following unmodified Go modules are pinned in `go.mod` and verified through `go.sum`. [dependencies.json](third_party/dependencies.json) records their source locations, integration and notice hashes. Full copyright and license texts, including secondary Go, mmap, netdb, musl and embedded notices, are preserved in [third_party/licenses](third_party/licenses).

| Component | Version | License / purpose |
| --- | --- | --- |
| modernc.org/sqlite | v1.38.2 | BSD-3-Clause driver; SQLite deliverable public domain; local SQL state |
| modernc.org/libc | v1.66.3 | BSD-3-Clause plus MIT musl/netdb and secondary permissive notices; SQLite support |
| modernc.org/mathutil | v1.7.1 | BSD-3-Clause |
| modernc.org/memory | v1.11.0 | BSD-3-Clause and included Go/mmap notices |
| github.com/dustin/go-humanize | v1.0.1 | MIT |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause |
| github.com/mattn/go-isatty | v0.0.20 | MIT |
| github.com/ncruces/go-strftime | v0.1.9 | MIT |
| github.com/remyoudompheng/bigfft | v0.0.0-20230129092748-24d4a6f8daec | BSD-3-Clause; immutable commit in module version |
| golang.org/x/exp | v0.0.0-20250620022241-b7579e27df2b | BSD-3-Clause; immutable commit in module version |
| golang.org/x/sys | v0.34.0 | BSD-3-Clause; OS APIs, including hardlink checks |
| Go runtime and standard library | go1.27.1 build | BSD-3-Clause; archive/TLS/HTTP/JSON/image/hash primitives |

`go run ./tools/checklicenses` checks module versions and notice hashes. Dependency upgrades require a review of changed files and embedded notices. The compiler and development-only module-cache tools are excluded from release archives.

## External programs

`grub2-theme-preview` 2.10.0 is a separately installed CLI, Copyright Sebastian Pipping, licensed GPL-2.0-or-later. GRUB Manager invokes it through an isolated preview adapter. GRUB, Polkit, Bubblewrap, QEMU, OVMF, xorriso, mtools and the renderer are supplied separately by the system or test setup.

## Theme assets

The built-in demo includes its full MIT notice in every imported and installed revision.

The `debian/starfield` recipe imports Debian's installed `grub-theme-starfield` 2.12-9+deb13u2 package. Every revision preserves `theme.txt`, `README`, `DEBIAN-COPYRIGHT`, `FONT-COPYRIGHT` and `GPL-3`. The notices identify the layout as MIT, artwork as CC-BY-SA-3.0, and DejaVu font data as Bitstream-Vera with public-domain changes. The [catalog](internal/catalog/catalog.json) records its source and content digest.
