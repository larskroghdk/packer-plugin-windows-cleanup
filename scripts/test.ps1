#Requires -Version 5.1

<#
.SYNOPSIS
    Runs deterministic local checks without contacting a Windows guest.

.NOTES
    Author: Lars Krogh
    Version: 0.2.0
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

function Invoke-Check {
    param(
        [string]$Command,
        [string[]]$Arguments
    )
    Write-Output ("Checking: " + $Command + ' ' + ($Arguments -join ' '))
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE"
    }
}

if (@(& gofmt -l main.go main_test.go provisioner guest internal test/integration).Count) {
    throw 'Run gofmt'
}

Invoke-Check -Command go -Arguments @('mod', 'tidy', '-diff')
Invoke-Check -Command go -Arguments @('build', './...')
Invoke-Check -Command go -Arguments @('vet', './...')
Invoke-Check -Command go -Arguments @('vet', '-tags', 'integration', './test/integration')
Invoke-Check -Command staticcheck -Arguments @('./...')
Invoke-Check -Command govulncheck -Arguments @('./...')
Invoke-Check -Command go -Arguments @('test', './...')

$findings = @(Invoke-ScriptAnalyzer -Path guest/scripts -Recurse -Severity Warning, Error)
if ($findings.Count) {
    $findings | Format-Table -Wrap;
    throw 'PSScriptAnalyzer findings'
}

Invoke-Check -Command packer -Arguments @('fmt', '-check', '-recursive', 'examples')
Invoke-Check -Command packer -Arguments @('fmt', '-check', '-recursive', 'test/integration')
Invoke-Check -Command powershell.exe -Arguments @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command', 'Import-Module Pester -MinimumVersion 5.0; Invoke-Pester -Path test/pester -Output Detailed -CI')
