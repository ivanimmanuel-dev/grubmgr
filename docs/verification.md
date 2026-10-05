# GRUB Manager 0.1.0 verification

Historical fixture-only verification, completed 2026-10-05. For current capabilities and VM results, see [architecture](architecture.md) and [Linux verification](linux-verification.md).

## Implementation

The shared Go core and CLI implement catalog browsing, local and reviewed HTTPS imports, immutable manifests and SHA-256 identities, bounded ZIP/TAR extraction, theme/image/PF2 validation, rooted system detection, SQLite receipts, read-only plan tokens, and fixture transactions with failure recovery. JSON output, optional structured logs, XDG paths and stable exit codes are included.

`install`, `switch`, `remove` and later theme `rollback` work against marked synthetic Debian roots. Install does not activate. Removal retains assets for rollback. The source tree includes five distro fixtures, tests, documentation, license notices and Linux/Windows GitHub Actions.

## Results

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

Counts include named subtests. These were local runs; remote CI and race detection were not part of this verification.

Tests cover each journal phase and the commit boundary, active asset retention, cleanup of new assets after failure, interruption/recovery, restoration failure, drift refusal, stale/root-mismatched plans, current-kernel preservation during later rollback, immutable revision selection, and symlink/hardlink containment. Archive tests cover path traversal, absolute and drive paths, duplicate/case/file-directory conflicts, links/special files and resource limits. The cases used generated fixtures.

Tests used portable Go 1.27.1 toolchains verified against official release checksums. Windows used a short workspace temporary path to avoid directory-promotion restrictions; Linux used its native temporary filesystem.

## Version scope

Real activation, privileged helper/Polkit, actual GRUB generation/checking, QEMU preview execution, update discovery, registry refresh, pin/unpin commands, garbage collection, package export, GUI, native packages and additional activation backends were unimplemented in 0.1.0. Rendering used synthetic fixtures, and static validation covered a grammar subset. Windows fixture writes used a recovery journal with in-place replacement. Physical boot and power-loss behavior were outside this verification.

The subsequent Debian backend and Polkit work is recorded in [the design review](phase2-review.md) and [0.2.0 verification](phase2-verification.md).

## Source reuse and notices

The PF2 inspector and domain logic are original implementations. Storage and parsing use Go standard libraries and the unmodified pinned `modernc.org/sqlite` stack under the preserved permissive/public-domain notices. See [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) and [the dependency ledger](../third_party/dependencies.json) for exact versions, sources and notice hashes. Preview and community-theme integration were outside this version.

All activation demonstrations used synthetic roots.
