# GitHub Copilot instructions

## Project and implementation

Follow [AGENTS.md](../AGENTS.md). This repository implements a Go Packer provisioner for final cleanup and validation of Windows VM templates over WinRM.

- Use the Packer Plugin SDK lifecycle and existing configuration, communicator, reporting and embedded-script patterns.
- Exchange guest inputs and results through the JSON guest protocol. Keep user-supplied data out of executable command strings.
- Keep guest scripts compatible with Windows PowerShell 5.1 and Server Core. Use ASCII and CRLF for PowerShell files.
- Generate HCL2 schema changes with `go generate ./provisioner`; do not maintain a handwritten parallel schema.
- Keep changes focused and update configuration documentation and examples when behavior changes.

## Cleanup safety

- Never expand deletion roots to make a failing test pass. Validate path syntax, allowed boundaries, every ancestor and each visited entry for reparse points before mutation.
- Preserve Windows Installer, WinSxS and servicing stores, production profiles, networking, activation and WinRM. Use supported servicing interfaces for component cleanup.
- New destructive operations require explicit configuration, a legacy/source rationale, dry-run behavior, reporting and behavior tests.
- Preserve exclusions, manifest restrictions and idempotency. Do not introduce arbitrary command execution or claim deleted data can be rolled back.
- Keep guest mutation tests inside Pester `TestDrive` or explicitly authorized disposable live VMs. Never run cleanup against the development host.

## Lifecycle and failures

- Sysprep and shutdown are builder-owned. Do not invoke shutdown or generalization from the ordinary provisioner; follow the documented finalizer staging contract.
- Keep transport, timeout and final validation failures fatal, including when `fail_on_error=false`.
- Preserve pending-reboot and active-servicing gates, operation deadlines, failure reporting and WinRM postflight validation.
- Do not remove a timed-out guest workspace while servicing children may still be running.

## Validation and documentation

- Run `./scripts/check.ps1` before delivery. It covers Go formatting, build, vet, staticcheck and tests, PowerShell analysis, HCL formatting and Pester under Windows PowerShell 5.1.
- Add behavior tests for changed cleanup logic, including relevant dry-run, path-boundary, reparse-point, exclusion and failure cases.
- When configuration changes, regenerate the HCL2 schema and verify the generated diff.
- Maintain [docs/legacy-mapping.md](../docs/legacy-mapping.md) when a legacy operation's disposition changes.
- Consult [architecture](../docs/architecture.md), [safety](../docs/safety.md), [Sysprep](../docs/sysprep.md) and [testing](../docs/testing.md) for the detailed contracts.
- Report checks actually run and any limitations. Mocks and local checks do not establish real Windows image validation; production validation requires the documented disposable VM matrix.
