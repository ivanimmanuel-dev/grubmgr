# Linux VM verification

Historical test record, 2026-10-05, for the `0.3.0-rc.1` archives identified below. Arch, Kali and Ubuntu completed the listed workflows in disposable QEMU guests. Current support is documented in [Linux support](linux-support.md).

## Tested profiles

| System | GRUB package | Layout |
| --- | --- | --- |
| Arch rolling | `2:2.16-1` | Flat Btrfs root, subvolume ID 5; `/boot` on root; FAT `/efi` |
| Kali rolling | `2.14-2+kali1` | ext4 root including `/boot`; FAT `/boot/efi` |
| Ubuntu 24.04.5 | `2.12-1ubuntu7.3` | ext4 root; separate ext4 `/boot`; FAT `/boot/efi` |

All guests were x86-64, UEFI, Secure Boot disabled, with private virtual disks and firmware variables. The official image URLs and SHA-256 values are recorded in [the evidence summary](linux-verification.json). Kali's artifact filename says 2026.2; its `/etc/os-release` reports 2026.1 and `kali-rolling`. The guest used `kali-last-snapshot` package repositories.

## Results

| System | Executed run | Recovery tests, including named subtests | Result |
| --- | --- | --- | --- |
| Arch | `arch-matrix-wvdv2mad` | 30 passed; 0 skipped | Completed |
| Kali | `kali-matrix-rx04o718` | 29 passed; 1 skipped | Completed |
| Ubuntu | `ubuntu-upgrade-6gwd750r` | 29 passed in `ubuntu-matrix-p_rrgo_j`; 0 skipped | Completed |

Arch and Kali each completed a fresh matrix run with frozen installer, test binary and scripts. Ubuntu used two runs: `ubuntu-matrix-p_rrgo_j` completed activation, two reboots, kernel-preserving rollback and the recovery suite, then failed preview under Ubuntu's namespace restrictions. The final continuation cloned that powered-off guest, installed the final package, repeated the password and package-lifecycle checks, enabled the packaged application-specific AppArmor profile, rendered the preview, and repeated activation/rollback with two further successful reboots. Both kernels already existed in the continuation. The JSON records the baseline recovery suite separately from the final-archive checks.

The single skip on Kali is the Arch-only EFI automount test. Ubuntu's earlier test binary did not contain that new test. Arch passed it after the existing EFI automount was made idle.

Each target passed real local-console Polkit tests for cancellation, an incorrect password and successful administrator authentication. Boot files remained unchanged in those inspection requests. The temporary test password and autologin were removed before the transaction tests used their separately scoped guest-only authorization rule.

The workflow checked import, validation, install without activation, theme selection, HD-variant switching, later rollback, retained kernel entries, successful boot, and package removal/reinstallation without changing boot files or retained state. Recovery cases included package locks, process termination, generator failure, phase failures and interruptions, failed rollback, stale plans, modified defaults, unsafe ownership/permissions and ENOSPC during file replacement.

## Reboots and retained kernels

| System | Before theme reboot | After theme reboot | After rollback reboot |
| --- | --- | --- | --- |
| Arch | `fe7df140-9e7e-40e8-b450-23b90773a4ef` | `b78101e7-d60c-4ffa-a577-3d7196159ecb` | `1f656d10-aa6f-4b54-98d1-87ad6cc67122` |
| Kali | `b27c11e1-3f22-46a3-a311-9333cfeaf71a` | `4f598650-622d-4be7-a95d-df0813d56ecb` | `7fce0c6f-438d-4127-87ad-ae82733be2f1` |
| Ubuntu | `e4bacfa6-89fc-475f-bb7e-075115e6cf5b` | `db6264e5-5e09-4c90-b837-44aa813c42ed` | `d6ba95f1-1b7d-4a64-b5ab-b37f9b44ecb9` |

- Arch: `vmlinuz-linux`, `vmlinuz-linux-lts`.
- Kali: `vmlinuz-6.18.12+kali-cloud-amd64`, `vmlinuz-7.1.5+kali-amd64`, `vmlinuz-7.1.5+kali-cloud-amd64`.
- Ubuntu: `vmlinuz-6.8.0-142-generic`, `vmlinuz-6.8.0-146-lowlatency`.

Arch booted its updated regular kernel and later its LTS kernel. Kali booted the newer cloud kernel after rollback; the generic kernel was retained but not separately boot-tested. Ubuntu's earlier run booted the added lowlatency kernel; the final continuation used it for both reboots.

## Preview and packaging

All three previews ran real GRUB under QEMU through the optional external `grub2-theme-preview` 2.10.0 renderer. The 1024×768 PNGs were visually inspected. These use a generated menu and the original Cyberpunk Demo theme, with no host boot disk, home directory or network exposed to the preview.

Ubuntu retained `kernel.apparmor_restrict_unprivileged_userns=1`. An unrelated direct Bubblewrap invocation still failed after loading the profile attached to `/usr/bin/grubmgr`. The profile grants namespace access to that application and its children, with Bubblewrap providing isolation. See [setup and scope](linux-support.md#themes-and-preview).

| VM-tested archive | SHA-256 |
| --- | --- |
| Arch (43 installed files verified) | `5388630f55b7c3eac3a17c72da0b5f2363572e5b33e7bac1a67fee95d6a427e0` |
| Kali (43 installed files verified) | `7edca7f9c6217f6c800ee12dffc17a236612b445fe7f448c6034945b1fe7c0a3` |
| Ubuntu (43 installed files verified) | `7edca7f9c6217f6c800ee12dffc17a236612b445fe7f448c6034945b1fe7c0a3` |

The three installed executable hashes match across these final runs. The Arch archive predates the addition of the optional Ubuntu AppArmor profile and documentation changes; that profile is not used on Arch. Ubuntu's earlier recovery run used the archive recorded by `failure_suite_package_sha256` in the JSON. CI rebuilds packages from the committed source; its archive hashes can differ from these retained VM-tested archives.

## Ordinary checks and reproduction

The Linux race suite passed 96 named cases including subtests. Windows tests and vet passed. Linux vet, module verification, license verification, formatting, package inspection and Python compilation passed. The Linux backend race tests also passed after the EFI automount change. Normal CI runs fixtures and builds both `.deb` and pacman archives; it does not run real boot tests.

The executed fresh-run commands used `scripts/vm/run-matrix.py arch` and `scripts/vm/run-matrix.py kali` with the frozen package, Go 1.27.1 compiler and extracted QEMU tools. The equivalent Ubuntu fresh-run command now includes its optional preview-policy setup. See [Linux support](linux-support.md#build-and-test) and the frozen input hashes retained with the local evidence.

Raw logs, installer manifests and screenshots are retained under `outputs/linux-verification/`. VM disks and downloaded images were removed after exporting evidence. Rolling-image URLs may disappear; a changed image must pass checksum review and new VM verification.

## Limits

Support is limited to the exact package versions and layouts above, with the root-owned test marker and matching QEMU identity. Other releases/layouts, BIOS, Secure Boot enabled, snapshot integration and GRUB Customizer remain refused. Only the demo recipe is approved on the new targets; the existing Debian Starfield verification remains separate.

Physical boot, arbitrary power loss and separate boot tests for every retained kernel were outside this run. ENOSPC coverage exercised file replacement on a bounded tmpfs; it did not fill the entire boot filesystem.
