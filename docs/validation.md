# Local validation record

Validated 19 November 2026 on a Windows development host:

- `scripts/check.ps1`: passed Go build, vet (including integration-tagged harness), staticcheck, Go tests, PowerShell warning/error analysis, HCL formatting and 22 Pester tests in Windows PowerShell 5.1.
- Native command behavior: verified captured stdout, exit 0, reboot-required exit 3010, failure exit 7 and timeout failure using harmless child processes.
- `go generate ./provisioner`: reproducible schema hash; `go mod tidy -diff`: no changes.
- Built `dist/packer-plugin-windows-cleanup.exe`, installed it into workspace-local Packer plugin storage, and validated both example and integration HCL through Packer.
- `goreleaser check`: passed.
- `go test -tags integration -v ./test/integration`: harness compiled and skipped because `WC_TEST_HOST` was unset. No live cleanup, servicing, reboot, generalization or clone certification was performed.

The optional `go test -cover` command encountered a local toolchain error resolving `runtime/coverage`; no coverage percentage is claimed. The ordinary test suite and required checks pass independently.

The [VM matrix](testing.md) and the [QEMU shutdown/clone checks](sysprep.md) remain release gates. GitHub workflows are supplied but have not been executed remotely. No push, tag or public release was created by this implementation task.
