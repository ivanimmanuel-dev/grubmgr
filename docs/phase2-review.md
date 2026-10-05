# Phase 2 implementation review

Baseline: 2026-10-05, commit `149853d`, with the identical tree uploaded as `13dc7b8`.

- Windows: 83 named test cases passed. The sandbox blocks the loopback HTTPS test; the unrestricted run passed.
- Linux: 86 named test cases passed with `TMPDIR=/tmp`. A first run on Windows-backed temporary storage failed the case-collision test.
- Transactions already journal each phase before proceeding. Immediate recovery checks before/expected hashes; later rollback selects retained assets around current boot state.
- The planner currently reads a user-owned SQLite database and generates a synthetic candidate. Those assumptions cannot cross a root boundary.
- Fixture writes lack the ownership, permission, directory sync and native package-manager coordination needed for real activation.
- `doctor` detects layouts but does not establish which bootloader firmware actually used. The first real backend will require a disposable-VM marker as well as the Debian layout checks.
- Normal CI remains unit tests, fixture tests, formatting, vet, module/license checks, builds and Linux race detection. VM tests are separate.

The implementation will retain the transaction phases and receipt model. Backend-specific operations will supply locking, candidate generation, validation and durable replacement. The helper will read only fixed typed requests and bounded package bytes; it will never accept executable or destination paths.

Initial test target: Debian 13 amd64, QEMU Q35, UEFI/OVMF without Secure Boot, ext4 root and a FAT EFI system partition. Exact installed package versions and boot evidence are recorded after VM execution. Other systems remain refused.

Candidate generation uses Debian's documented `grub-mkconfig -o FILE`; no custom defaults flag exists in that interface. The helper must journal and stage the theme assignment first, generate to a separate candidate, run `grub-script-check`, then replace the generated configuration. See the [Debian manual](https://manpages.debian.org/trixie/grub-common/grub-mkconfig.8.en.html).
