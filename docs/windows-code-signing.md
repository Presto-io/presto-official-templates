# Windows Code Signing Policy

This document defines the SignPath application materials, public code signing policy, trust boundary, and downstream contracts for Presto-io Windows template executables.

## Code signing policy

Free code signing provided by SignPath.io, certificate by SignPath Foundation.

The signed scope is limited to Presto-io controlled official Windows template executables and future Presto-reviewed verified Windows template executables.

Presto-io does not use its publisher identity to sign arbitrary community executables. Public trusted signing is a release trust boundary for assets that Presto-io controls or has explicitly reviewed and accepted into a verified process.

## SignPath application materials

- [ ] project scope: `Presto-io Windows template executables`
- [ ] repository: `Presto-io/presto-official-templates`
- [ ] license: `MIT`
- [ ] release workflow: `presto-official-templates/.github/workflows/release.yml`
- [ ] signed objects: `presto-template-{name}-windows-amd64.exe` and `presto-template-{name}-windows-arm64.exe`
- [ ] maintainers and responsibilities: committers/reviewers/approvers
- [ ] trigger policy: protected `v*` tag/release workflow only
- [ ] publisher identity: stable SignPath Foundation-backed publisher for public signing

## Trust boundary

- `official` Windows `.exe` assets must be signed before public release.
- `verified` Windows `.exe` assets must satisfy the same Authenticode validity rules before entering verified trust.
- `community` templates are not signed with the Presto-io publisher identity and must not be labeled `signed` or `verified` by default.
- self-signed output is `dev/UAT only` and never valid for official/verified public release.

## Release signing policy

- Only protected `v*` tags or protected release workflow runs may request public SignPath signing.
- PRs, forks, ordinary branches, and local/manual commands must not request public trusted signing.
- Windows `.exe` assets must be signed and verified before `SHA256SUMS` is generated.
- SignPath pending/unavailable/failure, signing failure, verification failure, missing timestamp, invalid certificate chain, or publisher mismatch must block Windows official publication or mark it as `public trusted signing blocked`.
- unsigned official Windows `.exe` must not be published as a normal official release asset.

Phase 28 must preserve the artifact filenames and move the release sequence to:

```text
build -> SignPath signing -> verify -> sha256sum
```

`SHA256SUMS`, registry `sha256`, and CDN mirrors must all be generated from the signed bytes.

## SignPath GitHub trusted-build policy

The SignPath trusted-build policy file is `.signpath/policies/presto-io/release-signing.yml`.

Phase 28's signing request must use GitHub workflow artifact IDs rather than locally produced files. This keeps public trusted signing tied to GitHub Actions origin metadata, protected release/tag workflow execution, and the artifacts produced by the official template build matrix.

## Self-signed dev/UAT lane

The local self-signed lane is `dev/UAT only`. It proves the technical Authenticode flow, trust-root installation, signing, verification, and trust-root cleanup on Windows, but self-signed output cannot be used for public official/verified releases.

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\sign-windows-dev.ps1 -InputPath .\presto-template-gongwen-windows-amd64.exe -InstallTrustRoot
pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\sign-windows-dev.ps1 -RemoveTrustRoot
```

## Authenticode verification gate

Public release verification must use:

```powershell
Get-AuthenticodeSignature .\presto-template-gongwen-windows-amd64.exe
signtool verify /pa /tw .\presto-template-gongwen-windows-amd64.exe
```

Authenticode status valid, certificate chain valid, timestamp present, and publisher identity stable are hard gates. Missing signatures, invalid chains, missing timestamps, or publisher drift must block official/verified Windows publication instead of being downgraded to warnings.

## Downstream phase contracts

### Phase 28 release workflow

Phase 28 owns the real release workflow integration. It must insert public SignPath signing after build artifact download and before `SHA256SUMS`, fail closed for official Windows `.exe` assets, and avoid public signing requests from PR, fork, ordinary branch, or local/manual execution paths.

### Phase 29 registry and Presto trust chain

Phase 29 owns registry metadata and Presto install gating. Future Windows platform metadata must include `signature.status`, `signature.provider`, `signature.publisher`, `signature.timestamped`, and `signature.verifiedAt`.

The registry and Presto install chain must treat `official` and `verified` Windows `.exe` assets the same for signature validity: signed status, expected provider, stable publisher identity, timestamp, and verification time must be present before a Windows asset is considered signed.

### Phase 30 Windows UAT

Phase 30 owns runtime proof on Windows. It must verify the public trusted signing result with Authenticode tools and record `public trusted signing blocked` if SignPath approval is still pending or unavailable. Self-signed testing can prove the technical lane, but it cannot prove public trust for normal users.
