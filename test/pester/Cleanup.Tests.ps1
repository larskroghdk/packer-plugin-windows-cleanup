#Requires -Version 5.1
<#
.SYNOPSIS
    Tests cleanup in isolated TestDrive paths and mocks servicing APIs.
.NOTES
    Author: Lars Krogh
    Version: 0.1.0
#>
BeforeAll {
    . (Join-Path $PSScriptRoot '..\..\guest\scripts\WindowsCleanup.ps1')
    function New-TestOptions {
        return @{ dry_run = $false; minimum_age_hours = 24; exclusions = @(); reset_component_base = $false; operation = 'temporary_files'; cleanup_paths = @(); installer_paths = @(); artifact_paths = @(); log_paths = @(); log_export_path = ''; build_profile_sids = @(); required_services = @(); required_features = @(); required_software_paths = @() }
    }
    function New-TestResult {
        return @{ status = 'success'; message = ''; output = ''; removed = (New-Object 'Collections.Generic.List[string]'); candidates = (New-Object 'Collections.Generic.List[string]'); warnings = (New-Object 'Collections.Generic.List[string]'); deleted_file_bytes = [long]0 }
    }
}

Describe 'Path protection' {
    It 'rejects traversal, device paths, alternate streams and ambiguous normalization' {
        foreach ($path in @('C:\Temp\..\Windows', '\\server\share\file', '\\?\C:\Temp', 'C:\Temp\file:secret', 'C:\Temp\NUL.txt', 'C:\Temp\file.', 'C:\Temp\\file', 'C:\Temp\*')) {
            { Assert-WCPath $path } | Should -Throw
        }
    }
    It 'uses complete path boundaries' {
        Test-WCWithin 'C:\Windows\Temporary\x' 'C:\Windows\Temp' | Should -BeFalse
        Test-WCWithin 'C:\Windows\Temp\x' 'C:\Windows\Temp' | Should -BeTrue
        Test-WCWithin 'C:\Windows\Temp' 'C:\Windows\Temp' | Should -BeFalse
    }
    It 'rejects a junction before traversing it' {
        $outside = Join-Path $TestDrive 'outside'; $root = Join-Path $TestDrive 'root'
        $null = New-Item -ItemType Directory -Path $outside, $root
        $target = Join-Path $outside 'keep.txt'; Set-Content -LiteralPath $target -Value 'keep'
        $link = Join-Path $root 'link'; $null = New-Item -ItemType Junction -Path $link -Target $outside
        { Assert-WCTarget (Join-Path $link 'keep.txt') @($root) } | Should -Throw
        Test-Path -LiteralPath $target | Should -BeTrue
    }
    It 'rejects a sibling prefix and system directories' {
        { Assert-WCTarget 'C:\Windows\Installer' @('C:\Windows\Temp') } | Should -Throw
        { Assert-WCTarget 'C:\Windows\Temp2' @('C:\Windows\Temp') } | Should -Throw
    }
}

Describe 'Guarded file removal' {
    BeforeEach {
        $root = Join-Path $TestDrive ([guid]::NewGuid().ToString('N'))
        $null = New-Item -ItemType Directory -Path $root
        $stale = Join-Path $root 'old.txt'; [IO.File]::WriteAllText($stale, 'abc')
        (Get-Item -LiteralPath $stale).LastWriteTimeUtc = [DateTime]::UtcNow.AddDays(-3)
        $options = New-TestOptions; $result = New-TestResult
    }
    It 'removes stale files, preserves new files and is idempotent' {
        $new = Join-Path $root 'fresh.txt'; Set-Content -LiteralPath $new -Value 'new'
        Remove-WCItem -Path $root -Roots @($root) -Options $options -Result $result
        Test-Path -LiteralPath $stale | Should -BeFalse
        Test-Path -LiteralPath $new | Should -BeTrue
        $result.deleted_file_bytes | Should -Be 3
        $second = New-TestResult
        Remove-WCItem -Path $root -Roots @($root) -Options $options -Result $second
        $second.removed.Count | Should -Be 0
    }
    It 'never mutates files in dry run' {
        $options.dry_run = $true
        Remove-WCItem -Path $root -Roots @($root) -Options $options -Result $result
        Test-Path -LiteralPath $stale | Should -BeTrue
        $result.candidates.Count | Should -Be 1
        $result.removed.Count | Should -Be 0
        $result.deleted_file_bytes | Should -Be 0
    }
    It 'honors literal subtree exclusions' {
        $options.exclusions = @($stale)
        Remove-WCItem -Path $root -Roots @($root) -Options $options -Result $result
        Test-Path -LiteralPath $stale | Should -BeTrue
        $result.candidates.Count | Should -Be 0
    }
    It 'removes stale read-only files and keeps processing the root' {
        $readOnly = Join-Path $root 'a-readonly.txt'; [IO.File]::WriteAllText($readOnly, 'ro')
        $item = Get-Item -LiteralPath $readOnly
        $item.LastWriteTimeUtc = [DateTime]::UtcNow.AddDays(-3)
        $item.IsReadOnly = $true
        Remove-WCItem -Path $root -Roots @($root) -Options $options -Result $result
        Test-Path -LiteralPath $readOnly | Should -BeFalse
        Test-Path -LiteralPath $stale | Should -BeFalse
        $result.warnings.Count | Should -Be 0
    }
    It 'skips locked files with a warning' {
        $handle = [IO.File]::Open($stale, 'Open', 'ReadWrite', 'None')
        try { Remove-WCItem -Path $root -Roots @($root) -Options $options -Result $result }
        finally { $handle.Dispose() }
        Test-Path -LiteralPath $stale | Should -BeTrue
        $result.warnings.Count | Should -Be 1
    }
    It 'processes explicit installer and artifact manifests despite recent timestamps' {
        foreach ($category in @('installer_cleanup', 'provisioning_artifacts')) {
            $file = Join-Path $root ($category + '.tmp'); Set-Content -LiteralPath $file -Value 'staging'
            Remove-WCItem -Path $file -Roots @($root) -Options $options -Result $result -Manifest
            Test-Path -LiteralPath $file | Should -BeFalse
        }
    }
    It 'validates all manifest paths before deleting any of them' {
        Mock Get-WCRoot { @($root) }
        $options.operation = 'installer_cleanup'; $options.installer_paths = @($stale, 'C:\Windows\Installer')
        { Invoke-WCFileCleanup $options $result } | Should -Throw
        Test-Path -LiteralPath $stale | Should -BeTrue
    }
}

