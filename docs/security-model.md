# Security model and explicit limits

The implemented boundary is unprivileged package handling plus a synthetic-root backend. **There is no real activation path, Polkit policy, shell execution, GRUB invocation or QEMU invocation in the product.** A theme is data. No upstream installer was used to build/test this project.

## Inputs and containment

Catalogue entries are schema data. Unknown entries are browse-only; reviewed remote artifacts additionally require an exact recipe URL and SHA-256. HTTPS and hashes provide transport/content integrity; a reviewed recipe is the trust anchor. A hash alone does not prove author identity, license ownership or safe rendering. Remote catalogue signing and authentication frameworks are deferred.

Archive extraction uses Go decoders into ordinary-user staging. Portable, case-folded path checks are applied before writes. All links and special files are rejected, with member/byte/image limits. Local imports get the same inventory/path checks. Theme syntax is parsed as a bounded subset and never sourced as shell.

OS inspection uses `os.Root` relative handles. With `--root`, no fallback reads the host's `/etc`, `/boot`, `/proc`, firmware paths or utility search path. A missing fixture value is unknown. Application requires an explicit non-volume root with exact `.grubmgr-fixture` marker and conventional Debian evidence. Parent symlinks and multiply linked mutation targets are refused. Rooted operations prevent link traversal outside the fixture; Windows truncation checks the opened handle's link count first.

Explicitly supplied local input paths and recipe files may be outside the fixture: these are requested data inputs, not OS inspection or mutation destinations. Cache/state paths beneath the fixture are checked for links before opening. SQLite uses an ordinary pathname API, so fixture/store directories must be owned and controlled by the calling user. The system does not promise protection against a hostile process running as that same user concurrently replacing the store/root namespace. A future privileged helper must provide an independently enforced boundary against user-controlled state and race conditions.

`--root` must point to disposable synthetic data. It is not permission to operate on a mounted real system, and the marker is not a security credential. Never add the marker to a real system tree. The build does not install bootloader binaries or modify EFI/NVRAM, BLS, Secure Boot policy, kernel arguments, boot entry order or native package repositories.

## Plan and transaction integrity

Planning opens receipts read-only, does not download, and does not create plan files. Plan tokens contain a typed request and a state fingerprint. The fixture backend recomputes destinations and content checks under an exclusive operation lock. Tokens cannot supply programs, raw shell, arbitrary destination paths or replacement bytes. Stale/root-mismatched plans are refused.

The phase journal stores snapshots before mutation, verifies staged assets and candidate bytes, and records receipt changes separately. Incomplete/failed journals block new plans. Immediate rollback restores only a state matching the before/expected hashes; external drift requires recovery review. Old active assets are retained. Later user rollback generates the fixture theme block around the current configuration instead of writing an obsolete whole-file boot snapshot.

The fixture lock serializes grubmgr apply/recover, and a state operation lock serializes receipt publication with fetch. Neither locks OS package managers, protects against malicious same-user writers or certifies abrupt power-loss behavior. Windows writes are journaled but not atomic. Restoration errors are reported as `recovery_required`; success is never inferred merely from attempting a restore. See architecture/testing for stale-lock handling and limits of interruption tests.

## Preview and privilege

`grub2-theme-preview` remains a separately installed optional dependency. `preview` currently reports missing dependencies or explicitly refuses execution when they are present, because the sandbox adapter is not yet validated. No catalogue arguments reach QEMU, no host disks are attached, and no renderer runs as root.

The next real backend needs a root-owned fixed helper, narrow Polkit actions, independent package/path validation, metadata/label preservation, bounded subprocesses with fixed environments, native-update coordination, candidate GRUB checks, recovery journals outside user control, and disposable-VM boot tests. Do not turn the fixture generator into a production backend by removing its gate.

## License boundary

MIT applies to original grubmgr code and synthetic assets. Dependency notices are retained under `third_party/licenses`. The package importer preserves upstream notices but does not manufacture missing grants. No Gorgeous-GRUB assets/descriptions, community theme artwork, GPL renderer source, or manager installer code is bundled. The source-reuse ledger documents actual imports only.
