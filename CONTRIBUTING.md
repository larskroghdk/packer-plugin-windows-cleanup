# Contributing

Read [AGENTS.md](AGENTS.md), [architecture](docs/architecture.md) and [safety](docs/safety.md). Use the Go version in go.mod. Regenerate HCL after changing Config: `go generate ./provisioner` (packer-sdc from the SDK version must be on PATH). Run `./scripts/check.ps1` from PowerShell. Live tests require a disposable VM and separate opt-in.

Document cleanup support, failure behavior and limitations. No new arbitrary path roots, silent errors, automatic reboots, or inferred user-profile deletion. Preserve the reference project's author attribution for reused code. New APIs should have focused Go doc comments; guest functions need comment-based help and script author/version metadata.
