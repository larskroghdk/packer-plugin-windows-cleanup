#Requires -Version 5.1
<#
.SYNOPSIS
    Implements guarded cleanup operations for disposable Windows image builds.

.DESCRIPTION
    Dot-source this file.
    No operation runs at import.
    Paths are literal, confined to fixed roots, and checked for reparse points before traversal and deletion.

.NOTES
    Author: Lars Krogh
    Version: 0.1.1
#>

function Assert-WCPath {
    [OutputType([string])]
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)]
        [string]$Path
    )
    if ($Path -notmatch '^[a-zA-Z]:\\[^<>:"/|?*\x00-\x1f]+$') {
        throw 'Invalid absolute cleanup path'
    }
    foreach ($part in $Path.Substring(3).Split('\')) {
        if (-not $part -or $part -in @('.', '..') -or $part.TrimEnd(' ', '.') -ne $part -or $part -match '^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])(\.|$)') { throw 'Ambiguous cleanup path' }
    }
    $current = $Path
    while ($current) {
        if (Test-Path -LiteralPath $current) {
            $item = Get-Item -LiteralPath $current -Force -ErrorAction Stop
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) {
                throw 'Reparse point in cleanup path'
            }
        }
        $current = Split-Path -Parent $current
    }
    return [IO.Path]::GetFullPath($Path)
}

