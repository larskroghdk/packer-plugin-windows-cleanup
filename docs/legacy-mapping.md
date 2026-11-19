# Sealing layer review and disposition

Source: `C:\Git\Atea-TemplateFactory\DK-Factory\scripts\windows\600_SealingLayer`. All files at review time are inventoried with content hashes in [legacy-inventory.md](legacy-inventory.md). No supporting files beyond the inventoried PowerShell scripts were present. Commands, registry/service targets, exit-code handling, paths and OS gates were reviewed rather than copying scripts wholesale.

OS legend: **All** means the feature is relevant to Server 2016–2025/Windows 10–11 subject to edition/capability, not that the legacy implementation was validated on all of them. **Client** means client/desktop-specific; **Platform** means dependent on hardware, role or cloud agent. Legacy version tests often match only old captions or major version 10 and do not establish comprehensive support. Sources refer to [research.md](research.md). “Excluded” means outside this cleanup plugin; retain a separately reviewed build step if needed.

| Legacy script | Purpose / affected component | OS relevance | Risk and template suitability | Plugin decision / source rationale |
| --- | --- | --- | --- | --- |
| Add-AteaFirstBootTask | SYSTEM startup task for FirstBootCommands.cmd | All; Task Scheduler | Persistent privileged task; unsuitable as generic cleanup | Excluded; deployment-specific first-boot design |
| Add-AteaHealthReport | HTML health/log report with optional remote AI analysis | All, capability-dependent | Logs can contain secrets; network transfer and external credentials | Modified to local structured JSON health/cleanup report; no external AI dependency; Health/Logs |
| Add-AteaLoginMessage | Legal notice registry values | All | Organization policy, no cleanup value | Excluded; hardening layer |
| Add-AteaStartupScript | Elevated startup PowerShell scheduled task | All | Leaves executable privileged state in published image | Excluded; deployment policy |
| Clear-AteaAllEventLogs | Enumerate and clear every channel | All | Destroys audit and troubleshooting evidence; unsafe default | Modified: explicit channels, export first, Security protected, default off; Logs |
| Clear-AteaBranchCache | Clear BranchCache content cache | Server/client where feature installed | Workload cache state and performance effects | Excluded from generic cleanup; feature owner may call supported Clear-BCCache |
| Clear-AteaErrorReports | Recursive WER Temp/archive/queue deletion | All; desktop cleanup availability varies | Can remove active reports and evidence | Modified to optional installed cleanmgr WER handler; no direct WER directory deletion; WER |
| Convert-AteaCnspecJsonToNunit | Compliance result format conversion | All; external cnspec input | Evidence processing, not OS cleanup | Excluded; retain in reporting pipeline |
| Copy-AteaTestResults | Collect build/compliance evidence | All | Must preserve evidence before deletion | Modified to explicit log/event exports; external compliance pipeline remains responsible; Logs |
| Delete-AteaSoftwareDistributionDownloads | Stop update services and delete download cache | All | Active payload races; service startup changes; update regressions | Excluded direct deletion; windows_update uses DISM superseded components; Servicing/DISM |
| Disable-AteaAutologon | Disable AutoAdminLogon and remove registry password | All | Needed for credential hygiene but impacts unattended boot design; LSA secrets not covered | Excluded registry change; explicit secret files via artifact manifest, credential revocation outside plugin |
| Disable-AteaComputerRestore | Disable System Restore | Client where available | Changes recovery policy, loses restore capability | Excluded; not template cleanup |
| Disable-AteaInsecureWinRM | Change auth, delete listeners, disable firewall rules | All | Terminates Packer access and may prevent final capture | Excluded; preserve communicator through shutdown; Packer/Shutdown |
| Disable-AteaRemoteDesktop | Disable RDP via Terminal Server registry | All where RDP exists | Changes management policy | Excluded; hardening layer |
| Edit-AteaDeprovisioningScript | Rewrite an external deployment/deprovisioning script | Platform | String edits depend on agent/version; may bypass supported agent flow | Excluded; platform-owned deprovisioning |
| Enable-AteaCertificateRevocation | Secure Boot AvailableUpdates and update task | Platform UEFI | Firmware trust/revocation may break boot or require staged reboots | Excluded; separate firmware/security servicing |
| Enable-AteaKeyboardAndMouse | Restore input-related registry configuration | Platform | Driver/input policy; not cleanup | Excluded; hardware/optimization layer |
| Enable-AteaPagefile | Enable automatic pagefile via legacy WMI | All | Memory/dump requirements and reboot effects | Excluded; preserve current pagefile configuration |
| Enable-AteaUAC | Set EnableLUA and administrator consent policy | All | Security baseline and reboot implications | Excluded; preserve approved hardening |
| Get-AteaSecureBootUpdateTaskStatus | Query Secure-Boot-Update scheduled task/status | Platform UEFI/task present | Useful specialized evidence, insufficient generic health test | Excluded; retain in firmware validation pipeline |
| Invoke-AteaAppxRepair | Re-register every Appx manifest | Client/desktop Appx installations | Broad package mutation can create Sysprep user/provisioning mismatches | Excluded repair; finalizer detects conservative Appx blockers; Sysprep |
| Invoke-AteaAutomaticMaintenance | Start automatic maintenance and wait for tasks | All, task-dependent | Unbounded/indirect workload changes and concurrent cleanup | Excluded; deterministic selected servicing only; DISM |
| Invoke-AteaCheckDisk | Schedule chkdsk /F and mark volumes dirty | All | Forces boot repair/reboots, changes disk state before capture | Excluded automatic repair; prepare healthy disks before cleanup; Health |
| Invoke-AteaDiskDefrag | ReTrim/defrag volume | Platform storage support | VM storage/backend-specific, not proof of erasure or reclaimed capacity | Excluded; storage layer handles compaction/TRIM |
| Invoke-AteaDotNETNativeImageGeneratorRecompile | Run framework NGen compilation | All where .NET Framework installed | Runtime-specific workload; can leave background compilation | Excluded; software installation/optimization phase |
| Invoke-AteaHealthCheck | DISM CheckHealth/RestoreHealth and SFC repair | All | Repairs can download content and require reboot; exit success alone is insufficient health evidence | Modified to read health state without automatic repair; Health/DISM |
| Invoke-AteaImageCleanup | Restore/cleanup components, ResetBase, update cache and temp deletion | All, source/edition dependent | Mixes repair, irreversible servicing and broad deletion; stops services | Split: supported DISM cleanup, opt-in ResetBase, guarded Temp, no datastore deletion; Servicing/DISM |
| Invoke-AteaMpScan | Signature update and Defender scan | All where Defender available | Network/time cost, threat policy belongs to security validation | Excluded; run before sealing and retain evidence |
| Invoke-AteaSecureDelete | Download SDelete/zero free space or ReTrim | Platform storage | Thin-provisioning growth, downloads, timeouts; no universal forensic guarantee | Excluded; ordinary manifest deletion makes no secure-erasure claim; Storage |
| Invoke-AteaSysprep | Validate then execute generalization/shutdown | All subject to Microsoft restrictions | WinRM disappears; ordinary provisioner cannot safely continue | Modified to explicit QEMU builder-owned staged finalizer, exclusive attempt marker; Sysprep/Shutdown |
| Invoke-AteaSysprepAWS | EC2Launch/EC2Config-specific image preparation | AWS agent/version dependent | Generic Sysprep can bypass cloud initialization | Excluded; cloud builder/agent owns supported finalization |
| New-AteaCnspecScanReport | Generate HTML compliance report | All; external input | Evidence contains sensitive data and belongs outside image | Excluded; preserve/export through compliance pipeline |
| New-AteaSetupComplete2BatchJob | Persist setup-completion commands | All; unattend/setup design | Postdeployment privileged execution and ordering | Excluded; deployment system owns first boot |
| Remove-AteaAppxPackages | Remove Appx packages for all users | Client/desktop Appx | Can break apps, servicing and Sysprep provisioning parity | Excluded; no mass package removal; Sysprep |
| Remove-AteaAzureArcSetupFeature | Disable Arc optional feature and registry state | Server versions exposing feature | Role/feature removal and pending reboot | Excluded; approved feature baseline before cleanup; DISM |
| Remove-AteaAzureGuestAgent | Stop/delete Azure agent services and startup registry | Azure/platform | Breaks cloud provisioning, support and management | Excluded; platform image policy |
| Remove-AteaCachedNetworkList | Delete network profile registry keys | All | Firewall profile/network identity may change | Excluded; preserve network configuration |
| Remove-AteaDisabledFeatures | Remove all disabled feature payloads with DISM | All; optional features | Eliminates repair/re-enable sources, future install may need media | Excluded payload removal; component cleanup only; DISM |
| Remove-AteaFiles | Delete OS.ready and unattend files | All | Answer files can hold secrets but may still be needed | Modified: only explicitly listed exact files; no recursive parent deletion; Sysprep |
| Remove-AteaFolders | Delete .nupkg installers and organization Tools | All; organization layout | Broad tools deletion can damage installed apps/caches | Modified: explicit installer/artifact manifests in dedicated staging root |
| Remove-AteaIsoMounts | Eject optical media through Shell.Application | Platform/desktop COM | GUI/COM dependency and source-media availability | Excluded; builder owns attached media and capture |
| Remove-AteaLastKnownDhcpServer | Rewrite interface DHCP server registry | All | Unsupported network-state manipulation, leases/identity effects | Excluded; preserve required networking |
| Remove-AteaLocalAccountTokenFilterPolicy | Delete remote UAC filtering policy | All | Can revoke elevated WinRM access mid-build | Excluded; hardening must coordinate communicator access |
| Remove-AteaMondooFiles | Remove user Mondoo configuration/credentials | All where installed | Exact secret target, not arbitrary user data; deletion does not revoke key | Modified: stage explicit build credentials under approved roots; revocation external |
| Remove-AteaMSPaint | Remove Paint Windows capability | Client/capability dependent | Application/feature policy and repair media requirements | Excluded; feature baseline before cleanup; DISM |
| Remove-AteaOldBCDentries | Delete firmware boot entries | Platform UEFI | May render template unbootable or remove valid recovery paths | Excluded; builder/firmware maintenance |
| Remove-AteaPagefile | Disable automatic pagefile/remove settings | All | Affects memory, crash dumps and reboots | Excluded; preserve OS memory configuration |
| Remove-AteaPagefileFromDisk | Delete pagefile.sys directly | All | Active protected system file; crash/recovery impact | Excluded; not a temporary artifact |
| Remove-AteaRDPCertificateHash | Remove RDP listener certificate hash | All with RDP | Changes TLS identity and remote management trust | Excluded; deployment/hardening workflow |
| Remove-AteaRemoteDesktopBlock | Change RDP policy to restore access | All with RDP | Security/management policy, no cleanup purpose | Excluded; do not undo prior hardening |
| Remove-AteaSnippingTool | Remove Snipping Tool capability | Client/capability dependent | App removal and servicing/source requirements | Excluded; feature policy |
| Remove-AteaStaticPersistentRoutes | Remove persistent network routes | All | Can immediately break WinRM and deployed routing | Excluded; preserve networking |
| Remove-AteaWindowsAdminCenterSetup | Remove Admin Center setup feature | Server/capability dependent | Installation feature baseline rather than temporary files | Excluded feature removal; downloaded installers may use explicit staging manifest |
| Remove-AteaWindowsFeedbackHub | Remove Feedback Hub capability | Client/capability dependent | Package policy, future servicing impact | Excluded; feature baseline |
| Remove-AteaWindowsOldFolder | Recursively delete Windows.old | Upgraded installations | Removes OS rollback data, ACL and servicing risks | Excluded direct deletion; supported OS cleanup belongs to a separately authorized upgrade-retention policy |
| Remove-AteaWindowsServerBackup | Remove server backup feature | Server where available | Removes recovery capability/feature payload | Excluded; preserve approved roles |
| Remove-AteaWindowsWirelessNetworking | Remove wireless networking feature | Server/feature dependent | Can break connectivity and device setup | Excluded; preserve networking |
| Set-AteaFilterAdministratorToken | Change administrator UAC policy | All | Authentication/elevation side effects | Excluded; hardening layer |
| Set-AteaKmsServer | Set licensing server | All; licensing edition dependent | Activation identity/network/licensing policy | Excluded; preserve licensing state |
| Set-AteaMAKKey | Install MAK key | All; licensing edition dependent | Key exposure/activation consumption | Excluded; licensing belongs to deployment policy |
| Set-AteaWindowsDefenderPreferences | Configure Defender policies and protection settings | All with Defender | Security baseline changes and capability differences | Excluded; preserve hardening and diagnostics |
| Start-AteaMRTScan | Run Malicious Software Removal Tool | Tool available on applicable images | Scan completion/exit handling and scope vary; not cleanup | Excluded; separate security scan with retained evidence |
| Test-AteaForSecureBootChanges | Trigger task and inspect Secure Boot events | Platform UEFI/current updates | Trigger has side effects/reboot implications; event IDs are version-specific | Excluded; specialized firmware pipeline |
| Test-AteaSecureBootCertificates | Read DB/DBX, mount EFI and inspect signer | Platform UEFI | Temporary drive-letter collisions; firmware-specific state | Excluded; specialized signed boot validation |
| Test-AteaSecureBootUEFICA2023 | Check UEFI CA data | Platform UEFI | String matching does not certify complete boot-chain safety | Excluded; specialized firmware validation |
| Test-AteaWMIrepository | Verify/salvage WMI repository | All | Automatic salvage mutates provider registrations; localization-sensitive detection | Excluded repair; CIM detection failures fail cleanup instead of resetting repository |
| Update-AteaBootManager | Set Secure Boot AvailableUpdates and trigger update | Platform UEFI/current updates | Boot trust transition and required staged reboots | Excluded; firmware/security servicing before cleanup |
| Update-AteaCertificateDefinitions | Add UEFI CA through AvailableUpdates/task | Platform UEFI/current updates | Firmware trust updates can affect compatibility | Excluded; firmware/security servicing |
| Update-AteaFirmware | Update Secure Boot SVN | Platform UEFI/current updates | Anti-rollback change may be irreversible | Excluded; separate explicit firmware authorization |
| Wait-AteaForMinutes | Fixed sleep before capture | All | Time passing is not proof of update/health completion | Replaced with state validation, action deadlines and builder shutdown wait; Shutdown/DISM |

New operations without a direct legacy script: communicator-user Recycle Bin cleanup; explicit unloaded temporary profile deletion using CIM with conservative link refusal; required service/software/optional feature validation; machine-readable measured before/after reporting. These remain opt-in where they delete user-scoped content.
