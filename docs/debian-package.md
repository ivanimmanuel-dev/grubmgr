# Debian package

The `.deb` supports the Debian, Ubuntu and Kali VM profiles. It installs the CLI, two helpers, Polkit action, optional AppArmor profile and documentation. `/var/lib/grubmgr` is created with mode `0700`. Installation leaves boot settings unchanged; the VM harness provisions activation separately.

## Build

Use an ordinary Linux account with Go 1.27.1, Python 3.11+, Git and dpkg-deb:

```sh
python3 scripts/build-deb.py
python3 scripts/check-deb.py outputs/debian/*.deb
```

`--go PATH` selects the compiler; `--output DIRECTORY` selects the output directory. Builds target Linux amd64 with CGO disabled. Staging uses [dpkg-deb's root ownership option](https://manpages.debian.org/trixie/dpkg/dpkg-deb.1.en.html), so building requires no elevation.

The package version is `0.3.0~rc.1-1`; the CLI reports `0.3.0-rc.1`. The tilde sorts the candidate before a final release. Builds use the source commit timestamp unless `SOURCE_DATE_EPOCH` is supplied. Adjacent checksum and manifest files identify the archive and installed files; they are unsigned.

## Install

Inside a supported disposable VM:

```sh
sudo apt install ./grubmgr_0.3.0~rc.1-1_amd64.deb
grubmgr doctor
```

The package depends on pkexec and polkitd. GRUB and preview tools are optional suggestions, installed separately for the test environment. The [Linux support guide](linux-support.md) covers the renderer, firmware and Ubuntu's optional AppArmor setup.

The shipped policy requires administrator authentication in an active local session. Terminal commands use `pkttyagent`. The package contains no maintainer scripts, activation marker, test authorization rule or community theme assets.

## Remove

```sh
sudo apt remove grubmgr
```

Removal and purge retain theme assets and nonempty application state so GRUB can still read its selected theme. Reinstallation restores the application and uses those receipts. Asset garbage collection is planned separately.

## Verification

Linux CI builds and inspects the package, then publishes the `.deb`, checksum and manifest as the `grubmgr-debian-amd64` artifact. Inspection checks file hashes, ownership, permissions, executable architecture, Polkit policy and package contents.

The [VM harness](vm-testing.md) adds installed-file verification, real password authentication, activation/reboots, rollback, failure recovery, preview and package lifecycle tests. See the [Debian 0.2.0 package record](debian-package-verification.md) and [0.3.0 Linux verification](linux-verification.md) for the exact tested archives.
