# Security model

GRUB Manager treats themes as data. Import and preview run under the user's account. Activation uses a fixed Polkit helper and is restricted to the [supported disposable VM profiles](linux-support.md).

## Package inputs

Archives and local imports reject links, special files, traversal, case collisions and oversized members. Theme parsing and image/font decoding enforce resource limits. HTTPS imports require an exact recipe URL and SHA-256 digest. The digest identifies content; author and license evidence are separate recipe fields.

The fixture engine uses a marked synthetic root supplied through `--root`. Inspection and destination paths resolve beneath that root. Fixture stores are owned by the caller and can be changed by other processes under the same account. A real mounted installation is not a fixture.

## Privileged operations

The CLI calls `/usr/bin/pkexec /usr/libexec/grubmgr-helper`. The helper accepts bounded JSON for six operations: `inspect`, `status`, `history`, `plan`, `apply` and `recover`. It derives executable and destination paths from the selected profile. The protocol has no shell, root override or failure-injection fields.

The installed Polkit action requires administrator authentication from an active local session. `PKEXEC_UID` identifies the ordinary user. Terminal callers register `pkttyagent` with their PID and kernel start time, wait for registration, and stop the agent after the request. Passwords pass directly through the terminal.

The helper verifies the compiled recipe and content pin, manifest revision, file hashes, destination ownership, link policy and current configuration fingerprints. It keeps an independent root-owned cache and journal; the user database and caller-supplied review flags cannot approve a package.

Only the profile's generator and checker run, using fixed paths, a cleared environment, bounded output and a timeout. Installed root-owned GRUB scripts are trusted administrative configuration. Dpkg or pacman locks coordinate package updates. Activation changes `GRUB_THEME`; bootloader installation, kernel options, default OS, timeout and firmware policy remain outside the helper interface.

Immediate recovery checks original and expected hashes before restoring snapshots, then verifies the restored bytes. Conflicts retain the journal with `recovery_required`. Later rollback regenerates configuration with the current boot inventory. Journals contain configuration text and should be kept private.

## Preview isolation

Preview uses Bubblewrap with private mount, PID and network namespaces. It exposes read-only `/usr`, the selected theme and an output directory. The renderer receives a generated menu; the QEMU adapter restricts drives, memory and execution time. Host boot disks, the home directory and network access are excluded. Missing tools or unavailable namespaces stop preview.

Ubuntu's optional AppArmor profile permits user namespaces for `/usr/bin/grubmgr` and its children. It is an application permission profile, with Bubblewrap providing isolation. An administrator loads it during preview setup; package installation leaves system policy unchanged.

## Deployment and test limits

Activation requires exact distro, package-version and layout checks, a root-owned VM marker, and matching QEMU SMBIOS identity. The marker prevents accidental use outside the test environment; an existing root administrator controls that environment and can bypass advisory locks.

The VM suite exercises selected process interruptions and recovery boundaries. Physical hardware, arbitrary power loss, BIOS, Secure Boot enabled, snapshot integrations, GRUB Customizer and other untested layouts are unsupported. See [verification](linux-verification.md) for the executed cases. Expanding to physical systems requires a privileged-helper review and further boot testing.

The VM harness uses a separate tester-only authorization rule after testing the shipped password policy. That rule belongs exclusively to the disposable guest.

## Licenses

Original code and demo assets use MIT. Imported artwork and fonts retain their notices. The GPL preview renderer is installed separately. The [third-party notices](../THIRD_PARTY_NOTICES.md) record dependencies and theme provenance.
