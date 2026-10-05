# GRUB theme management ecosystem audit

Public-source review conducted 2026-10-04–2026-10-05, before implementation. Activity, adoption and feature assessments refer to the inspected revisions below. Current component choices are in the [reuse plan](reuse-plan.md); shipped behavior is in [architecture](architecture.md).

## Findings

The reviewed projects provide theme collections, installers, validation utilities and real-GRUB preview. None combined immutable theme revisions, verified installation plans, narrow privilege separation, distro-specific activation and recoverable transactions. GRUB Manager concentrates on that package-management layer, using distro GRUB tools, an external preview engine and standard libraries for storage and parsing.

The most useful source-adaptation candidate was sarbojitrana/grub-themes's lint/PF2 work. Gorgeous-GRUB provides discovery links. Package metadata and asset permissions require individual review.

## Method

- Sources included GitHub metadata and pinned files, Launchpad, package indexes, distro documentation, Polkit documentation and the OCS specification.
- The inventory includes four distinct GRUB Theme Managers. Both sarbojitrana/grub-themes and vinceliuice/grub2-themes are included. GNOME Theme Manager refers to unaibenidorm's GRUB-capable GTK application.
- Stars and forks measure repository attention. Comparable installation counts were unavailable. “Dormant” means the latest observed default-branch commit was more than a year old; none of the 15 GitHub repositories was archived at retrieval.
- Feature findings come from static review of selected source files, headers, trees and documentation. Upstream software was not executed; bootloader files were unchanged. Reported design concerns are static findings.
- Source links are pinned to inspected commits. Metadata and release pages are live links. Direct metadata took precedence over stale search extracts.

## Repository inventory

