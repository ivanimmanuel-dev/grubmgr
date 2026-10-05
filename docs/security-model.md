# Security model

Themes are data. The CLI does not run community installers, source theme files as shell, or elevate itself. Real activation requires an explicitly supported disposable VM profile. Detection on another system does not enable writes.

## User inputs

Archives and local imports reject links, special files, traversal, case collisions and oversized members. Image/font decoding and theme parsing have bounds. HTTPS imports need an exact URL and SHA-256 recipe. A digest proves content identity, not authorship or permission. Unknown-license entries remain unavailable for activation.

`--root` inspects only its marked synthetic fixture and never falls back to host `/boot`, `/etc` or firmware. It must not point to a real mounted installation. Fixture stores are not protected from another process running as their owner.

## Privileged boundary

The CLI invokes only `/usr/bin/pkexec /usr/libexec/grubmgr-helper`. The installed helper accepts a bounded JSON request with fixed operations: inspect, status, history, plan, apply and recover. There are no executable, shell, destination, root-override or failure-injection fields. `PKEXEC_UID` must identify an ordinary user; authorization is enforced by the installed Polkit action and protected executable.

The helper independently checks package identity, every byte, the compiled recipe/content pin, revision, destinations, root ownership, links and current configuration fingerprints. Caller-supplied reviewed/license flags cannot approve a package. It writes its own protected cache, receipts and recovery snapshots. The ordinary-user database is not authoritative.

The production Polkit policy requires administrator authentication for an active session. The VM harness installs a separate tester-only authorization rule **inside the disposable guest**. Do not install that rule on a workstation.

Terminal callers register the installed `pkttyagent` as their ordinary user and wait for registration before invoking `pkexec`. The subject includes the caller's PID and kernel start time. The agent uses the controlling terminal; passwords never enter grubmgr's request, output or logs. The agent stops when the request ends. No authentication rule is changed by the application.

Only fixed profile-selected generator/checker paths run, with a cleared environment, bounded output and timeout. Root-owned distro configuration scripts execute as part of the installed generator; these are trusted administrative inputs, not theme package code. Dpkg or pacman locks coordinate package updates. Only `GRUB_THEME` changes; the helper does not reinstall GRUB, alter kernel arguments/default OS/timeout, or write partitions, EFI variables or Secure Boot policy.

Immediate recovery checks original/expected hashes and verifies restored bytes. Conflicts retain the root journal and report `recovery_required`. Later rollback regenerates with the current boot inventory. Journals may contain configuration text and are not public artifacts.

## Preview

Preview runs as an ordinary Linux user through Bubblewrap. It has no real `/boot`, host home, host disks or network. The renderer gets a generated menu; a fixed adapter constrains QEMU drives, memory and time. Missing tools or unavailable namespaces produce a diagnostic rather than falling back to an unsandboxed launch. Read-only `/usr` exposes installed system programs and public assets. A QEMU image cannot certify a physical boot or make malicious renderer/firmware bugs impossible.

Ubuntu's optional packaged AppArmor profile allows user namespaces for the installed `/usr/bin/grubmgr` and its children. It follows Ubuntu's application permission mechanism and does not replace Bubblewrap isolation or provide full AppArmor confinement. Loading it is an explicit administrator dependency-setup step; package installation does not reload policy or change system-wide namespace restrictions.

## Limits

The supported backend requires a root-owned VM marker and matching QEMU SMBIOS value as well as exact distro/version/layout checks. Those are accidental-use guards, not a defense against an administrator deliberately bypassing them. BIOS, Secure Boot enabled, immutable systems, customizer/snapshot integrations and untested GRUB versions or layouts are refused. Ubuntu's separate ext4 `/boot` and Arch's flat Btrfs root have distinct profiles; other layouts are not inferred from them. See [Linux targets](linux-support.md).

SIGKILL recovery is tested at selected journal boundaries; arbitrary power loss at every syscall is not certified. Another root process can bypass advisory locks. No automatic repair after an unbootable physical system is promised. A privileged-helper review remains a gate before expanding beyond test VMs.

MIT covers original code and synthetic assets. Imported theme artwork/fonts retain their own licenses and notices. The GPL preview renderer remains a separately installed program. Grub of Tsushima lacks confirmed permissions for its third-party artwork/fonts and has no approved activation recipe.
