# Security policy

## Supported versions

Only the latest release receives security fixes. Fixes ship in a new release, not as backports.

| Version | Supported |
| --- | --- |
| Latest release | Yes |
| Older releases | No |

## Reporting a vulnerability

Do not open a public issue, pull request or discussion.

Use [private vulnerability reporting](https://github.com/larskroghdk/packer-plugin-windows-cleanup/security/advisories/new) through the repository's "Report a vulnerability" button on the Security tab.

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
- Weaknesses in the release or CI workflows, including unpinned Actions or excessive token permissions, that could compromise published plugin binaries.

Out of scope:

- Limitations documented in [Safety and reports](https://github.com/larskroghdk/packer-plugin-windows-cleanup/blob/main/docs/safety.md): malicious concurrent path replacement on the guest, forensic erasure of deleted files, copies in snapshots or host logs, and sensitive file names in the host report, which belongs in an access-controlled directory.
- Vulnerabilities in Windows, DISM, Disk Cleanup, Sysprep or WinRM. Report these to the [Microsoft Security Response Center](https://msrc.microsoft.com/report).
- Vulnerabilities in Packer, its builders or the Packer plugin SDK. Report these through [HashiCorp's security policy](https://www.hashicorp.com/security).
- Unexpected cleanup results that cannot be triggered across a boundary listed above. Report these as a bug.

## Process

The maintainer acknowledges reports, confirms or rejects the issue, and agrees a disclosure date with the reporter. The target acknowledgement time is within 7 days. This is a single-maintainer project, so timelines are best effort.

Fixes are released in a new version and disclosed through a GitHub Security Advisory. A CVE is requested through GitHub when appropriate. Reporters are credited unless they prefer not to be.

## Verifying releases

Download the `SHA256SUMS` file and its detached signature from the GitHub release. Verify the signature with GPG using the maintainer's release signing key, then check the plugin ZIP against the checksums.
