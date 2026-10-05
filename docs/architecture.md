# Architecture

The Go CLI shares ingestion, validation and storage code with two transaction engines. `internal/debian` implements Linux activation for all four distributions; `internal/backend` and `internal/transaction` implement synthetic fixtures.

## Package storage

Imports are staged, validated and moved into an immutable revision directory on the same filesystem. A revision hashes the recipe and file inventory, including license notices. SQLite stores receipts and transaction journals.

| Store | Location |
| --- | --- |
| User imports | XDG data directory |
| User receipts | XDG state directory |
| Preview images and logs | XDG cache directory |
| Helper imports, receipts and journals | `/var/lib/grubmgr/{data,state,cache}` |
| Installed assets | `/boot/grub/themes/grubmgr/NAMESPACE/NAME/REVISION` |

The [package format](package-format.md) defines identity, extraction limits and validation rules.

## Privileged helper

The CLI sends bounded JSON through `/usr/bin/pkexec /usr/libexec/grubmgr-helper`. The protocol provides `inspect`, `status`, `history`, `plan`, `apply` and `recover`. Executable paths and destinations come from the selected distribution profile.

Polkit requires administrator authentication in an active local session. Terminal callers register `pkttyagent` using their process ID and kernel start time; passwords pass directly through the terminal.

The helper receives package bytes and a manifest, checks the recipe against its compiled catalog, and rebuilds the file inventory. It verifies digests, ownership and configuration fingerprints before writing to its root-owned store. User receipts cannot approve a package.

GRUB tools run at fixed paths with a cleared environment, bounded output and a timeout. Existing root-owned GRUB scripts are administrative configuration used by the generator.

## Plans and transactions

A plan token binds the action, target, variant, package revision and system fingerprint. Apply acquires the manager and distribution package locks, rebuilds the plan, and compares tokens.

```text
planned → prepared → assets_staged → settings_staged
        → candidate_validated → activated → committed
failure → recovering → rolled_back | recovery_required
```

Activation changes `GRUB_THEME` in `/etc/default/grub`, preserving unrelated bytes. The profile's generator writes `/boot/grub/.grubmgr-TRANSACTION.cfg`. The helper runs `grub-script-check`, checks the selected theme and Linux entries, then replaces `/boot/grub/grub.cfg` with a synced rename that preserves ownership and permissions. Files with extended attributes are rejected.

The journal records original bytes and expected hashes. Recovery restores snapshots when the current files match either the original or expected state. A conflicting edit leaves the transaction in `recovery_required`. Later rollback generates a fresh configuration with the current kernels and a retained theme revision.

SQLite uses full synchronous commits. Configuration replacement, asset staging and receipt commits are coordinated by the journal. Theme assets remain after failure and removal.

## Locks and fixtures

The helper takes `flock` for its own operations and POSIX locks for dpkg. Pacman uses an exclusive `db.lck` hard-linked to a durable ownership file; recovery reclaims it only when inode identity and contents match. Configuration fingerprints also detect edits made outside these locks.

Fixtures use synthetic configuration generation and user-owned stores. Linux replacement uses rename; Windows fixture writes use in-place replacement backed by the recovery journal. [Fixture recovery](testing.md#fixture-recovery) describes stale lock handling.

## Preview

`grub2-theme-preview` 2.10.0 constructs a boot image from a generated menu. Bubblewrap exposes read-only system tools and the selected theme, a private temporary filesystem, and an output directory. The QEMU adapter uses software emulation, read-only generated media, disabled networking and a bounded runtime. It waits for the menu, captures a frame through QMP and writes a PNG.

Ubuntu's optional AppArmor profile grants the application user namespaces; Bubblewrap supplies the process and filesystem isolation. See [preview setup](installation.md#preview-setup).