function Test-WCWithin {
    [OutputType([bool])]
    [CmdletBinding()]
    param(
        [string]$Path,
        [string]$Root,
        [switch]$AllowRoot
    )
    return ($AllowRoot -and $Path.Equals($Root, [StringComparison]::OrdinalIgnoreCase)) -or $Path.StartsWith($Root.TrimEnd('\') + '\', [StringComparison]::OrdinalIgnoreCase)
}

function Get-WCRoot {
    [CmdletBinding()]
    param()
    $roots = @((Join-Path $env:SystemRoot 'Temp'), (Join-Path $env:ProgramData 'PackerCleanup\Staging'))
    foreach ($buildProfile in Get-CimInstance -ClassName Win32_UserProfile -ErrorAction Stop) {
        if (-not $buildProfile.Special -and $buildProfile.LocalPath -and (Test-WCWithin $buildProfile.LocalPath ($env:SystemDrive + '\Users'))) {
            $roots += Join-Path $buildProfile.LocalPath 'AppData\Local\Temp'
        }
    }
    return $roots
}

function Assert-WCTarget {
    [CmdletBinding()]
    param(
        [string]$Path,
        [string[]]$Roots
    )
    $full = Assert-WCPath $Path
    $allowed = $false
    foreach ($root in $Roots) {
        if ($root -match '\\(Unattend\.xml|OS\.ready)$') {
            if ($full.Equals($root, [StringComparison]::OrdinalIgnoreCase) -and -not (Test-Path -LiteralPath $full -PathType Container)) {
                $allowed = $true
            }
        }
        elseif (Test-WCWithin $full $root -AllowRoot) {
            $allowed = $true
        }
    }
    if (-not $allowed) {
        throw 'Cleanup target is outside approved temporary/staging roots'
    }
    if ((Test-WCWithin $PSScriptRoot $full -AllowRoot) -and $full -ne (Join-Path $env:SystemRoot 'Temp')) {
        throw 'Cleanup target contains the running provisioner'
    }
    return $full
}

function Get-WCPendingRebootReason {
    [CmdletBinding()]
    param()
    $reasons = @()
    foreach ($key in @('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending', 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired')) {
        if (Test-Path -LiteralPath $key) { $reasons += $key }
    }
    $manager = Get-ItemProperty -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager' -ErrorAction Stop
    if ($manager.PSObject.Properties['PendingFileRenameOperations'] -and $manager.PendingFileRenameOperations) {
        $reasons += 'PendingFileRenameOperations'
    }
    if (Test-Path -LiteralPath (Join-Path $env:SystemRoot 'WinSxS\pending.xml')) {
        $reasons += 'CBS pending.xml'
    }
    if (Get-Command Get-WindowsPackage -ErrorAction SilentlyContinue) {
        $pending = @(Get-WindowsPackage -Online -ErrorAction Stop | Where-Object { $_.PackageState -in @('InstallPending', 'UninstallPending') })
        if ($pending.Count) {
            $reasons += 'Pending DISM packages'
        }
    }
    if (Get-Command Get-WindowsOptionalFeature -ErrorAction SilentlyContinue) {
        if (@(Get-WindowsOptionalFeature -Online -ErrorAction Stop | Where-Object { $_.State -in @('EnablePending', 'DisablePending') }).Count) {
            $reasons += 'Pending DISM features'
        }
    }
    return $reasons
}

function Get-WCFact {
    [OutputType([hashtable])]
    [CmdletBinding()]
    param()
    $os = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
    $cpu = Get-CimInstance -ClassName Win32_Processor -ErrorAction Stop | Select-Object -First 1
    $nativeArchitecture = switch ([int]$cpu.Architecture) {
        9 {
            'AMD64'
        };
        12 {
            'ARM64'
        };
        0 {
            'x86'
        };
        default {
            'Unknown'
        }
    }
    $version = Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion' -ErrorAction Stop
    $reasons = @(Get-WCPendingRebootReason)
    $installer = New-Object -ComObject Microsoft.Update.Installer
    $busy = [bool]$installer.IsBusy
    [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($installer)
    $busy = $busy -or @(Get-Process -ErrorAction Stop | Where-Object { $_.ProcessName -in @('TiWorker', 'MoUsoCoreWorker', 'msiexec', 'dism') }).Count -gt 0
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    try {
        $elevated = (New-Object Security.Principal.WindowsPrincipal($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    }
    finally {
        $identity.Dispose()
    }
    $winrm = (Get-Service -Name WinRM -ErrorAction Stop).Status -eq 'Running'
    $listeners = @(Get-ChildItem WSMan:\localhost\Listener -ErrorAction Stop)
    $winrm = $winrm -and $listeners.Count -gt 0
    return @{
        product_name = [string]$os.Caption; version = [string]$os.Version; build = [int]$os.BuildNumber
        product_type = [int]$os.ProductType; installation_type = [string]$version.InstallationType
        architecture = [string]$nativeArchitecture; elevated = $elevated
        native_process = [Environment]::Is64BitProcess -and $env:PROCESSOR_ARCHITECTURE -eq $nativeArchitecture; pending_reboot = $reasons.Count -gt 0
        pending_reasons = @($reasons); update_busy = $busy; winrm = $winrm
        capabilities = @{
            dism        = (Test-Path -LiteralPath (Join-Path $env:SystemRoot 'System32\dism.exe'))
            recycle_bin = [bool](Get-Command Clear-RecycleBin -ErrorAction SilentlyContinue)
            wer_cleanup = (Test-Path -LiteralPath (Join-Path $env:SystemRoot 'System32\cleanmgr.exe')) -and (Test-Path -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VolumeCaches\Windows Error Reporting Files')
        }
    }
}

function Invoke-WCNative {
    [OutputType([hashtable])]
    [CmdletBinding()]
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [int]$TimeoutSeconds = 3500
    )
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = $FilePath
    $start.Arguments = $Arguments -join ' '
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.WindowStyle = [Diagnostics.ProcessWindowStyle]::Hidden
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $start
    try {
        if (-not $process.Start()) {
            throw 'Native operation could not be started'
        }
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
            $process.Kill();
            throw 'Native operation timed out; discard this build or investigate servicing before capture'
        }
        $process.WaitForExit()
        $text = ($stdout.GetAwaiter().GetResult(), $stderr.GetAwaiter().GetResult()) -join "`n"
        if ($process.ExitCode -notin @(0, 3010)) {
            throw "Native operation failed with exit code $($process.ExitCode)"
        }
        return @{
            exit_code = $process.ExitCode;
            text      = $text
        }
    }
    finally { $process.Dispose() }
}

function Remove-WCItem {
    [CmdletBinding(SupportsShouldProcess)]
    param(
        [string]$Path,
        [string[]]$Roots,
        $Options,
        $Result,
        [switch]$Manifest
    )
    foreach ($excluded in @($Options.exclusions) + @($PSScriptRoot)) {
        if ($excluded -and (Test-WCWithin $Path $excluded -AllowRoot)) {
            return
        }
    }
    $full = Assert-WCTarget $Path $Roots
    if (-not (Test-Path -LiteralPath $full)) {
        return
    }
    $item = Get-Item -LiteralPath $full -Force -ErrorAction Stop
    if ($item.PSIsContainer) {
        foreach ($child in Get-ChildItem -LiteralPath $full -Force -ErrorAction Stop) {
            Remove-WCItem -Path $child.FullName -Roots $Roots -Options $Options -Result $Result -Manifest:$Manifest
        }
        if ($Roots -notcontains $full -and @(Get-ChildItem -LiteralPath $full -Force).Count -eq 0) {
            if (-not $Options.dry_run -and $PSCmdlet.ShouldProcess($full, 'Remove empty directory')) { $null = Assert-WCTarget $full $Roots; Remove-Item -LiteralPath $full -Force -ErrorAction Stop; $Result.removed.Add($full) }
        }
        return
    }
    if (-not $Manifest -and $item.LastWriteTimeUtc -gt [DateTime]::UtcNow.AddHours( - [int]$Options.minimum_age_hours)) { return }
    $Result.candidates.Add($full)
    if ($Options.dry_run -or -not $PSCmdlet.ShouldProcess($full, 'Remove build file')) {
        return
    }
    try {
        $handle = [IO.File]::Open($full, 'Open', 'Read', 'None'); $handle.Dispose()
    }
    catch [IO.IOException] {
        if ($Manifest) {
            throw 'Explicit build artifact is in use and could not be removed'
        }
        $Result.warnings.Add('In-use file skipped'); return
    }
    $null = Assert-WCTarget $full $Roots
    $length = $item.Length
    Remove-Item -LiteralPath $full -Force -ErrorAction Stop
    $Result.removed.Add($full); $Result.deleted_file_bytes += $length
}

function Invoke-WCFileCleanup {
    [CmdletBinding()]
    param(
        $Options,
        $Result
    )
    $roots = @(Get-WCRoot)
    $paths = @(); $manifest = $false
    switch ($Options.operation) {
        'temporary_files' {
            $paths = @($roots | Where-Object { $_ -ne (Join-Path $env:ProgramData 'PackerCleanup\Staging') }) + @($Options.cleanup_paths)
        }
        'installer_cleanup' {
            $paths = @($Options.installer_paths);
            $manifest = $true
        }
        'provisioning_artifacts' {
            $paths = @($Options.artifact_paths);
            $manifest = $true
            $roots += @((Join-Path $env:SystemRoot 'Panther\Unattend.xml'), (Join-Path $env:SystemRoot 'Panther\Unattend\Unattend.xml'), (Join-Path $env:SystemRoot 'System32\Sysprep\Unattend.xml'), ($env:SystemDrive + '\OS.ready'))
        }
        'log_cleanup' {
            $paths = @($Options.log_paths);
            $manifest = $true
        }
    }
    foreach ($path in $paths) {
        if ($path) {
            $null = Assert-WCTarget $path $roots
        }
    }
    foreach ($excluded in @($Options.exclusions)) {
        if ($excluded) {
            $null = Assert-WCPath $excluded
        }
    }
    if ($Options.operation -eq 'log_cleanup' -and $Options.log_export_path -and -not $Options.dry_run) {
        $export = Assert-WCPath $Options.log_export_path
        if (-not (Test-WCWithin $export (Join-Path $env:ProgramData 'PackerCleanup\Exports') -AllowRoot)) {
            throw 'Log exports must be under ProgramData\PackerCleanup\Exports'
        }
        foreach ($path in $paths) {
            if (-not $path -or -not (Test-Path -LiteralPath $path)) {
                continue
            }
            if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
                throw 'Exported log manifests must list individual files'
            }
            $null = New-Item -ItemType Directory -Path $export -Force
            $null = Assert-WCPath $export
            $destination = Join-Path $export ([guid]::NewGuid().ToString('N') + '-' + [IO.Path]::GetFileName($path))
            Copy-Item -LiteralPath $path -Destination $destination -ErrorAction Stop
            if ((Get-FileHash -LiteralPath $path).Hash -ne (Get-FileHash -LiteralPath $destination).Hash) {
                throw 'Log export verification failed'
            }
        }
    }
    foreach ($path in $paths) {
        if (-not $path) {
            continue
        }
        try {
            Remove-WCItem -Path $path -Roots $roots -Options $Options -Result $Result -Manifest:$manifest
        }
        catch {
            $Result.warnings.Add($_.Exception.Message); $Result.status = 'failed'; $Result.message = 'One or more cleanup targets could not be processed'
        }
    }
    if (-not $paths.Count) {
        $Result.status = 'skipped'; $Result.message = 'No manifest entries configured'
    }
}

function Invoke-WCServicing {
    [CmdletBinding()]
    param(
        $Options,
        $Result
    )
    $facts = Get-WCFact
    if ($facts.pending_reboot -or $facts.update_busy) {
        throw 'Servicing deferred because an update or reboot is pending'
    }
    $dism = Join-Path $env:SystemRoot 'System32\dism.exe'
    $before = Invoke-WCNative $dism @('/Online', '/English', '/Cleanup-Image', '/AnalyzeComponentStore', '/NoRestart')
    $Result.output = $before.text
    if (-not $Options.dry_run) {
        $arguments = @('/Online', '/English', '/Cleanup-Image', '/StartComponentCleanup', '/NoRestart')
        if ($Options.reset_component_base) {
            $arguments += '/ResetBase'
        }
        $cleanup = Invoke-WCNative $dism $arguments
        $Result.output += "`n" + $cleanup.text
        if ($cleanup.exit_code -eq 3010) {
            $Result.warnings.Add('DISM requires a reboot')
        }
        else {
            $after = Invoke-WCNative $dism @('/Online', '/English', '/Cleanup-Image', '/AnalyzeComponentStore', '/NoRestart'); $Result.output += "`n" + $after.text
        }
    }
}

function Test-WCSystem {
    [CmdletBinding()]
    param(
        $Options,
        $Result
    )
    $facts = Get-WCFact
    if ($facts.pending_reboot -or $facts.update_busy) {
        throw 'System is not ready: pending reboot or active updates'
    }
    if (-not $facts.winrm) {
        throw 'WinRM is not available'
    }
    foreach ($name in @('WinRM', 'RpcSs') + @($Options.required_services)) {
        if ($name -and (Get-Service -Name $name -ErrorAction Stop).Status -ne 'Running') {
            throw "Required service is not running: $name"
        }
    }
    foreach ($path in @($Options.required_software_paths)) {
        if ($path -and -not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw 'Required software file is missing'
        }
    }
    foreach ($feature in @($Options.required_features)) {
        if ($feature -and (Get-WindowsOptionalFeature -Online -FeatureName $feature -ErrorAction Stop).State -ne 'Enabled') {
            throw "Required optional feature is not enabled: $feature"
        }
    }
    $health = Repair-WindowsImage -Online -CheckHealth -NoRestart -ErrorAction Stop
    $Result.output = [string]$health.ImageHealthState
    if ($health.ImageHealthState -ne 'Healthy' -or $health.RestartNeeded) {
        throw 'Component store is unhealthy or requires a reboot'
    }
}

function Remove-WCBuildProfile {
    [CmdletBinding(SupportsShouldProcess)]
    param(
        $Options,
        $Result
    )
    $profiles = @(Get-CimInstance -ClassName Win32_UserProfile -ErrorAction Stop)
    $selected = @()
    foreach ($sid in @($Options.build_profile_sids)) {
        if ($sid -notmatch '^S-1-5-21-(\d+-){3}\d+$') {
            throw 'Invalid build profile SID'
        }
        $buildProfile = $profiles | Where-Object { $_.SID -eq $sid }
        if (-not $buildProfile) {
            continue
        }
        if ($buildProfile.Special -or $buildProfile.Loaded -or $buildProfile.SID -eq [Security.Principal.WindowsIdentity]::GetCurrent().User.Value) {
            throw 'Protected or loaded profile cannot be removed'
        }
        $path = Assert-WCPath $buildProfile.LocalPath
        if ($path -notmatch ('^' + [regex]::Escape($env:SystemDrive + '\Users\packer-build-') + '[a-zA-Z0-9_-]+$')) {
            throw 'Only explicitly named packer-build-* profiles can be removed'
        }
        $stack = New-Object 'Collections.Generic.Stack[string]'; $stack.Push($path)
        while ($stack.Count) {
            $directory = $stack.Pop();
            foreach ($item in Get-ChildItem -LiteralPath $directory -Force -ErrorAction Stop) {
                $null = Assert-WCPath $item.FullName; if ($item.PSIsContainer) {
                    $stack.Push($item.FullName)
                }
            }
        }
        $selected += $buildProfile
    }
    foreach ($buildProfile in $selected) {
        $Result.candidates.Add([string]$buildProfile.LocalPath)
        if (-not $Options.dry_run -and $PSCmdlet.ShouldProcess($buildProfile.LocalPath, 'Remove build profile')) {
            $buildProfile | Remove-CimInstance -ErrorAction Stop; $Result.removed.Add([string]$buildProfile.LocalPath)
        }
    }
}

function Clear-WCEventLog {
    [CmdletBinding()]
    param(
        $Options,
        $Result
    )
    if (-not $Options.log_export_path) {
        throw 'Event log export directory is required'
    }
    $destination = Assert-WCPath $Options.log_export_path
    $exportRoot = Join-Path $env:ProgramData 'PackerCleanup\Exports'
    if (-not (Test-WCWithin $destination $exportRoot -AllowRoot)) {
        throw 'Exports must be under ProgramData\PackerCleanup\Exports'
    }
    foreach ($name in @($Options.event_logs)) {
        if (-not $name -or $name -eq 'Security' -or $name -match '[\\:*?"<>|]' -or $name.Contains('..')) {
            throw 'Protected or invalid event log'
        }
    }
    if ($Options.dry_run) {
        $Result.message = 'Selected event logs would be exported then cleared';
        return
    }
    $null = New-Item -ItemType Directory -Path $destination -Force
    $null = Assert-WCPath $destination
    $tool = Join-Path $env:SystemRoot 'System32\wevtutil.exe'
    foreach ($name in @($Options.event_logs)) {
        $file = Join-Path $destination (($name.Replace('/', '_')) + '-' + [guid]::NewGuid().ToString('N') + '.evtx')
        $null = Invoke-WCNative $tool @('epl', ('"' + $name + '"'), ('"' + $file + '"'))
        if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
            throw 'Event log export was not created'
        }
        $null = Invoke-WCNative $tool @('cl', ('"' + $name + '"'))
    }
    $Result.message = 'Event logs exported to guest; retrieve exports before image publication'
}

function Clear-WCErrorReporting {
    [CmdletBinding()]
    param(
        $Options,
        $Result
    )
    if ($Options.dry_run) {
        $Result.message = 'Windows WER cleanup handler would select eligible reports';
        return
    }
    $mutex = New-Object Threading.Mutex($false, 'Global\PackerWindowsCleanupDiskCleanup')
    $locked = $false; $saved = @()
    try {
        $locked = $mutex.WaitOne(0)
        if (-not $locked) {
            throw 'Another Disk Cleanup operation is active'
        }
        if (Get-Process -Name cleanmgr -ErrorAction SilentlyContinue) {
            throw 'Disk Cleanup is already running'
        }
        $root = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VolumeCaches'
        foreach ($key in Get-ChildItem -LiteralPath $root -ErrorAction Stop) {
            $value = Get-ItemProperty -LiteralPath $key.PSPath -ErrorAction Stop
            $old = $value.PSObject.Properties['StateFlags65530']
            $saved += @{
                path   = $key.PSPath;
                exists = ($null -ne $old);
                value  = $(if ($old) {
                        $old.Value
                    }
                    else {
                        0
                    })
            }
            $enabled = if ($key.PSChildName -eq 'Windows Error Reporting Files') { 2 } else { 0 }
            $null = New-ItemProperty -LiteralPath $key.PSPath -Name StateFlags65530 -PropertyType DWord -Value $enabled -Force -ErrorAction Stop
        }
        $native = Invoke-WCNative (Join-Path $env:SystemRoot 'System32\cleanmgr.exe') @('/sagerun:65530')
        $Result.output = $native.text
        $Result.message = 'Windows WER handler completed; Windows determines eligibility; per-file removals are not exposed'
    }
    finally {
        foreach ($entry in $saved) {
            if ($entry.exists) {
                Set-ItemProperty -LiteralPath $entry.path -Name StateFlags65530 -Value $entry.value -ErrorAction Stop
            }
            else {
                Remove-ItemProperty -LiteralPath $entry.path -Name StateFlags65530 -ErrorAction Stop
            }
        }
        if ($locked) {
            $mutex.ReleaseMutex()
        };
        $mutex.Dispose()
    }
}

function Install-WCFinalizer {
    [CmdletBinding()]
    param(
        $Options,
        $Result
    )
    Test-WCSystem $Options $Result
    $os = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
    $computer = Get-CimInstance -ClassName Win32_ComputerSystem -ErrorAction Stop
    if ($os.ProductType -eq 2 -or $computer.PartOfDomain) {
        throw 'Domain controllers and domain-joined images cannot use this generalization strategy'
    }
    if (-not (Test-Path -LiteralPath (Join-Path $env:SystemRoot 'System32\Sysprep\Sysprep.exe'))) {
        throw 'Sysprep is unavailable'
    }
    $state = Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Setup\State' -ErrorAction Stop
    if ($state.ImageState -ne 'IMAGE_STATE_COMPLETE') {
        throw 'Image setup is incomplete or already generalized'
    }
    $directory = Join-Path $env:ProgramData 'PackerCleanup\Finalization'
    $null = Assert-WCPath $directory
    if (Test-Path -LiteralPath (Join-Path $directory 'attempted')) {
        throw 'Generalization was already attempted; inspect Sysprep Panther logs'
    }
    if ($Options.dry_run) {
        $Result.message = 'Sysprep prerequisites checked; no finalizer staged'; return
    }
    $null = New-Item -ItemType Directory -Path $directory -Force
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetAccessRuleProtection($true, $false)
    foreach ($sid in @('S-1-5-18', 'S-1-5-32-544')) {
        $identity = New-Object Security.Principal.SecurityIdentifier($sid)
        $acl.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule($identity, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')))
    }
    Set-Acl -LiteralPath $directory -AclObject $acl
    foreach ($name in @('WindowsCleanup.ps1', 'Invoke-WindowsCleanupFinalizer.ps1')) {
        $destination = Join-Path $directory $name
        $null = Assert-WCPath $destination
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot $name) -Destination $destination -Force -ErrorAction Stop
    }
    $Result.message = 'Finalizer staged; no generalization executed by provisioner'
}

function Invoke-WCOperation {
    [OutputType([hashtable])]
    [CmdletBinding()]
    param(
        $Options
    )
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $result = @{
        id = [string]$Options.operation; status = 'success'; message = ''; output = ''
        removed = (New-Object 'Collections.Generic.List[string]');
        candidates = (New-Object 'Collections.Generic.List[string]')
        warnings = (New-Object 'Collections.Generic.List[string]');
        deleted_file_bytes = [long]0; duration_seconds = 0.0
    }
    try {
        $disk = Get-CimInstance -ClassName Win32_LogicalDisk -Filter "DeviceID='$env:SystemDrive'" -ErrorAction Stop
        $result.free_bytes_before = [long]$disk.FreeSpace
        switch ($Options.operation) {
            { $_ -in @('temporary_files', 'installer_cleanup', 'provisioning_artifacts', 'log_cleanup') } { Invoke-WCFileCleanup $Options $result }
            'component_store' {
                Invoke-WCServicing $Options $result
            }
            'validate_system' {
                Test-WCSystem $Options $result
            }
            'remove_user_profiles' {
                Remove-WCBuildProfile $Options $result
            }
            'clear_event_logs' {
                Clear-WCEventLog $Options $result
            }
            'error_reporting' {
                Clear-WCErrorReporting $Options $result
            }
            'empty_recycle_bin' {
                if (-not $Options.dry_run) {
                    Clear-RecycleBin -DriveLetter $env:SystemDrive.Substring(0, 1) -Force -ErrorAction Stop
                };
                $result.message = 'Current communicator user recycle bin only'
            }
            'stage_sysprep' { Install-WCFinalizer $Options $result }
            default {
                throw 'Unknown cleanup operation'
            }
        }
        $disk = Get-CimInstance -ClassName Win32_LogicalDisk -Filter "DeviceID='$env:SystemDrive'" -ErrorAction Stop
        $result.free_bytes_after = [long]$disk.FreeSpace
        if ($result.status -eq 'success') {
            if ($Options.dry_run) {
                $result.status = 'dry_run'
            }
            elseif ($result.warnings.Count) {
                $result.status = 'warning'
            }
        }
    }
    catch {
        $result.status = 'failed';
        $result.message = $_.Exception.Message
    }
    $watch.Stop();
    $result.duration_seconds = $watch.Elapsed.TotalSeconds
    return $result
}