Describe 'Servicing and final validation' {
    BeforeEach {
        $options = New-TestOptions; $result = New-TestResult
        Mock Get-WCFact { @{ pending_reboot = $false; update_busy = $false; winrm = $true } }
        Mock Invoke-WCNative { @{ exit_code = 0; text = 'Measured DISM analysis' } }
    }
    It 'uses supported DISM commands and leaves ResetBase opt in' {
        Invoke-WCServicing $options $result
        Should -Invoke Invoke-WCNative -Times 1 -ParameterFilter { $Arguments -contains '/StartComponentCleanup' -and $Arguments -notcontains '/ResetBase' }
        Should -Invoke Invoke-WCNative -Times 2 -ParameterFilter { $Arguments -contains '/AnalyzeComponentStore' }
    }
    It 'passes ResetBase only when explicitly selected' {
        $options.reset_component_base = $true; Invoke-WCServicing $options $result
        Should -Invoke Invoke-WCNative -Times 1 -ParameterFilter { $Arguments -contains '/ResetBase' }
    }
    It 'performs analysis only in dry run' {
        $options.dry_run = $true; Invoke-WCServicing $options $result
        Should -Invoke Invoke-WCNative -Times 0 -ParameterFilter { $Arguments -contains '/StartComponentCleanup' }
    }
    It 'refuses servicing during pending reboot or active updates' {
        Mock Get-WCFact { @{ pending_reboot = $true; update_busy = $false } }
        { Invoke-WCServicing $options $result } | Should -Throw
        Should -Invoke Invoke-WCNative -Times 0
    }
    It 'does not mistake a successful health query for a healthy image' {
        Mock Get-Service { [pscustomobject]@{ Status = 'Running' } }
        Mock Repair-WindowsImage { [pscustomobject]@{ ImageHealthState = 'Repairable'; RestartNeeded = $false } }
        { Test-WCSystem $options $result } | Should -Throw
    }
    It 'rejects loaded and special profiles' {
        Mock Get-CimInstance { [pscustomobject]@{ SID = 'S-1-5-21-1-2-3-1001'; Special = $true; Loaded = $false; LocalPath = 'C:\Users\packer-build-test' } }
        $options.build_profile_sids = @('S-1-5-21-1-2-3-1001')
        { Remove-WCBuildProfile $options $result } | Should -Throw
    }
    It 'refuses to clear security audit logs' {
        $options.event_logs = @('Security'); $options.log_export_path = 'C:\ProgramData\PackerCleanup\Exports'
        { Clear-WCEventLog $options $result } | Should -Throw
    }
}

Describe 'Pending reboot detection' {
    BeforeEach {
        Mock Test-Path { $false }
        Mock Get-ItemProperty { [pscustomobject]@{} }
        Mock Get-WindowsPackage { @() }
        Mock Get-WindowsOptionalFeature { @() }
    }
    It 'detects registry and file rename requirements' {
        Mock Test-Path { $true } -ParameterFilter { $LiteralPath -like '*RebootPending' }
        Mock Get-ItemProperty { [pscustomobject]@{ PendingFileRenameOperations = @('old', 'new') } }
        $reasons = @(Get-WCPendingRebootReason)
        $reasons.Count | Should -Be 2
        $reasons | Should -Contain 'PendingFileRenameOperations'
    }
    It 'detects pending packages and feature changes' {
        Mock Get-WindowsPackage { [pscustomobject]@{ PackageState = 'InstallPending' } }
        Mock Get-WindowsOptionalFeature { [pscustomobject]@{ State = 'EnablePending' } }
        $reasons = @(Get-WCPendingRebootReason)
        $reasons | Should -Contain 'Pending DISM packages'
        $reasons | Should -Contain 'Pending DISM features'
    }
    It 'reports no reboot only when the probes are clean' {
        @(Get-WCPendingRebootReason).Count | Should -Be 0
    }
}

Describe 'Native command handling without servicing mocks' {
    It 'captures successful output and Windows reboot-required exit codes' {
        $success = Invoke-WCNative -FilePath 'powershell.exe' -Arguments @('-NoProfile', '-Command', 'Write-Output native-ok; exit 0')
        $success.exit_code | Should -Be 0
        $success.text.Trim() | Should -Be 'native-ok'
        $reboot = Invoke-WCNative -FilePath 'powershell.exe' -Arguments @('-NoProfile', '-Command', 'exit 3010')
        $reboot.exit_code | Should -Be 3010
    }
    It 'uses real exit codes and bounds native execution' {
        # These execute only harmless child PowerShell processes on the test host.
        { Invoke-WCNative -FilePath 'powershell.exe' -Arguments @('-NoProfile', '-Command', 'exit 7') } | Should -Throw '*exit code 7*'
        { Invoke-WCNative -FilePath 'powershell.exe' -Arguments @('-NoProfile', '-Command', 'Start-Sleep 10') -TimeoutSeconds 1 } | Should -Throw '*timed out*'
    }
}
