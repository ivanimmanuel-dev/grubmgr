# GRUB Manager 0.1.0 verification

Completed **2026-10-05**, America/Toronto.

## Delivered

The shared Go core and CLI implement catalogue browsing, local and reviewed HTTPS imports, immutable manifests and SHA-256 identities, bounded ZIP/TAR extraction, theme/image/PF2 validation, rooted system detection, SQLite receipts, read-only plan tokens, and real fixture transactions with failure recovery. JSON output, optional structured logs, XDG paths and stable exit codes are included.

`install`, `switch`, `remove` and later theme `rollback` work against marked synthetic Debian roots. Install does not activate. Removal retains assets for rollback. The source tree includes five distro fixtures, tests, documentation, license notices and Linux/Windows GitHub Actions.

## Checks actually run

| Check | Result |
| --- | --- |
| Windows amd64, Go 1.27.1: complete `go test -count=1 -json ./...` | **83 named test/subtest cases passed**, zero failures |
| Linux amd64 in Ubuntu WSL, Go 1.27.1: same complete suite | **86 named test/subtest cases passed**, zero failures, no named test skips |
| Windows skips | One case-sensitive directory test and two symlink tests; ZIP collision tests and hardlink tests passed. All three skipped cases passed on Linux |
| `go vet ./...` | Passed on Windows and Linux |
| Formatting (`gofmt -l cmd internal tools`) | Clean |
| `go mod verify` | Passed; pinned modules verified |
| `go run ./tools/checklicenses` | Passed on Windows and Linux; pinned dependency versions and preserved notice hashes verified |
| Windows and Linux amd64 builds | Built with `-trimpath`; both executables ran successfully. The Linux release was also tested/built with `CGO_ENABLED=0` and confirmed statically linked |
| Windows CLI demonstration | version, host read-only doctor, fixture doctor, search, info, fetch, validate, list, install plan/apply, HD switch plan/apply, status, history, later rollback all succeeded |
| Linux CLI demonstration | Executables ran version and read-only Debian/Fedora fixture doctor; the complete CLI install/switch integration test also passed on Linux |
| Refusal behavior | Preview without dependencies and browse-only Minegrub fetch returned documented exit code 4 |

The named-case counts include subtests, not a claim of line coverage. Test packages without test files are not counted as failures or skipped test cases. GitHub Actions is configured; no remote Actions run is claimed. Race detection is configured in Linux CI but was not run during this local pass.

Tests cover each journal phase and the commit boundary, active asset retention, cleanup of new assets after failure, interruption/recovery, restoration failure, drift refusal, stale/root-mismatched plans, current-kernel preservation during later rollback, immutable revision selection, and symlink/hardlink containment. Archive tests cover path traversal, absolute and drive paths, duplicate/case/file-directory conflicts, links/special files and resource limits. These are bounded synthetic defensive fixtures, not third-party exploit reproductions.

No installed Go toolchain was found on Windows or in WSL. Official **Go 1.27.1** portable archives were downloaded into the workspace and verified against the official release SHA-256 values. No host-wide toolchain installation was made. The default Windows sandbox temporary path was too long/restricted for directory promotion; the documented short workspace temporary path was used for the passing Windows run. The Linux run used ordinary temporary fixture roots.

## Deliberately deferred

Real activation, privileged helper/Polkit, actual GRUB generation/checking, QEMU preview execution, update discovery, registry refresh, pin/unpin commands, garbage collection, package export, GUI, native packages and additional activation backends remain disabled or unimplemented. The fixture renderer is explicitly synthetic. Static validation does not certify physical bootability, font rendering or complete GRUB grammar support. Windows journaled writes do not claim atomic replacement or power-loss safety.

The next implementation step is a **separately reviewed, root-owned Debian helper and Polkit contract**, followed by disposable Debian 12 BIOS/UEFI VM tests for real candidate generation, metadata preservation, boot readability, failure recovery and reboot verification. Follow the exact gate in [testing.md](testing.md); do not remove the fixture guard to experiment on a real installation.

## Source reuse and notices

No audited GRUB-manager source was copied, including the potential MIT PF2/lint candidate. The PF2 inspector and domain logic are original. Commodity functionality uses Go standard libraries and the unmodified pinned `modernc.org/sqlite` stack under the preserved permissive/public-domain notices. See [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) and [the dependency ledger](../third_party/dependencies.json) for exact versions, sources and notice hashes. No GPL preview code or community theme artwork was bundled.

**No real bootloader configuration was changed and no GRUB command or community theme installer was executed.** All activation demonstrations occurred in synthetic roots.
