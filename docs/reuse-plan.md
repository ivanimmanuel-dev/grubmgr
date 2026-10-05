# Reuse decisions and development plan

Based on the [ecosystem audit](ecosystem-audit.md) completed on 2026-10-05. This document records component choices and remaining work for `0.3.0-rc.1`; [architecture](architecture.md) describes the implementation.

GRUB Manager provides package identity, compatibility checks, plans and recoverable activation around an existing GRUB installation. Compression, storage, authorization, configuration generation and rendering use established libraries or system tools.

## Components

| Component | Decision | Integration and license |
| --- | --- | --- |
| Go standard library | Use | Archive decoding, TLS, HTTP, hashing, JSON and image decoding; BSD-3-Clause notices retained |
| SQLite | Use through `modernc.org/sqlite` | Receipts and transaction journals; BSD-3-Clause driver and public-domain SQLite, with binding dependencies inventoried |
| GNU GRUB | Use distro tools | Fixed generator and script-checker paths selected by each Linux profile |
| Polkit | Use distro authorization | Fixed installed helper and `auth_admin` policy; terminal authentication through `pkttyagent` |
| [grub2-theme-preview](https://github.com/hartwork/grub2-theme-preview) | Use optional external CLI | Version 2.10.0; GPL-2.0-or-later, Copyright Sebastian Pipping; separately installed |
| Bubblewrap, QEMU, OVMF, xorriso, mtools | Use external preview tools | Isolated renderer with a generated menu, fixed drive policy and resource limits |
| [sarbojitrana/grub-themes](https://github.com/sarbojitrana/grub-themes) | Reference | MIT lint/PF2 code was considered; the current validator and PF2 inspector are original |
| [Gorgeous-GRUB](https://github.com/Jacksaur/Gorgeous-GRUB) | Discovery source | Use factual upstream links to prepare individual recipes; catalog media and descriptions need permission before mirroring |
| [OCS](https://wiki.freedesktop.org/www/Specifications/open-collaboration-services/) | Future protocol adapter | Provider metadata would feed discovery; reviewed recipes would remain the activation authority |
| GNOME Theme Manager | Interface and OCS reference | GPLv3 application; source import not selected |
| theamify and the small MIT managers | Interface reference | Search, doctor, variants and separate fetch/apply operations; installer engines not selected |
| GrubStudio | Editor-layout reference | Prototype; ISC declaration needs notice clarification before source reuse |
| GRUB Customizer | Compatibility reference | Detect proxy-managed configurations; its boot-entry editing model is outside the theme manager |
| vinceliuice, Dark Matter and other theme collections | Individual package candidates | Review each artifact, font and artwork license before enabling imports |
| Background Cycler | Deferred | Rotation would require a derived immutable revision rather than editing active assets |

No complete manager was selected as a fork. The original engineering is the layer joining these components: reviewed package recipes, immutable revisions, distro profiles, a constrained helper, transaction journals and rollback with current kernels.

## Attribution and dependency review

Original application code is MIT. The [dependency ledger](../third_party/dependencies.json) records linked components, exact versions, sources, integration and notice digests. [Third-party notices](../THIRD_PARTY_NOTICES.md) describes the shipped dependencies and external tools.

Before importing source or enabling a theme recipe:

1. Record the upstream release or commit, file scope, copyright holders, license expression and modifications.
2. Check the license for the intended integration and distribution, including transitive dependencies and asset exceptions.
3. Preserve file headers and full notices in source, packages and installed theme revisions.
4. Recheck changed files and grants on updates; keep unclear assets unavailable for activation.

MIT candidates include utilities by Ferdi Izzulhaq, Khalil, Nightworker, Don Artkins and droopi, and lint/PF2 work by Sarbojit Rana. None is currently incorporated. The [audit](ecosystem-audit.md#project-assessments-and-reuse-decisions) links the inspected revisions and license evidence.

External program invocation is the selected integration for GPL tools. Any future redistribution still requires review of those tools' terms. Theme assets keep their own licenses; repository visibility and catalog membership do not establish permission to redistribute them.

## Package-management layer

The current implementation follows these decisions:

- **Identity:** separate upstream version, recipe revision, artifact digest, tree digest and manifest revision. Keep exact file inventories and notices.
- **Discovery:** ship a reviewed catalog snapshot. Unreviewed entries can link to upstream projects while remaining unavailable for activation.
- **Ingestion:** import data through bounded ZIP/TAR or directory handling. Require explicit roots for ambiguous packages and digests for HTTPS artifacts.
- **Validation:** check the supported theme grammar, references, images and PF2 names. Use GRUB in QEMU for visual verification.
- **Planning:** bind the requested change to package identity, variant and current system state. Show destinations, settings, privilege and recovery scope before applying.
- **Activation:** have the helper independently verify the package and plan, coordinate package-manager locks, stage assets, generate a candidate and replace validated configuration.
- **Recovery:** distinguish immediate restoration of transaction snapshots from later theme rollback using current kernels. Retain assets and unresolved journals.

See [package format](package-format.md), [architecture](architecture.md) and [security model](security-model.md) for the corresponding contracts.

## Distro integration

Profiles define distro identity, GRUB version, executable paths, package-manager coordination and boot layout. Detection combines these checks with firmware, mounts, defaults, fragments and ownership. An `ID_LIKE` value or installed command alone cannot select an activation profile.

Debian, Ubuntu, Kali and Arch have tested VM profiles. Fedora is detection-only. Additional integrations need separate evidence:

| Target | Required work |
| --- | --- |
| Fedora | Preserve the EFI forwarding stub, BLS and `grubenv`; validate the `/boot/grub2` generation path |
| openSUSE | Cover snapshot/custom-menu integration and distinguish supported GRUB layouts |
| NixOS | Export a pinned package and declarative theme setting |
| Atomic systems | Define ownership and coordination with the system update mechanism |
| BIOS or Secure Boot | Establish firmware, font and boot-readability coverage |
| GRUB Customizer | Test interoperability with its script/proxy configuration |
| Physical machines | Review the helper and recovery model, then test documented hardware and storage combinations |

The helper changes `GRUB_THEME`. Boot-entry editing, kernel options, default OS, timeout, partitioning and firmware policy are outside this scope.

## Remaining work

| Area | Next step | Acceptance evidence |
| --- | --- | --- |
| Community packages | Add reviewed recipes with complete asset provenance and varied layouts | Import, validation, preview, activation and notice retention in the relevant VM profiles |
| Theme updates | Add candidate comparison, explicit revision selection and pin controls | Changed source/license metadata requires review; fetching leaves the active version unchanged |
| Catalog refresh | Implement provider adapters and review a maintained metadata-update framework such as TUF | Integrity and freshness checks, bounded redirects, provider identity and cached rollback data |
| Retention | Add a dry-run collector for manager-owned, unreferenced revisions | Active themes, pending transactions and rollback references remain intact |
| Recovery | Extend interruption, storage-failure and offline-recovery coverage | Recoverable state or an explicit retained conflict at each tested boundary |
| GUI | Build a frontend over the existing core and plan protocol | CLI and GUI produce equivalent validation and transaction results |

The [test guide](testing.md) defines ordinary checks; [VM testing](vm-testing.md) covers authorization, generation, failure recovery and reboots. The [Linux verification record](linux-verification.md) identifies completed runs and their limits.

Theme authoring, background rotation, arbitrary generators, cloud accounts, boot repair, boot-entry reordering and unattended activation remain outside the current release.
