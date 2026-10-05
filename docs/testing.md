# Testing

## Automated checks

Run from the repository root with Go 1.27.1:

```sh
go test ./...
go vet ./...
go mod verify
go run ./tools/checklicenses
test -z "$(gofmt -l cmd internal tools)"
go build ./cmd/...
```

CI runs Windows and Linux tests, Linux race detection, package builds and archive inspection. Ordinary tests use temporary synthetic roots and an in-process HTTPS server. GRUB generation, QEMU and the privileged helper are exercised by the separate VM suite.

On Windows, a short temporary path avoids path-length restrictions:

```powershell
New-Item -ItemType Directory -Force work/test-tmp | Out-Null
$env:TEMP = Join-Path $PWD work/test-tmp
$env:TMP = $env:TEMP
go test ./...
```

HTTPS tests require local sockets. Case-collision directory tests skip on case-insensitive Windows filesystems; equivalent ZIP checks still run. Symlink tests skip when the account cannot create links. Linux CI covers both cases.

## Coverage

| Area | Cases |
| --- | --- |
| CLI | Import, validation, install/switch plans and application, status/history, JSON and exit codes |
| Detection | Debian, Arch, Fedora EFI stub, non-GRUB and ambiguous fixtures; rooted inspection |
| Ingestion | Traversal, absolute paths, duplicate/case conflicts, links, special files, resource limits, HTTPS digests and redirect policy |
| Identity | Repeatable imports, tampering, ambiguous roots, exact revision selection and notice retention |
| Validation | Nested images, palette PNG, missing/case-mismatched assets, broken images, scripts, PF2 names/framing and variants |
| Planning | Read-only snapshots, fingerprints, stale state, unknown provenance and unsupported backends |
| Transactions | Phase and commit failures, install without activation, switching, removal, rollback and asset retention |
| Recovery | Interrupted journals, restore failures, external edits and pending-transaction refusal |
| Boot inventory | Later rollback after adding a kernel entry |
| Containment | Symlink escapes and hardlink targets |
| Licenses | Dependency versions, preserved notice hashes and module integrity |

`fixtures/` contains synthetic system text. Tests generate archive, image and PF2 samples. `internal/catalog/demo` contains the original multi-variant theme. Copy a fixture into `work/` for manual testing; keep real mounted installations outside fixture workflows.

## Recovery demo

Failure injection uses the Go test dependency `transaction.Faults`. Focused recovery tests:

```sh
go test -v ./internal/transaction -run 'TestFailuresEveryPhase|TestSwitchFailuresRetainActiveAssets|TestInterruptedRecovery|TestRecoveryFailureAndDrift'
```

After killing a fixture process, confirm it has stopped and preserve the fixture root and SQLite files. Remove its stale `.grubmgr/apply.lock` and `.grubmgr/state/operation.lock`, if present, then run `grubmgr --root ROOT recover`. A hash conflict requires inspection of the snapshots and competing files before retrying. Recovery output can include configuration text.

## VM tests

The [Debian harness](vm-testing.md) and [Linux matrix](linux-support.md#build-and-test) exercise authorization, package installation, activation, reboots, kernel-preserving rollback, failure recovery, preview and removal/reinstallation. Tests compiled with `grubmgr_vmtest` run only inside the disposable guests.

For recorded results, see [Debian package verification](debian-package-verification.md) and [Linux VM verification](linux-verification.md). Each report identifies its archive and test scope.
