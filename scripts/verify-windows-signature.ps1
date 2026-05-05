[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string[]] $InputPath,
  [string[]] $ExpectedPublisher = @('CN=Presto-io Dev UAT Code Signing'),
  [string[]] $ExpectedPublisherThumbprint = @(),
  [switch] $RequireTimestamp
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$isWindowsVariable = Get-Variable -Name IsWindows -ValueOnly -ErrorAction SilentlyContinue
$isDesktopWindowsPowerShell = $PSVersionTable.PSEdition -eq 'Desktop'
if (-not ($isDesktopWindowsPowerShell -or $isWindowsVariable)) {
  throw 'This script requires Windows Authenticode APIs.'
}

$signtool = Get-Command signtool -ErrorAction SilentlyContinue
$signtoolCommandText = 'signtool verify /pa /tw'

foreach ($path in $InputPath) {
  $resolved = Resolve-Path -LiteralPath $path -ErrorAction Stop
  if ([System.IO.Path]::GetExtension($resolved.Path) -ne '.exe') {
    throw "InputPath must be a Windows .exe file: $path"
  }

  $signature = Get-AuthenticodeSignature -FilePath $resolved.Path
  if ($signature.Status -ne 'Valid') {
    throw "Authenticode signature status is not Valid: $($resolved.Path) status=$($signature.Status)"
  }

  if (-not $signature.SignerCertificate) {
    throw "Authenticode signature has no signer certificate: $($resolved.Path)"
  }

  $signerSubject = $signature.SignerCertificate.Subject
  if ($ExpectedPublisher.Count -gt 0 -and ($ExpectedPublisher -notcontains $signerSubject)) {
    throw "Publisher mismatch: $($resolved.Path) signer='$signerSubject' expected='$($ExpectedPublisher -join ', ')'"
  }

  $signerThumbprint = $signature.SignerCertificate.Thumbprint
  if ($ExpectedPublisherThumbprint.Count -gt 0 -and ($ExpectedPublisherThumbprint -notcontains $signerThumbprint)) {
    throw "Publisher certificate mismatch: $($resolved.Path) thumbprint='$signerThumbprint'"
  }

  $timestampCertificate = $null
  if ($signature.PSObject.Properties.Name -contains 'TimeStamperCertificate') {
    $timestampCertificate = $signature.TimeStamperCertificate
  }
  $timestampObserved = $null -ne $timestampCertificate

  if ($RequireTimestamp -and -not $timestampObserved -and -not $signtool) {
    throw "Timestamp metadata is absent and signtool is not available for timestamp verification: $($resolved.Path)"
  }

  if ($signtool) {
    & $signtool.Source verify /pa /tw $resolved.Path
    $signtoolExitCode = $LASTEXITCODE
    if ($signtoolExitCode -ne 0) {
      throw "$signtoolCommandText failed for $($resolved.Path) with exit code $signtoolExitCode"
    }

    if ($RequireTimestamp) {
      $timestampObserved = $true
    }
  }

  if ($RequireTimestamp -and -not $timestampObserved) {
    throw "Timestamp metadata is absent: $($resolved.Path)"
  }

  Write-Output ("signature valid path='{0}' signer='{1}' timestampObserved={2}" -f $resolved.Path, $signerSubject, $timestampObserved)
}
