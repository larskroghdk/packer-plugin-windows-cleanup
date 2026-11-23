# Agent instructions

`windows-cleanup` is a Packer provisioner for the final cleanup and validation of Windows VM templates over WinRM. It deletes files on the guest, so the safety rules below are hard constraints. A change that breaks one is wrong even if the tests pass. The detailed contracts are in [docs/architecture.md](docs/architecture.md), [docs/safety.md](docs/safety.md), [docs/sysprep.md](docs/sysprep.md) and [docs/testing.md](docs/testing.md).

## Module map

| Path | Role |
| --- | --- |
| `main.go` | Plugin registration and version |
| `provisioner/` | Configuration, validation, operation planning, communicator calls (`remote.go`), host report |
| `guest/scripts/` | Embedded Windows PowerShell 5.1 operations: `WindowsCleanup.ps1`, the entry script and the Sysprep finalizer |
| `internal/report/` | Report and guest fact types |
| `test/pester/` | Guest tests confined to Pester `TestDrive` |
| `test/integration/` | Opt-in live tests on a disposable VM |

## Cleanup safety

- Never widen deletion roots to make a test pass. Before any change, validate path syntax, the allowed boundaries, every ancestor and each visited entry for reparse points.
- Preserve Windows Installer, WinSxS and the servicing stores, production profiles, networking, activation and WinRM. Use DISM for component cleanup.
- A new destructive operation needs explicit configuration, a documented legacy or source rationale, dry-run behavior, reporting and behavior tests. Update [docs/legacy-mapping.md](docs/legacy-mapping.md) when a legacy operation's disposition changes.
- Preserve exclusions, manifest restrictions and idempotency. Don't add arbitrary command execution, and don't claim deleted data can be recovered.
- Sysprep and shutdown belong to the builder. The ordinary provisioner never shuts down or generalizes. Follow the finalizer staging contract in docs/sysprep.md.
- Transport, timeout and final-validation failures always fail the build, even with `fail_on_error = false`. Keep the pending-reboot and active-servicing gates, the operation deadlines and the WinRM postflight check.
- Don't remove a timed-out guest workspace while servicing child processes may still be running.

## Go

- Use the Go version in [go.mod](go.mod). Run every guest call through `remote.action`, which exchanges JSON files. Keep user-supplied data out of command text.
- After changing `Config`, run `go generate ./provisioner` (needs `packer-sdc` v0.6.13) and commit the regenerated `config.hcl2spec.go`.
- Update [docs/configuration.md](docs/configuration.md) and the examples when behavior changes.

## PowerShell

- Guests run elevated, 64-bit Windows PowerShell 5.1, including Server Core. Keep `.ps1` files ASCII with CRLF line endings.
- Keep incidental output off the success pipeline and check native exit codes.
- Keep guest tests inside Pester `TestDrive` or an explicitly authorized disposable VM. Never run cleanup against the development host.

## Checks

Run `./scripts/test.ps1` from PowerShell before committing. CI ([.github/workflows/test.yml](.github/workflows/test.yml)) runs the same script. It checks that `go.mod` is tidy, then runs gofmt, build, vet, staticcheck, govulncheck, Go tests, PSScriptAnalyzer, `packer fmt` and Pester under Windows PowerShell 5.1. It needs PSScriptAnalyzer 1.22.0 and Pester 5.6.1. For configuration changes, also check that `go generate ./provisioner` leaves no diff.

Add behavior tests for changed cleanup logic, covering dry run, path boundaries, reparse points, exclusions and failures where they apply. Mocks and local checks don't validate a real Windows image, so report them separately from live results.

## Skills

- [release-plugin](.agents/skills/release-plugin/SKILL.md): CI failures, packaging and authorized signed releases.

Activating a skill grants no authority to publish or to operate on a guest. [docs/agent-guidance.md](docs/agent-guidance.md) explains how each tool finds these files.
