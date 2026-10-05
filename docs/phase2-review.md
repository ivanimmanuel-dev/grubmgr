# Debian backend design review

Historical review of the fixture-only implementation on 2026-10-05: local commit `149853d`, identical repository tree at `13dc7b8`. The resulting backend is documented in [architecture](architecture.md) and [Debian verification](phase2-verification.md).

- Windows: 83 named test cases passed. The sandbox blocks the loopback HTTPS test; the unrestricted run passed.
- Linux: 86 named test cases passed with `TMPDIR=/tmp`. A first run on Windows-backed temporary storage failed the case-collision test.
- Transactions journaled each phase. Recovery checked original and expected hashes; later rollback selected retained assets against current boot state.
- The planner read a user-owned SQLite database and generated a synthetic candidate. The real backend needed independent root-owned state and distro generation.
- Fixture writes lacked the ownership, permission, directory sync and package-manager coordination required for real activation.
- `doctor` detected layouts but could not identify the active firmware bootloader. The proposed backend required a disposable-VM marker and Debian layout checks.
- CI covered fixtures, formatting, vet, dependency notices, builds and Linux race detection. VM tests were designed as a separate suite.

The design retained the transaction phases and receipt model. Backend-specific operations would supply locking, candidate generation, validation and durable replacement. The helper contract used fixed typed requests and bounded package bytes, with executable and destination paths derived internally.

Initial test target: Debian 13 amd64, QEMU Q35, UEFI/OVMF without Secure Boot, ext4 root and a FAT EFI system partition. The [verification record](phase2-verification.md) identifies the installed packages and boot results.

The design selected Debian's documented `grub-mkconfig -o FILE`, which reads the installed defaults. The helper would journal and stage the theme assignment, generate a separate candidate, run `grub-script-check`, then replace the generated configuration. See the [Debian manual](https://manpages.debian.org/trixie/grub-common/grub-mkconfig.8.en.html).
