# Debian package

The private beta package installs the CLI at `/usr/bin/grubmgr`, the two helpers in `/usr/libexec`, the Polkit action and documentation. It creates `/var/lib/grubmgr` with mode 0700. It contains no maintainer scripts, boot settings, VM activation marker, authorization bypass or theme assets.

## Build

Use an ordinary Linux account with Go 1.27.1, Python 3.11+, Git and dpkg-deb:

```sh
python3 scripts/build-deb.py
python3 scripts/check-deb.py outputs/debian/*.deb
```

`--go /path/to/go` selects the compiler; `--output DIRECTORY` selects the output directory. Builds target Linux amd64 with CGO disabled. The builder uses a temporary staging directory and [dpkg-deb's root ownership option](https://manpages.debian.org/trixie/dpkg/dpkg-deb.1.en.html); it does not require sudo. Adjacent SHA-256 and manifest files identify the archive and installed files. These hashes identify bytes; they are not a publisher signature.

The current Debian version is `0.2.0~rc.1-1`; the CLI reports `0.2.0-rc.1`. The tilde sorts the candidate before the final release. The builder uses the source commit timestamp unless `SOURCE_DATE_EPOCH` is supplied.

## Installation and removal

Installation is currently for disposable test VMs:

```sh
sudo apt install ./grubmgr_0.2.0~rc.1-1_amd64.deb
grubmgr doctor
```

Installing the package does not enable real activation. The VM harness separately provisions the deployment marker after checking the VM identity. Do not copy that test setup onto a workstation.

The package depends on pkexec and polkitd. GRUB/preview packages are optional suggestions so installing grubmgr does not pull in a bootloader. The optional external preview renderer remains separately installed; see the VM harness for the tested version. The shipped policy requires administrator authentication in an active local session and denies remote/inactive sessions. [pkexec provides a text authentication agent](https://polkit.pages.freedesktop.org/polkit/pkexec.1.html) when no session agent is registered.

`sudo apt remove grubmgr` removes the application files. Existing theme assets and nonempty state are retained, including on purge: removal must not invalidate the theme path currently used by GRUB. Reinstalling restores the application without discarding those receipts. Removing theme assets is a separate, currently deferred cleanup operation.

## Verification

The completed fresh-VM run and exact tested archive are recorded in [package verification](debian-package-verification.md).

Normal Linux CI builds and inspects the archive without installing it, then publishes the `.deb`, checksum and manifest as a workflow artifact. `scripts/check-deb.py` checks file hashes, root ownership, modes, executable architecture, the strict Polkit policy and absence of maintainer scripts or boot settings.

The separate VM harness accepts `--deb PATH` and requires the adjacent `.manifest.json`. Without that argument it builds one package before starting the VM. It freezes the package, test binary and guest scripts, verifies installed hashes, exercises the production password prompt, then runs the boot/failure/preview workflow. Reinstall, remove, purge and reinstall must preserve boot configuration, retained assets and receipts. Frozen input hashes are rechecked before `PASS`.

The password checks use a real local virtual-console login on seat0 and Polkit's text agent: cancellation, a wrong password and a correct password. The test password and autologin setup are removed afterward. The long transaction suite then uses the existing scoped guest authorization rule. A graphical desktop authentication agent is not covered by this test.