| ID / repository | Stars / forks | Last default-branch commit (UTC) | Latest GitHub release | Language | License evidence | Assessment |
| --- | ---: | --- | --- | --- | --- | --- |
| P01 [GRUB Theme Manager (ferdiizzulhaq)](https://github.com/ferdiizzulhaq/GRUB-Theme-Manager) | 1 / 0 | [2025-11-17 · 8da2d2e9](https://github.com/ferdiizzulhaq/GRUB-Theme-Manager/commit/8da2d2e9becee3f183c4a86c44fcefdd2cda76fc) | [v5.0](https://github.com/ferdiizzulhaq/GRUB-Theme-Manager/releases/tag/v5.0) (2025-11-17) | Python | MIT | Quiet; Fedora-focused |
| P02 [Grub Theme Manager (KHLALA-Gh)](https://github.com/KHLALA-Gh/grub-themes-manager) | 6 / 0 | [2026-01-23 · 69b99ee0](https://github.com/KHLALA-Gh/grub-themes-manager/commit/69b99ee09caef13deb2146fbe8cbcffc6a8f8b32) | None observed | Python | MIT | Quiet; small CLI |
| P03 [GRUB Theme Manager (kartikbhati011)](https://github.com/kartikbhati011/Grub-theme-manager) | 5 / 0 | [2026-05-30 · f0e23a63](https://github.com/kartikbhati011/Grub-theme-manager/commit/f0e23a634b7534ba50cf6313f82720d2fa79953d) | None observed | Shell | No root license; some themes separately licensed | Recent in 2026; small |
| P04 [GuideOS GRUB Theme Manager](https://github.com/GuideOS/guideos-grub-theme-manager) | 0 / 0 | [2026-05-02 · aeca3a34](https://github.com/GuideOS/guideos-grub-theme-manager/commit/aeca3a34cc4dfbae8ab194757e65cdc78e9b9d5f) | [1.0.3](https://github.com/GuideOS/guideos-grub-theme-manager/releases/tag/1.0.3) (2026-05-02) | Python | MIT | Distro-specific; small |
| P05 [GrubStudio](https://github.com/sooocil/grubstudio) | 6 / 2 | [2025-06-17 · a6ab96a2](https://github.com/sooocil/grubstudio/commit/a6ab96a286341d869f040b708060a39ed1862c62) | None observed | TypeScript | ISC declared in package.json; no license file | Dormant prototype |
| P06 [grub-themes (sarbojitrana)](https://github.com/sarbojitrana/grub-themes) | 3 / 0 | [2026-08-31 · ccf55b75](https://github.com/sarbojitrana/grub-themes/commit/ccf55b75eac8586e70d117de6d4e346e729cc418) | [v1.0.0](https://github.com/sarbojitrana/grub-themes/releases/tag/v1.0.0) (2026-08-28) | Go | MIT code; OFL font provenance needs checking | Recent; very young |
| P07 [theamify](https://github.com/DonArtkins/theamify) | 0 / 0 | [2026-08-29 · 22aae256](https://github.com/DonArtkins/theamify/commit/22aae2563cd4bd15d587a35bd94026738d0f8d2f) | None observed | JavaScript | MIT | Recent; very small |
| P08 [GNOME Theme Manager](https://github.com/unaibenidorm/Gnome-Theme-Manager) | 25 / 1 | [2026-09-30 · c603f7a6](https://github.com/unaibenidorm/Gnome-Theme-Manager/commit/c603f7a67ec1e7dc33e95b7b22bb8acbc026f3f9) | [Beta-5.0](https://github.com/unaibenidorm/Gnome-Theme-Manager/releases/tag/Beta-5.0) (2026-09-30) | Python | GPLv3; exact only/or-later grant needs confirmation | Recent; beta |
| P09 [Gorgeous-GRUB](https://github.com/Jacksaur/Gorgeous-GRUB) | 6,025 / 112 | [2026-08-24 · 5c389f76](https://github.com/Jacksaur/Gorgeous-GRUB/commit/5c389f769cba39b3f0f08469d59d1575634fe679) | None observed | Markdown / assets | No explicit repository license found | Recently curated; established audience |
| P10 [Gorgeous GRUB Installer](https://github.com/FLEXIY0/gorgeous-grub-installer) | 10 / 0 | [2026-01-15 · a2074410](https://github.com/FLEXIY0/gorgeous-grub-installer/commit/a207441057d6abd5f9b1603820fa00514696f5fe) | None observed | Shell | MIT | Quiet; very small |
| P11 [grub2-theme-preview](https://github.com/hartwork/grub2-theme-preview) | 431 / 15 | [2026-10-01 · 72cf7c82](https://github.com/hartwork/grub2-theme-preview/commit/72cf7c82c5d0a62090c75ebb7f71b09d66647fae) | [2.10.0](https://github.com/hartwork/grub2-theme-preview/releases/tag/2.10.0) (2026-05-24) | Python | GPL-2.0-or-later in source and setup.py | Maintained; maintenance mode |
| P12 [grub2-themes (vinceliuice)](https://github.com/vinceliuice/grub2-themes) | 4,641 / 289 | [2026-09-01 · 4c5a7712](https://github.com/vinceliuice/grub2-themes/commit/4c5a77125b93f833edc9bf7b14a899faa8ac79c6) | [2025-07-23](https://github.com/vinceliuice/grub2-themes/releases/tag/2025-07-23) (2025-07-23) | Shell | GPLv3; audit individual fonts/assets | Recent; established audience |
| P13 [GRUB-Tweaks](https://github.com/VandalByte/grub-tweaks) | 428 / 16 | [2024-12-12 · 5fa056c8](https://github.com/VandalByte/grub-tweaks/commit/5fa056c88517227f81031f076cd1560409de6006) | None observed | Markdown / assets | MIT | Dormant documentation |
| P14 [GRUB Background Cycler](https://github.com/Jacksaur/GRUB-Background-Cycler) | 19 / 2 | [2025-01-20 · 88f38b06](https://github.com/Jacksaur/GRUB-Background-Cycler/commit/88f38b065a27f861f5062de9a52466ca478fd9e4) | None observed | Shell | Informal reuse-with-credit grant; no standard license | Dormant small script |
| P15 [Dark Matter GRUB theme](https://github.com/VandalByte/darkmatter-grub2-theme) | 406 / 20 | [2026-06-14 · 42ed4b9e](https://github.com/VandalByte/darkmatter-grub2-theme/commit/42ed4b9e786b32946f1d14c4018d2215be061020) | None observed | Python | GPLv3; audit individual fonts/assets | Recent in 2026; established theme |

**P16 — [GRUB Customizer](https://git.launchpad.net/grub-customizer):** C++/GTKmm desktop application with a GRUB script/proxy model, CMake build and distro packaging. Latest observed upstream revision [805e8d31, 2026-05-07](https://git.launchpad.net/grub-customizer/commit/?id=805e8d3134a47afee2cc6712f58497136c892bca), identifying version 5.2.8. [Debian sid](https://packages.debian.org/sid/grub-customizer) also packages 5.2.8-1. Long-running since 2010 and present in distro repositories; installed-user counts are unknown. Launchpad declares GNU GPLv3; [Solus's package manifest](https://github.com/getsolus/packages/blob/main/packages/g/grub-customizer/package.yml) records GPL-3.0-or-later. Verify exact file grants/exceptions before any source reuse; COPYING was listed but its contents could not be retrieved through the web tool.

**P17 — [Pling / GNOME-Look GRUB catalog](https://www.gnome-look.org/browse?cat=109):** hosted catalog accessed through the language-neutral [OCS REST specification](https://wiki.freedesktop.org/www/Specifications/open-collaboration-services/). The entry covers a hosted service and protocol. The GNOME Theme Manager source supplies evidence of current OCS client integration; service availability, download counts, server implementation license and a latest server release were not established in this audit. Content licenses vary by upload. Each upload requires its own asset-license review.

The metadata for each GitHub row is reproducible through its repository endpoint, for example [Gorgeous-GRUB metadata][P09-meta], [grub-themes metadata][P06-meta] and [preview metadata][P11-meta]. The commit links fix the source revisions supporting the conclusions.

## Feature coverage

Legend: **S** = implemented in inspected source; **D** = documented; **P** = partial; **—** = absent from the inspected scope. Backup scope and privileged interfaces are assessed separately below.

| Project | Install / switch | Distro handling | Backup / rollback | Validation | Preview |
| --- | --- | --- | --- | --- | --- |
| P01 ferdiizzulhaq | S local folders | P Fedora paths | P defaults snapshots/manual restore | P file/substrings | S external QEMU tool |
| P02 KHLALA-Gh | S local selection | P command/path detection | — | P theme discovery | — |
| P03 kartikbhati011 | S bundled themes | P command detection | P one overwritten defaults backup | P theme file presence | — |
| P04 GuideOS | S local switching | P Debian/GuideOS assumptions | P one defaults backup | P discovery only | S image gallery |
| P05 GrubStudio | — host installation | — | — | P parser package scaffold | P static demo pane |
| P06 sarbojitrana | S bundled/local themes | P layout probing | S failure restore of two config files; incomplete asset rollback | S lint/PF2/assets | S approximate PNG/TUI |
| P07 theamify | S download/use/remove | P command/package-manager detection | P defaults snapshots | P theme.txt discovery | S terminal thumbnails |
| P08 GNOME Theme Manager | S online/local themes | P paths/commands | P defaults backup/manual restore | P archive checks/theme discovery | S GTK approximation |
| P09 Gorgeous-GRUB | D installation guide | D general guide | — | — | Screenshots, links to tools |
| P10 Gorgeous installer | S online/local operations | P path/tool assumptions | — transaction rollback found | P theme/install-script discovery | Catalog/interface, not verified GRUB renderer |
| P11 grub2-theme-preview | — | S preview dependency/firmware discovery | — | P observes real rendering, not full validator | S GRUB in QEMU |
| P12 vinceliuice | S collection installer | P several distro branches + Nix flake | P defaults backup/uninstall | P option checks | Screenshots |
| P13 GRUB-Tweaks | D recipes | D distro examples | D recovery recipes | — | D QEMU guide |
| P14 Background Cycler | Background rotation only | — | — | — | — |
| P15 Dark Matter | S theme installer | P several distro branches + Nix packaging | P reset/removal, not transaction rollback | P option checks | Screenshots |
| P16 GRUB Customizer | D theme/settings management | D configurable backend paths | D configuration recovery, not versioned theme rollback | P generated-config handling | Appearance controls, not verified QEMU integration |
| P17 OCS catalog | Downloads, not activation | Metadata, not backend | — | Upload metadata, not host validation | Uploaded media |

| Project | Online catalog / registry | Root / Polkit architecture | CLI | Theme updating / version tracking |
| --- | --- | --- | --- | --- |
| P01 | Links; gallery remains a TODO | pkexec utility calls | GUI; CLI TODO | — |
| P02 | Local directories | Process-level privileged operations | grub-tm | Tool update by git pull; no theme version receipts found |
| P03 | Bundled theme directories | Whole installer requires root | Interactive Bash | No immutable receipts found |
| P04 | Bundled/local directories | pkexec Bash command | GUI | Distro package updates; no per-theme receipts |
| P05 | Marketplace advertised; backend scaffold inspected | Browser/server scaffold | Development scripts | — |
| P06 | Local TOML manifests | Root apply/remove process | TUI + subcommands, dry-run | Manifest versions; no remote updater found |
| P07 | Editable pipe-delimited registry, Git cache | User downloads; sudo activation | Extensive command set | Tool self-update and theme re-download; no digest-pinned installed ledger found |
| P08 | OCS browsing + download variants | pkexec generated shell/Python scripts | GUI | Installed-theme metadata; immutable theme lock/upgrade transaction not found |
| P09 | Curated Markdown links | None | — | Git history of catalog, not installed versions |
| P10 | Embedded catalog and source-specific handlers | sudo, including upstream installer execution | Interactive shell UI | Git fetch/install; no version ledger found |
| P11 | None | Preview is unprivileged | Stable standalone CLI | Tool releases, not theme updates |
| P12 | Bundled variants | Root/sudo installer | Flags + dialog | Repository/releases; installed-version receipts not found |
| P13 | Links | Manual commands | Documentation | — |
| P14 | None | Recommends writable theme directory/startup job | Shell script | In-place background changes |
| P15 | Bundled variants | Root Python installer | Flags / interactive | Repository content; no versioned receipts found |
| P16 | No general online registry established | Elevated desktop application / packaged pkexec dependency | GUI launcher; not a theme-package CLI | Distro tool updates, not immutable community-theme versions |
| P17 | OCS content, downloads and metadata | No reason for catalog browsing to use root | HTTP API | Provider metadata; must independently lock downloaded content |

## Project assessments and reuse decisions

### P01 — ferdiizzulhaq/GRUB-Theme-Manager

Architecture: a single Python/PyQt5 application with a background worker for GRUB regeneration. Useful implemented features include local theme selection, manual defaults backups and invoking grub2-theme-preview.

**Potential source reuse:** MIT utility/UI fragments, with the Ferdi Izzulhaq copyright notice; review PyQt5's separate licensing before reusing the application architecture. There is no mature standalone backend here to adopt.

**Reimplement:** preview dependency diagnostics, installed/active theme distinction and a visible backup history.

**Limitations:** [source][P01-grub-theme-manager-v05-py] hard-codes /boot/grub2 and Fedora's generator, checks theme validity mainly by file presence/substrings, and backs up /etc/default/grub without preserving a complete theme revision. It copies staged configuration through pkexec and a predictable temporary filename. Use a dedicated helper and private staging instead. Preview options advertised in the UI exceed the options forwarded by preview_theme. Cross-distro integration would require a separate backend.

### P02 — KHLALA-Gh/grub-themes-manager

Architecture: small Python CLI with separate configuration and theme modules; configured local theme directories, interactive choice and direct installation.

**Potential source reuse:** MIT discovery/configuration utilities under Khalil's notice, if useful after review. Their size makes independent implementation inexpensive.

**Reimplement:** editable search paths, concise CLI operations and explicit missing-tool messages.

**Limitations:** [lib/theme.py][P02-lib-theme-py] selects a generator based on executable availability and writes defaults before regeneration. No compensating transaction or full backup is visible in that path. The [README][P02-README-md] calls git pull plus reinstall an update; this updates the application, not independently tracked theme versions. Generalizing its command-name detection would not establish boot-layout compatibility.

### P03 — kartikbhati011/Grub-theme-manager

Architecture: one Bash installer plus approximately 35 bundled third-party theme directories.

**Potential source reuse:** none approved. The [tree][P03-tree] has no root license, although several bundled themes contain their own licenses. Identify each theme's original upstream and rights separately.

**Reimplement:** theme selection and a visible active-theme indicator.

**Limitations:** [install.sh][P03-install-sh] runs as root, overwrites a single defaults backup, and regenerates the selected target directly. No immutable package receipts, reference validation, isolated preview or transactional rollback was found. Bundled themes require individual provenance review.

### P04 — GuideOS/guideos-grub-theme-manager

Architecture: Python/GTK3 local theme gallery with Debian packaging and bundled GuideOS themes.

**Potential source reuse:** MIT application fragments under Nightworker's notice; bundled images still require their own provenance check.

**Reimplement:** searchable gallery, active-theme badge and fast local discovery.

**Limitations:** [on_apply][P04-guideos-grub-theme-py] assumes /boot/grub/themes and update-grub, creates one defaults backup, then authorizes a generated Bash command. It also enables OS probing while applying appearance settings. This couples appearance with boot policy. Its gallery uses static images. No theme version database or complete rollback is present in the inspected application.

### P05 — sooocil/grubstudio

Architecture: TypeScript monorepo with Next.js/React/Monaco, an Express API and Prisma schema.

**Potential source reuse:** hold. The [root package manifest][P05-package-json] declares ISC, but no license file was found. Grant and notice coverage need clarification before source reuse.

**Reimplement:** editor/file tree/preview layout and an eventual authoring workflow.

**Limitations:** [EditorContent.tsx][P05-apps-web-src-components-editorComponents-EditorContent-tsx] renders a static boot-menu demo. [server.ts][P05-apps-api-src-server-ts] is a greeting endpoint, and [theme-parser's package][P05-packages-theme-parser-package-json] is a scaffold without parser implementation in the inspected tree. The advertised renderer and marketplace were unimplemented in the inspected revision. Dormant since June 2025; unsuitable as a validation, rendering or registry dependency in its inspected state.

### P06 — sarbojitrana/grub-themes

Architecture: Go CLI/Bubble Tea TUI with TOML theme metadata, asset generation, a PF2 reader, linting, approximate image rendering and an installer.

**Potential source reuse:** candidate for selected [lint][P06-internal-lint-lint-go] and [PF2][P06-internal-pf2-pf2-go] routines under MIT, preserving Sarbojit Rana's notice and upstream tests. Adapting these internal packages requires extraction and independent validation. Font assets have separate OFL/provenance obligations.

**Reimplement:** structured findings, manifest metadata, dry-run, and preservation checks for boot entries/custom configuration.

**Limitations:** [installer][P06-internal-install-install-go] can fall back to an arbitrary EFI grub.cfg, replaces theme assets before completion, and restores only two configuration files. Entry-count/regex checks do not establish boot equivalence. The linter's universal PNG restriction and flat-directory assumptions need target-version validation. The [preview parser][P06-internal-preview-parse-go] explicitly describes a layout approximation. Recent project with [open rendering reports](https://github.com/sarbojitrana/grub-themes/issues/2); source adaptation would need independent tests.

### P07 — DonArtkins/theamify

Architecture: npm-distributed JavaScript CLI/wizard delegating to a bundled Bash engine, editable registry and Git/theme caches.

**Potential source reuse:** MIT command/configuration utilities under Don Artkins's notice, conditional on dependency/license review. Prefer the UX ideas over adopting the engine.

**Reimplement:** get/use separation, doctor, searchable registry, caching, explicit application upgrade versus theme upgrade.

**Limitations:** [grub.sh][P07-vendor-lib-grub-sh] selects Fedora's EFI stub when dnf is present, overwrites existing theme files before defaults backup and does not restore on regeneration failure. [themes.sh][P07-vendor-lib-themes-sh] can execute a downloaded generator. Its updates refresh content rather than locking immutable installed revisions. [self.js][P07-src-lib-self-js] performs npm self-updates; application updates need separate handling from theme revisions. Preview uses thumbnails.

### P08 — unaibenidorm/Gnome-Theme-Manager

Architecture: Python/GTK4/Libadwaita GUI, OCS client, archive installer, theme library and GRUB/Plymouth editors.

**Potential source reuse:** GPLv3 client, GUI and archive code would require a compatible combined-work license. Source import was not selected; an independent client can implement the public OCS protocol.

**Reimplement:** [OCS browsing][P08-gnome_theme_manager-api-py], bounded caching, variants, dependency diagnostics, archive quotas and preview/installed states.

**Limitations:** [installer.py][P08-gnome_theme_manager-installer-py] adds explicit ZIP/TAR checks, but its other extractor paths do not share all those checks. Privileged work uses generated shell/Python scripts through pkexec, giving those scripts the privileged interface. [GRUB backup/restore][P08-gnome_theme_manager-grub_customizer-py] preserves defaults but omits immutable assets and transaction state. Its graphical preview approximates GRUB. Beta-5.0 expands the application beyond the versions described in older search summaries.

### P09 — Jacksaur/Gorgeous-GRUB

Architecture: Markdown catalog, screenshots, installation instructions and links to independently hosted themes/tools.

**Potential source reuse:** no blanket copy approved. The [pinned tree][P09-tree] has no repository license. Screenshots, descriptions and linked themes require separate rights review.

**Reimplement:** curated discovery, useful categories, clear author/upstream links and contributor submissions. Use upstream project names/URLs as research leads and independently create verified package metadata.

**Limitations:** no machine-readable version schema, artifact hashes, uniform download format, installed state or compatibility policy. A theme update is independent of catalog Git history. The [installation guide][P09-Installation-md] documents manual installation. Catalog mirroring would require a metadata agreement.

### P10 — FLEXIY0/gorgeous-grub-installer

Architecture: large Bash script with optional gum UI, embedded catalog and different handlers for upstream repository layouts.

**Potential source reuse:** MIT interface/metadata utilities, preserving the LICENSE copyright holder **droopi**. Linked artwork has separate terms.

**Reimplement:** search, source adapters and friendly progress/errors.

**Limitations:** the [generic GitHub installer][P10-gorgeous-grub-sh] explicitly runs an upstream install.sh with sudo when present. Other paths mutate grubenv/custom-menu files. Source-specific handling illustrates packaging diversity but offers no clear immutable revision, digest validation or transaction boundary. These execution paths are outside grubmgr's data-only package model.

### P11 — hartwork/grub2-theme-preview

Architecture: Python command-line tool that builds a temporary rescue image and boots real GRUB in QEMU/KVM, with EFI/OVMF discovery and an optional software-emulation path.

**Reuse decision:** preferred optional external preview engine. [setup.py][P11-setup-py] and [source header][P11-grub2_theme_preview-__main__-py] explicitly grant GPL-2.0-or-later even though GitHub did not detect a top-level license. Prefer separately installed packages and a documented CLI boundary; do not silently vendor its implementation.

**Reimplement:** dependency diagnostics and a small lifecycle wrapper for timeout/cancellation, approved paths and structured results.

**Limitations:** [maintenance mode][P11-README-md] since May 2026 means bug/dependency fixes continue but substantial feature PRs are not accepted. The observed release was 2.10.0. Its host-config default and QEMU networking defaults require an adapter that supplies a generated menu and isolation. Preview verifies guest rendering; host firmware and storage compatibility need separate tests.

### P12 — vinceliuice/grub2-themes

Architecture: popular shell/dialog installer, theme/asset templates, resolution/icon variants, font tools and a Nix flake.

**Potential reuse:** theme packages/assets after per-file license verification; the [GPLv3 license][P12-LICENSE] constrains code combinations, and fonts can have separate terms. Prefer upstream release artifacts or a separately reviewed offline asset recipe, not privileged execution of the installer.

**Reimplement:** variant metadata, resolution selection and explicit theme-root paths.

**Limitations:** [install.sh][P12-install-sh] contains distro branches but still falls back to Fedora's EFI configuration when the primary file is missing. That fallback can select a forwarding stub. Backup and uninstall handling omit immutable per-version rollback. The collection is useful for testing theme-layout diversity.

### P13 — VandalByte/grub-tweaks

Architecture: Markdown recipes covering themes, fonts, layout, preview and recovery.

**Potential reuse:** appropriately attributed MIT documentation fragments under Vandal's notice; inspect linked/copied material separately. Prefer original grubmgr instructions derived from distro documentation.

**Reimplement:** searchable troubleshooting categories and actionable dependency checks.

**Limitations:** dormant documentation; [recipes][P13-README-md] include edits to distro-generated scripts and boot repair. Those operations exceed theme-management scope. No registry, application architecture, transaction engine or validation library exists here.

### P14 — Jacksaur/GRUB-Background-Cycler

Architecture: tiny Bash script rotating numbered PNGs in place, intended for a startup task.

**Potential reuse:** the [README][P14-README-md] explicitly allows inclusion with credit, but there is no standard license defining broader modification/distribution terms. Hold code reuse pending clarification; the grant needs clarification for the intended distribution.

**Reimplement:** background rotation through a derived immutable revision, if added later.

**Limitations:** [Cycler.sh][P14-Cycler-sh] renames live assets in place without transaction handling. Instructions recommend user ownership of a boot-theme directory. That conflicts with grubmgr's root-owned activated artifacts. Dormant; rotation is deferred.

### P15 — VandalByte/darkmatter-grub2-theme

Architecture: Python installer for a finite collection of styles/icons/resolutions, with Nix packaging in the repository.

**Potential reuse:** GPLv3 theme content subject to individual asset/font provenance. No installer code import is selected.

**Reimplement:** a compact variant schema and theme-package presentation.

**Limitations:** [change_grub_theme][P15-darkmatter-theme-py] disables GRUB_ENABLE_BLSCFG when that setting exists. Altering Boot Loader Specification behavior to apply a theme changes boot semantics. The root-level installer and reset/removal workflow do not provide a versioned rollback contract.

### P16 — GRUB Customizer

Architecture: mature C++/GTKmm editor with configurable paths and script/proxy machinery for manipulating generated boot menus. Its [upstream explanation](https://answers.launchpad.net/grub-customizer/+faq/1355) describes proxy scripts and changes under /etc/grub.d; interoperability needs verification against the target packaged version.

**Potential reuse:** GPL-compatible source is theoretically available after exact grant verification, but no core reuse is selected. Its broader boot-menu mutation model does not fit this product.

**Reimplement:** separating proposed edits from saving, displaying configuration changes and clearly exposing recovery.

**Limitations:** Its boot-entry proxy layer and MBR-reinstallation features exceed theme-management scope. Proxy-managed configurations require separate interoperability tests. Its [May 2026 upstream fix](https://git.launchpad.net/grub-customizer/commit/?id=805e8d3134a47afee2cc6712f58497136c892bca) addresses changed Ubuntu memtest output: illustrating the maintenance required for boot-script transformations. Upstream was maintained at the audit date.

### P17 — OCS / Pling / GNOME-Look

Architecture: hosted content/variant/download metadata with an established REST protocol and XML/JSON responses.

**Reuse:** the public protocol via a fresh small HTTP adapter; use a maintained HTTP library rather than copying a GPL application client. If choosing a dedicated OCS library later, audit its exact version/license separately.

**Reimplement:** source identity mapping, request caching/backoff, source links, variant selection and normalized package records.

**Limitations:** mutable URLs and provider metadata are not immutable package receipts. Content, thumbnail and font rights must be verified per artifact. No assumption of cryptographic integrity, reproducibility or distro compatibility follows from listing a theme. Keep unknown/unreviewed results browse-only.

## Cross-project findings

1. **Configuration backup is not package rollback.** Defaults-only snapshots omit assets, selected revisions, generated output and drift from later kernel updates. Even P06's two-file rollback can leave replaced/deleted theme assets behind.
2. **Distro-aware means more than locating a command.** Firmware layout, actual mounts, config forwarding stubs, BLS, generated fragments, immutable systems and package ownership determine what can safely change.
3. **Data-only packages reduce installer scope.** Community install/build hooks may perform arbitrary operations. A package manager can ingest validated assets without accepting those scripts as trusted root actions.
4. **Polkit supplies authorization, not input validation.** The [pkexec manual](https://polkit.pages.freedesktop.org/polkit/pkexec.1.html) explicitly leaves argument validation to the program. A generic shell behind a password prompt is not the intended helper interface.
5. **Preview levels must be honest.** Label screenshots, approximate previews and actual GRUB-in-QEMU separately. Use [GNU's theme format](https://www.gnu.org/software/grub/manual/grub/html_node/Theme-file-format.html) for validation rules. The [GRUB script checker](https://manpages.debian.org/unstable/grub-common/grub-script-check.1.en.html) checks GRUB scripts, not complete theme/asset correctness.
6. **Version labels alone are insufficient.** A timestamp, branch name or manifest version must be resolved to an immutable revision and verified artifact digest. Tool self-update, catalog refresh, theme fetch and boot activation are different operations.
7. **License discovery must inspect actual files.** P11's GPL grant is in headers; P05 declares ISC in metadata; P09 has no general grant; P14 has informal permission. Repository metadata alone would misclassify them.

## Distro-specific constraints to carry into the design

| Environment | Evidence / concern | grubmgr requirement |
| --- | --- | --- |
| Fedora 34+ conventional GRUB | [Fedora documentation](https://fedoraproject.org/wiki/GRUB_2) identifies the EFI file as a forwarding stub | Validate the `/boot/grub2` output path and preserve the EFI stub and BLS |
| Debian/Ubuntu and derivatives | GRUB defaults/fragments and derivative customizations can differ | Detect effective configuration and supported generator behavior; preserve unrelated settings |
| Arch | [Arch theme guidance](https://wiki.archlinux.org/title/GRUB/Tips_and_tricks) notes theme paths must be readable before OS boot | Validate boot-visible storage; an existing /usr/share path is insufficient |
| openSUSE | [openSUSE documentation](https://doc.opensuse.org/documentation/leap/reference/html/book-reference/cha-grub2.html) describes generated configuration; snapshot/BLS variants require separate coverage | Separate tested layouts; do not silently change snapshot or boot-entry machinery |
| NixOS | [Official options](https://nixos.org/manual/nixos/unstable/options) expose boot.loader.grub.theme | Provide a declarative theme export/module |
| Atomic/transactional systems | Mutability and boot ownership differ from conventional installs | Read-only until a dedicated integration is tested |
| Secure Boot / encrypted or separate boot storage | Guest preview and OS-side file access do not establish host support | Test compatibility while preserving Secure Boot and encryption/module policy |
| Non-GRUB or ambiguous multi-boot systems | GRUB files may exist without being the active loader | Refuse activation; no automatic bootloader installation or migration |

## Reuse review

The audit imported no code or assets. MIT candidates require retained copyright and permission notices. GPL source requires compatibility review for the intended combined work; external tools were the preferred integration. Missing or unclear grants require permission before copying protected material. See [GNU's aggregation guidance](https://www.gnu.org/licenses/gpl-faq.en.html#MereAggregation) and [GitHub's licensing guidance](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/licensing-a-repository).

The subsequent implementation uses MIT for original code, an original validator/PF2 inspector, and the external preview CLI. Actual dependencies and notices are recorded in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md). The [reuse plan](reuse-plan.md) tracks remaining catalog and distro work.

[P01-grub-theme-manager-v05-py]: https://github.com/ferdiizzulhaq/GRUB-Theme-Manager/blob/8da2d2e9becee3f183c4a86c44fcefdd2cda76fc/grub-theme-manager-v05.py

[P02-README-md]: https://github.com/KHLALA-Gh/grub-themes-manager/blob/69b99ee09caef13deb2146fbe8cbcffc6a8f8b32/README.md
[P02-lib-theme-py]: https://github.com/KHLALA-Gh/grub-themes-manager/blob/69b99ee09caef13deb2146fbe8cbcffc6a8f8b32/lib/theme.py

[P03-tree]: https://github.com/kartikbhati011/Grub-theme-manager/tree/f0e23a634b7534ba50cf6313f82720d2fa79953d
[P03-install-sh]: https://github.com/kartikbhati011/Grub-theme-manager/blob/f0e23a634b7534ba50cf6313f82720d2fa79953d/install.sh

[P04-guideos-grub-theme-py]: https://github.com/GuideOS/guideos-grub-theme-manager/blob/aeca3a34cc4dfbae8ab194757e65cdc78e9b9d5f/guideos-grub-theme.py

[P05-package-json]: https://github.com/sooocil/grubstudio/blob/a6ab96a286341d869f040b708060a39ed1862c62/package.json
[P05-apps-web-src-components-editorComponents-EditorContent-tsx]: https://github.com/sooocil/grubstudio/blob/a6ab96a286341d869f040b708060a39ed1862c62/apps/web/src/components/editorComponents/EditorContent.tsx
[P05-apps-api-src-server-ts]: https://github.com/sooocil/grubstudio/blob/a6ab96a286341d869f040b708060a39ed1862c62/apps/api/src/server.ts
[P05-packages-theme-parser-package-json]: https://github.com/sooocil/grubstudio/blob/a6ab96a286341d869f040b708060a39ed1862c62/packages/theme-parser/package.json

[P06-meta]: https://api.github.com/repos/sarbojitrana/grub-themes
[P06-internal-install-install-go]: https://github.com/sarbojitrana/grub-themes/blob/ccf55b75eac8586e70d117de6d4e346e729cc418/internal/install/install.go
[P06-internal-lint-lint-go]: https://github.com/sarbojitrana/grub-themes/blob/ccf55b75eac8586e70d117de6d4e346e729cc418/internal/lint/lint.go
[P06-internal-preview-parse-go]: https://github.com/sarbojitrana/grub-themes/blob/ccf55b75eac8586e70d117de6d4e346e729cc418/internal/preview/parse.go
[P06-internal-pf2-pf2-go]: https://github.com/sarbojitrana/grub-themes/blob/ccf55b75eac8586e70d117de6d4e346e729cc418/internal/pf2/pf2.go

[P07-src-lib-self-js]: https://github.com/DonArtkins/theamify/blob/22aae2563cd4bd15d587a35bd94026738d0f8d2f/src/lib/self.js
[P07-vendor-lib-grub-sh]: https://github.com/DonArtkins/theamify/blob/22aae2563cd4bd15d587a35bd94026738d0f8d2f/vendor/lib/grub.sh
[P07-vendor-lib-themes-sh]: https://github.com/DonArtkins/theamify/blob/22aae2563cd4bd15d587a35bd94026738d0f8d2f/vendor/lib/themes.sh

[P08-gnome_theme_manager-installer-py]: https://github.com/unaibenidorm/Gnome-Theme-Manager/blob/c603f7a67ec1e7dc33e95b7b22bb8acbc026f3f9/gnome_theme_manager/installer.py
[P08-gnome_theme_manager-grub_customizer-py]: https://github.com/unaibenidorm/Gnome-Theme-Manager/blob/c603f7a67ec1e7dc33e95b7b22bb8acbc026f3f9/gnome_theme_manager/grub_customizer.py
[P08-gnome_theme_manager-api-py]: https://github.com/unaibenidorm/Gnome-Theme-Manager/blob/c603f7a67ec1e7dc33e95b7b22bb8acbc026f3f9/gnome_theme_manager/api.py

[P09-meta]: https://api.github.com/repos/Jacksaur/Gorgeous-GRUB
[P09-tree]: https://github.com/Jacksaur/Gorgeous-GRUB/tree/5c389f769cba39b3f0f08469d59d1575634fe679
[P09-Installation-md]: https://github.com/Jacksaur/Gorgeous-GRUB/blob/5c389f769cba39b3f0f08469d59d1575634fe679/Installation.md

[P10-gorgeous-grub-sh]: https://github.com/FLEXIY0/gorgeous-grub-installer/blob/a207441057d6abd5f9b1603820fa00514696f5fe/gorgeous-grub.sh

[P11-meta]: https://api.github.com/repos/hartwork/grub2-theme-preview
[P11-README-md]: https://github.com/hartwork/grub2-theme-preview/blob/72cf7c82c5d0a62090c75ebb7f71b09d66647fae/README.md
[P11-setup-py]: https://github.com/hartwork/grub2-theme-preview/blob/72cf7c82c5d0a62090c75ebb7f71b09d66647fae/setup.py
[P11-grub2_theme_preview-__main__-py]: https://github.com/hartwork/grub2-theme-preview/blob/72cf7c82c5d0a62090c75ebb7f71b09d66647fae/grub2_theme_preview/__main__.py

[P12-LICENSE]: https://github.com/vinceliuice/grub2-themes/blob/4c5a77125b93f833edc9bf7b14a899faa8ac79c6/LICENSE
[P12-install-sh]: https://github.com/vinceliuice/grub2-themes/blob/4c5a77125b93f833edc9bf7b14a899faa8ac79c6/install.sh

[P13-README-md]: https://github.com/VandalByte/grub-tweaks/blob/5fa056c88517227f81031f076cd1560409de6006/README.md

[P14-README-md]: https://github.com/Jacksaur/GRUB-Background-Cycler/blob/88f38b065a27f861f5062de9a52466ca478fd9e4/README.md
[P14-Cycler-sh]: https://github.com/Jacksaur/GRUB-Background-Cycler/blob/88f38b065a27f861f5062de9a52466ca478fd9e4/Cycler.sh

[P15-darkmatter-theme-py]: https://github.com/VandalByte/darkmatter-grub2-theme/blob/42ed4b9e786b32946f1d14c4018d2215be061020/darkmatter-theme.py
