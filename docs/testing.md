# 🧪 Testing and release gates

Run `./scripts/test.ps1`. It builds, vets, runs staticcheck and Go tests, checks HCL formatting and PowerShell analysis, and runs Pester under **Windows PowerShell 5.1**, not only pwsh. Install Pester 5.6.1 or later and PSScriptAnalyzer 1.22.0. Use packer-sdc v0.6.13 for `go generate ./provisioner`; generated output must have no diff. Go dependencies/toolchain are pinned to the reference project and checksum file.

Go tests cover plugin describe registration, real HCL type decoding, defaults/category coalescing, path syntax, supported OS classification, protocol transport failures, deadline propagation, reports, pending reboots, unsupported capability policy, fail-on-error, WinRM gating and finalizer-only Sysprep. Pester exercises actual TestDrive deletions, stale/fresh files, manifests, idempotency, exclusions, locks and junction refusal. Mocked servicing checks commands/ResetBase/dry-run; health tests inspect state, not just exit success. Native exit/timeout tests run harmless child PowerShell only. These tests never clean the development host.

## 🧫 Disposable live integration

Create a VM checkpoint outside the plugin. Seed `C:\ProgramData\PackerCleanup\Staging\integration-owned.txt` on the disposable guest, using an elevated session. Finish all updates/restarts. Use an elevated native WinRM user and a trusted HTTPS listener certificate.

```powershell
$env:WC_TEST_HOST = 'disposable.example.test'
$env:WC_TEST_USERNAME = 'Administrator'
$env:WC_TEST_PASSWORD = '<short-lived credential>'
go test -tags integration -count=1 -v -timeout 90m ./test/integration
# Only for a VM authorized for mutation:
$env:WC_TEST_ALLOW_MUTATION = 'yes'
go test -tags integration -count=1 -v -timeout 90m ./test/integration
```

Default is dry-run. The explicit mutation opt-in removes only the seeded manifest file, applies twice and verifies idempotency and WinRM postflight. `WC_TEST_USE_SSL=false` is available only for an isolated test network; certificate verification is never disabled by this harness. Reports and plugin installation use a private temporary host directory. Destroy/restore the VM after testing. This suite does not run in ordinary CI and has not been run without a supplied disposable guest.

## ✅ Required release matrix

Run Server 2016/2019/2022/2025 with both Core and Desktop Experience; Windows 10 and Windows 11; AMD64 plus ARM64 where claimed. For each supported image, retain baseline, dry-run, cleanup twice, servicing after real updates, pending reboot rejection/skip, installer repair/update/uninstall checks, WER capability behavior, optional profile/exclusion tests, report measurements and WinRM continuity. Test with non-English Windows and nondefault SystemRoot/ProgramData volumes.

Run a separate QEMU build for Sysprep: stage last, execute only through shutdown_command, observe normal power-off, verify failure/timeout does not produce a released image, boot a clone through OOBE, verify generalized image state and diagnostics, and ensure another attempt is refused. Also test user Appx mismatches, domain-joined machines and pending servicing. Null-builder tests do not certify shutdown.

Publishing stays a signed **draft** until the matrix and image security/compliance checks are reviewed. GitHub requires `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE`; do not tag a public release before validating the tag's exact commit. A passing local suite is not evidence of completed VM tests.
