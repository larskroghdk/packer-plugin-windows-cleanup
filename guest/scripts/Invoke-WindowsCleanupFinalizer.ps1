#Requires -Version 5.1

<#
.SYNOPSIS
    Launches Sysprep only from the QEMU builder's shutdown command.

.DESCRIPTION
    Revalidates prerequisites and records an exclusive attempt marker, then starts Sysprep asynchronously so the communicator returns before generalization.
    QEMU must wait for power-off and fail if shutdown times out.
    Panther logs and the attempt marker remain available in a failed image for investigation.

.NOTES
    Author: Lars Krogh
    Version: 0.1.0
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

Set-StrictMode -Version Latest

try {
    . (Join-Path $PSScriptRoot 'WindowsCleanup.ps1')
    $options = @{
        required_services       = @();
        required_software_paths = @();
        required_features       = @()
    }
    $result = @{}
    Test-WCSystem $options $result
    $computer = Get-CimInstance -ClassName Win32_ComputerSystem -ErrorAction Stop
    $os = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop
    if ($computer.PartOfDomain -or $os.ProductType -eq 2) {
        throw 'Unsupported domain generalization scenario'
    }
    $state = Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Setup\State' -ErrorAction Stop
    if ($state.ImageState -ne 'IMAGE_STATE_COMPLETE') {
        throw 'Image is not ready to generalize'
    }
    if (Get-Command Get-AppxPackage -ErrorAction SilentlyContinue) {
        $provisioned = @(Get-AppxProvisionedPackage -Online -ErrorAction Stop | Select-Object -ExpandProperty PackageName)
        foreach ($app in Get-AppxPackage -AllUsers -ErrorAction Stop) {
            if (-not $app.NonRemovable -and $app.PackageFullName -notin $provisioned) {
                throw 'Unprovisioned user Appx package detected; resolve before generalization'
            }
        }
    }
    $marker = Join-Path $PSScriptRoot 'attempted'
    $null = Assert-WCPath $marker
    $handle = [IO.File]::Open($marker, 'CreateNew', 'Write', 'None'); $handle.Dispose()
    $sysprep = Join-Path $env:SystemRoot 'System32\Sysprep\Sysprep.exe'
    $process = Start-Process -FilePath $sysprep -ArgumentList @('/generalize', '/oobe', '/quiet', '/shutdown') -WindowStyle Hidden -PassThru
    $process.Dispose()
}
catch {
    Write-Error -Message $_.Exception.Message -ErrorAction Continue; exit 1
}
