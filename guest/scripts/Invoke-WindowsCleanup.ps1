#Requires -Version 5.1

<#
.SYNOPSIS
    Exchanges JSON with the Packer provisioner.

.DESCRIPTION
    Runs one detection or cleanup action. Operation failures are returned in JSON;
    protocol failures exit nonzero. This entry point never reboots or shuts down.

.NOTES
    Author: Lars Krogh
    Version: 0.1.0
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidateSet(
        'Detect',
        'Run'
    )]
    [string]$Action,
    [Parameter(Mandatory)]
    [string]$InputPath,
    [Parameter(Mandatory)]
    [string]$OutputPath
)

$ErrorActionPreference = 'Stop'

Set-StrictMode -Version Latest

try {
    . (Join-Path $PSScriptRoot 'WindowsCleanup.ps1')
    if ($Action -eq 'Detect') {
        $result = Get-WCFact
    }
    else {
        $options = Get-Content -LiteralPath $InputPath -Raw | ConvertFrom-Json; $result = Invoke-WCOperation $options
    }
    $json = ConvertTo-Json -InputObject $result -Depth 20
    [IO.File]::WriteAllText($OutputPath, $json, (New-Object Text.UTF8Encoding($false)))
}
catch {
    Write-Error -Message $_.Exception.Message -ErrorAction Continue; exit 1
}
