# Working on windows-cleanup

Use the Packer SDK lifecycle and JSON guest protocol. Keep PowerShell compatible with 5.1, ASCII and CRLF. Run `./scripts/test.ps1` before delivery. Guest mutation tests must remain confined to Pester TestDrive or explicit disposable live VMs.

Never expand deletion roots to make a failing test pass. Check syntax, path boundaries, every ancestor and each visited entry for reparse points. Keep Installer, WinSxS, servicing stores, production profiles, networking, activation and WinRM intact. New destructive operations require explicit configuration, a legacy/source rationale, dry-run behavior, reporting and behavior tests.

Sysprep is builder-owned; do not invoke shutdown/generalization in the ordinary provisioner. Transport, timeout and final validation failures must remain build failures. Do not claim real Windows validation from mocks. Maintain docs/legacy-mapping.md when the legacy disposition changes.
