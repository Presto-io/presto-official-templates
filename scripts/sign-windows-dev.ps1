# dev/UAT only: this self-signed lane proves the technical Authenticode flow and must never be used for public official/verified releases.
[CmdletBinding()]
param(
  [Parameter(ValueFromPipeline = $true)]
  [string[]] $InputPath,
  [string] $Subject = 'CN=Presto-io Dev UAT Code Signing',
  [switch] $InstallTrustRoot,
  [switch] $RemoveTrustRoot
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

begin {
  $isWindowsVariable = Get-Variable -Name IsWindows -ValueOnly -ErrorAction SilentlyContinue
  $isDesktopWindowsPowerShell = $PSVersionTable.PSEdition -eq 'Desktop'
  if (-not ($isDesktopWindowsPowerShell -or $isWindowsVariable)) {
    throw 'This script requires Windows certificate stores.'
  }

  $certificateStores = @(
    'Cert:\CurrentUser\Root',
    'Cert:\CurrentUser\TrustedPublisher'
  )

  function Remove-DevTrustRoot {
    param([string] $CertificateSubject)

    foreach ($store in $certificateStores) {
      Get-ChildItem -Path $store |
        Where-Object { $_.Subject -eq $CertificateSubject } |
        Remove-Item -Force
    }
  }

  if ($RemoveTrustRoot) {
    Remove-DevTrustRoot -CertificateSubject $Subject
    Write-Output "dev/UAT only trust roots removed for $Subject"
    return
  }

  $paths = New-Object System.Collections.Generic.List[string]
}

process {
  if ($InputPath) {
    foreach ($path in $InputPath) {
      $paths.Add($path)
    }
  }
}

end {
  if ($RemoveTrustRoot) {
    return
  }

  if ($paths.Count -eq 0) {
    throw 'At least one -InputPath .exe file is required unless -RemoveTrustRoot is used.'
  }

  $cert = Get-ChildItem -Path 'Cert:\CurrentUser\My' -CodeSigningCert |
    Where-Object { $_.Subject -eq $Subject -and $_.NotAfter -gt (Get-Date) } |
    Sort-Object NotAfter -Descending |
    Select-Object -First 1

  if (-not $cert) {
    $cert = New-SelfSignedCertificate `
      -Type CodeSigningCert `
      -Subject $Subject `
      -CertStoreLocation 'Cert:\CurrentUser\My' `
      -KeyAlgorithm RSA `
      -KeyLength 3072 `
      -HashAlgorithm sha256 `
      -NotAfter (Get-Date).AddYears(1)
  }

  if ($InstallTrustRoot) {
    $tempCert = Join-Path ([System.IO.Path]::GetTempPath()) ('presto-io-dev-uat-code-signing-{0}.cer' -f $cert.Thumbprint)
    try {
      Export-Certificate -Cert $cert -FilePath $tempCert | Out-Null
      foreach ($store in $certificateStores) {
        Import-Certificate -FilePath $tempCert -CertStoreLocation $store | Out-Null
      }
    } finally {
      if (Test-Path -LiteralPath $tempCert) {
        Remove-Item -LiteralPath $tempCert -Force
      }
    }
  }

  foreach ($path in $paths) {
    $resolved = Resolve-Path -LiteralPath $path -ErrorAction Stop
    if ([System.IO.Path]::GetExtension($resolved.Path) -ne '.exe') {
      throw "InputPath must be a Windows .exe file: $path"
    }

    Set-AuthenticodeSignature -FilePath $resolved.Path -Certificate $cert -HashAlgorithm SHA256 | Out-Null

    $signature = Get-AuthenticodeSignature -FilePath $resolved.Path
    if ($InstallTrustRoot -and $signature.Status -ne 'Valid') {
      throw "Authenticode signature is not Valid after dev/UAT signing: $($resolved.Path) status=$($signature.Status)"
    }
  }

  Write-Output 'dev/UAT only signing complete'
}
