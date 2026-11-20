# ⚙️ Configuration

All attributes are optional unless required by another opt-in. HCL unknown attributes and wrong types are rejected by the generated SDK schema.

| Attribute | Type | Default / meaning |
| --- | --- | --- |
| `profile` | string | `safe` or `custom`; custom disables implicit temporary cleanup |
| `temporary_files` | bool | true for safe profile; stale Windows and user Temp content |
| `component_store` | bool | false; DISM analysis, StartComponentCleanup and post-analysis |
| `windows_update` | bool | false; selects the same supported superseded-component servicing once |
| `reset_component_base` | bool | false; adds irreversible ResetBase, requires servicing selection |
| `installer_cleanup` | bool | false; explicit `installer_paths` manifest |
| `provisioning_artifacts` | bool | false; explicit `artifact_paths` manifest |
| `log_cleanup` | bool | false; explicit `log_paths` manifest |
| `error_reporting` | bool | false; installed cleanmgr WER handler; Windows selects eligible reports |
| `empty_recycle_bin` | bool | false; communicator user's bin on SystemDrive |
| `remove_user_profiles` | bool | false; explicit unloaded `build_profile_sids`, restricted build naming |
| `clear_event_logs` | bool | false; requires explicit `event_logs` and `log_export_path`; Security always protected |
| `validate_system` | bool | true; health, reboot/update completion, WinRM, required services/software/features |
| `report_cleanup` | bool | true; write host JSON even on failure |
| `dry_run` | bool | false; no cleanup, finalizer staging, export or registry mutation; read-only health/DISM queries still log via Windows |
| `fail_on_error` | bool | true; false downgrades ordinary category failures, never explicit provisioning artifacts, transport or final validation |
| `unsupported_policy` | string | `error` or `skip` for unavailable selected capabilities; unsupported OS always rejected |
| `pending_reboot_policy` | string | `error` or `skip_servicing`; latter skips servicing/health/finalization for pending reboots and emits warnings; active updates always block cleanup |
| `operation_timeout` | duration string | `60m`; per guest action, 1s–24h |
| `minimum_age_hours` | integer | 24; positive; applies to ordinary Temp files, not explicit manifests |
| `report_path` | host path string | `windows-cleanup-report.json`; use unique paths for concurrent builds |
| `log_level` | string | `info` or `quiet`; never emits input manifests or file contents to UI |
| `cleanup_paths` | list(string) | extra targets within fixed temporary/staging roots |
| `installer_paths`, `artifact_paths`, `log_paths` | list(string) | empty; explicitly authorizes recent-file removal within allowed roots |
| `exclusions` | list(string) | literal file/subtree paths, case insensitive; no wildcard expansion |
| `build_profile_sids` | list(string) | empty; exact S-1-5-21 user SIDs, never select by inactivity |
| `event_logs` | list(string) | empty; exact channel names, e.g. Application; Security cannot be cleared |
| `log_export_path` | guest path string | empty; when set, export logs before cleanup under ProgramData\PackerCleanup\Exports; exported log manifests must list files |
| `required_services` | list(string) | empty; additional services that must be Running; WinRM/RpcSs always checked |
| `required_software_paths` | list(string) | empty; files that must exist, no Win32_Product repair side effects |
| `required_features` | list(string) | empty; DISM optional feature names that must be Enabled |
| `sysprep` | bool | false; stage only, QEMU strategy; see lifecycle documentation |

## 📂 Allowed paths

The guest derives Windows Temp, real non-special user profiles' AppData\Local\Temp, and ProgramData\PackerCleanup\Staging. The host validates absolute local syntax; the guest enforces these roots. Roots are fixed and cannot be extended through HCL. Move application staging into the dedicated staging root earlier in the build.

Only `artifact_paths` additionally permits these exact files: `%SystemRoot%\Panther\Unattend.xml`, `%SystemRoot%\Panther\Unattend\Unattend.xml`, `%SystemRoot%\System32\Sysprep\Unattend.xml`, and `%SystemDrive%\OS.ready`. Their parent directories remain protected. Ensure no remaining build step needs an answer file before selecting it.

Configuration paths do not expand environment variables or globs. Specify actual guest paths. UNC/device paths, ADS, dot segments, repeated separators, DOS device names and trailing dots/spaces are rejected. Every existing ancestor is checked for links. A discovered link is a reported failure rather than permission to traverse it.
