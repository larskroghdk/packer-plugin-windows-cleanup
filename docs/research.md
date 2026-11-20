# 📚 Source review

Reviewed 19 November 2026. The legacy scripts are evidence of build intent, not evidence that an operation is safe on every Windows release. Dispositions in the mapping are engineering judgments informed by the primary sources below; exclusions do not imply Microsoft forbids every use of that feature.

| ID | Primary source | Consequence |
| --- | --- | --- |
| Packer | [Plugin creation](https://developer.hashicorp.com/packer/docs/plugins/creation) | SDK provisioner registration, HCL schema, plugin protocol and release naming |
| Shutdown | [QEMU shutdown step](https://github.com/hashicorp/packer-plugin-qemu/blob/main/builder/qemu/step_shutdown.go) | Builder runs command then waits for shutdown; timeout halts the build |
| Servicing | [Clean up WinSxS](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/clean-up-the-winsxs-folder) | Use DISM rather than directory deletion; ResetBase removes update-uninstall ability |
| DISM | [Package servicing commands](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/dism-operating-system-package-servicing-command-line-options) | AnalyzeComponentStore and StartComponentCleanup; preserve package metadata; avoid concurrent servicing |
| Health | [SFC](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/sfc) | Validation and repair are distinct; cleanup does not automatically repair Windows |
| Sysprep | [Generalize a Windows installation](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/sysprep--generalize--a-windows-installation) | Separate irreversible generalization; Store app provisioning can block Sysprep; preserve Panther evidence |
| Logs | [wevtutil](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/wevtutil) | Export selected logs before optional clearing; never default to clearing all logs |
| WER | [WER settings](https://learn.microsoft.com/en-us/windows/win32/wer/wer-settings), [cleanmgr](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/cleanmgr), [Server Disk Cleanup](https://learn.microsoft.com/en-us/windows-server/storage/file-server/disk-cleanup) | Delegate eligibility to installed cleanup handler; don't disable reporting or copy GUI tools into Core |
| Storage | [manage-bde](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/manage-bde) | Storage/BitLocker state is separate from ordinary cleanup; no forensic-erasure claim |
| Core | [Manage Server Core](https://learn.microsoft.com/en-us/windows-server/administration/server-core/server-core-manage) | Capability probing, headless execution, no GUI-dependent core operations |

No Microsoft source establishes that wholesale deletion of update datastores, audit logs, network registry state, installer caches or all inactive profiles is required for reusable templates. Those actions are intentionally excluded or replaced with narrow mechanisms. Firmware, licensing, antimalware, cloud-agent and hardening changes remain with their specialist build layers and platform support documentation; they are not generalized as cleanup recommendations here.
