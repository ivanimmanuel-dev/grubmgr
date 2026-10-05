# Third-party notices and provenance

Original GRUB Manager code and synthetic fixture assets: Copyright (c) 2026 GRUB Manager contributors, MIT; see [LICENSE](LICENSE).

## Actual reuse

No source from the audited GRUB theme managers was copied or adapted. In particular, no `sarbojitrana/grub-themes` linter/PF2 source or installer was incorporated; our small PF2 section inspector and validation subset are independently implemented. Its audit commit remains a research reference, not a code dependency. No GPL program source, Gorgeous-GRUB descriptions/screenshots, Minegrub assets, or other community artwork is bundled.

The following are linked as **unmodified Go modules**, pinned in `go.mod` and integrity-checked in `go.sum`. File selection depends on the build target. The full source module is available at the versioned package link; no vendor modifications are present. Full original copyright/license texts, including secondary Go/mmap/netdb/musl notices and embedded permissive license blocks, are preserved in [third_party/licenses](third_party/licenses). [dependencies.json](third_party/dependencies.json) records exact versions, source locations, integration and notice digests.

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

The dependency license checker detects changed/missing notices and module/version additions. It is an inventory gate, not an automated legal opinion or a substitute for reviewing changes. If a target/dependency is added, inspect embedded notices as well as top-level licenses before releasing binaries. Development-only transitive tools in the module cache and the portable Go compiler are not included in the release archives.

`grub2-theme-preview` is a future independent optional external CLI (upstream GPL-2.0-or-later); neither it nor QEMU/firmware is distributed or executed here. Separately packaging those tools requires their own licenses and compliance review.

The builtin demo's full MIT license stays inside every imported and installed revision. Imported theme notices are never replaced with grubmgr's license. Unknown rights remain unknown. The catalogue's Minegrub entry is an independently written factual project link and is browse-only; it confers no asset-copying permission.
