# Security policy

## Supported versions

This project is pre-1.0 and has a single release line. Fixes land on `main` and ship in the next release; older releases are not patched.

## Reporting a vulnerability

When private vulnerability reporting is available, use "Report a vulnerability" on the Security tab. If it is unavailable, [open a public issue](https://github.com/larskroghdk/packer-plugin-windows-cleanup/issues/new) asking the maintainer for a private channel. Do not include vulnerability details in that issue.

Include the plugin version, Packer version and builder, guest OS and build, the provisioner block with secrets removed, the relevant JSON report or `PACKER_LOG=1` output with secrets and sensitive file names removed, steps to reproduce, and impact.

Do not include real credentials. Reproduce only on a disposable VM you own or are authorized to test. Never run the plugin against a production system or your workstation to demonstrate deletion.

## Scope

The plugin runs elevated Windows PowerShell 5.1 on the guest over WinRM and deletes files. In scope:

- Deletion outside the allowed roots and explicit manifests, including escapes through reparse points, junctions, symbolic links, path syntax or an ancestor directory.
- Removal of excluded paths, protected stores (Windows Installer, WinSxS, servicing data, the Windows Update datastore), the communicator or loaded profiles, the Security event log, activation state, networking or WinRM configuration.
- A dry run that changes the guest.
- Configuration or guest data reaching PowerShell command text instead of passing through the JSON files exchanged by the guest protocol.
- A guest workspace, uploaded script or staged Sysprep finalizer that a non-administrator can read or replace.
- Credentials or WinRM passwords exposed in Packer UI output, logs or release artifacts.
- A transport, timeout or final-validation failure, or an active-servicing or pending-reboot gate, that does not fail the build when it should.
- A registry change other than the Disk Cleanup `StateFlags65530` slot, or that slot not restored after a normally completed run.
- An operation that reports success while leaving servicing, the component store, WinRM or Sysprep finalizer state broken or inconsistent (system integrity).
- Weaknesses in the release or CI workflows, including unpinned Actions or excessive token permissions, that could compromise published plugin binaries.

Out of scope:

- Limitations documented in [Safety and reports](https://github.com/larskroghdk/packer-plugin-windows-cleanup/blob/main/docs/safety.md): malicious concurrent path replacement on the guest, forensic erasure of deleted files, copies in snapshots or host logs, and sensitive file names in the host report, which belongs in an access-controlled directory.
- Vulnerabilities in Windows, DISM, Disk Cleanup, Sysprep or WinRM. Report these to the [Microsoft Security Response Center](https://msrc.microsoft.com/report).
- Vulnerabilities in Packer, its builders or the Packer plugin SDK. Report these through [HashiCorp's security policy](https://www.hashicorp.com/security).
- Unexpected cleanup results that cannot be triggered across a boundary listed above. Report these as a bug.

## Process

The maintainer handles reports on a best-effort basis and coordinates a disclosure date with the reporter. Fixed issues are published as GitHub Security Advisories.

## Verifying releases

Download the `SHA256SUMS` file and its detached signature from the GitHub release. Verify the signature with GPG using the maintainer's release signing key, then check the plugin ZIP against the checksums.
