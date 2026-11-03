[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9A-Za-z][0-9A-Za-z._+-]*$')]
    [string]$Version,
    [string]$PackageDir,
    [string]$BuildDir,
    [ValidatePattern('^[0-9a-fA-F]{40}$')]
    [string]$ExpectedCommit,
    [switch]$SkipInstaller
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
if (-not $PackageDir) { $PackageDir = Join-Path $projectRoot 'dist/windows' }
if (-not $BuildDir) { $BuildDir = Join-Path $projectRoot 'bin' }
$PackageDir = (Resolve-Path -LiteralPath $PackageDir).Path
$BuildDir = (Resolve-Path -LiteralPath $BuildDir).Path
$packageName = "BiliTune-$Version-windows-x64"
$zipPath = Join-Path $PackageDir "$packageName-portable.zip"
$installerPath = Join-Path $PackageDir "BiliTune-Setup-$Version.exe"
$checksumPath = Join-Path $PackageDir 'SHA256SUMS.txt'

function Assert-PEFile([string]$Path) {
    $inputFile = [IO.File]::OpenRead($Path)
    try {
        if ($inputFile.Length -eq 0 -or $inputFile.ReadByte() -ne 0x4d -or $inputFile.ReadByte() -ne 0x5a) {
            throw "Missing or invalid Windows executable: $Path"
        }
    } finally { $inputFile.Dispose() }
}

$checksums = @{}
foreach ($line in (Get-Content -LiteralPath $checksumPath)) {
    if (-not $line.Trim()) { continue }
    if ($line -notmatch '^([a-fA-F0-9]{64})\s+(.+)$') { throw 'Invalid SHA256SUMS.txt entry.' }
    $digest = $Matches[1].ToLowerInvariant()
    $filename = $Matches[2].Trim()
    if ([IO.Path]::GetFileName($filename) -ne $filename -or $checksums.ContainsKey($filename)) {
        throw "Invalid or duplicate checksum filename: $filename"
    }
    $checksums[$filename] = $digest
}

$packages = @($zipPath)
if (-not $SkipInstaller) { $packages += $installerPath }
foreach ($file in $packages) {
    $filename = [IO.Path]::GetFileName($file)
    if (-not (Test-Path -LiteralPath $file -PathType Leaf) -or (Get-Item -LiteralPath $file).Length -eq 0) {
        throw "Missing or empty package: $filename"
    }
    $actual = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
    if (-not $checksums.ContainsKey($filename) -or $actual -ne $checksums[$filename]) {
        throw "Package checksum mismatch: $filename"
    }
}
if (-not $SkipInstaller) { Assert-PEFile $installerPath }

$binary = Join-Path $BuildDir 'bilitune.exe'
if ($ExpectedCommit) {
    $buildInfo = (& go version -m $binary) -join "`n"
    if ($LASTEXITCODE -ne 0 -or $buildInfo -notmatch "(?m)^\s*build\s+vcs\.revision=$ExpectedCommit\s*$" -or
        $buildInfo -notmatch '(?m)^\s*build\s+vcs\.modified=false\s*$') {
        throw 'The desktop binary does not match the expected clean Git revision.'
    }
}

Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [IO.Compression.ZipFile]::OpenRead($zipPath)
try {
    foreach ($filename in @('bilitune.exe', 'libmpv-2.dll')) {
        $source = Join-Path $BuildDir $filename
        Assert-PEFile $source
        $entry = $archive.GetEntry("$packageName/$filename")
        if (-not $entry -or $entry.Length -eq 0 -or $entry.Length -ne (Get-Item -LiteralPath $source).Length) {
            throw "Portable ZIP is missing the complete $filename."
        }
        $entryStream = $entry.Open()
        $hasher = [Security.Cryptography.SHA256]::Create()
        try {
            $entryHash = [BitConverter]::ToString($hasher.ComputeHash($entryStream)).Replace('-', '').ToLowerInvariant()
        } finally { $entryStream.Dispose(); $hasher.Dispose() }
        if ($entryHash -ne (Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash.ToLowerInvariant()) {
            throw "Portable ZIP contains a damaged or outdated $filename."
        }
    }
    foreach ($filename in @('README.txt', 'THIRD_PARTY_NOTICES.md')) {
        if (-not $archive.GetEntry("$packageName/$filename")) { throw "Portable ZIP is missing $filename." }
    }
    $entryCount = $archive.Entries.Count
} finally { $archive.Dispose() }

$zipBytes = (Get-Item -LiteralPath $zipPath).Length
$installerBytes = 0
if (-not $SkipInstaller) { $installerBytes = (Get-Item -LiteralPath $installerPath).Length }
if ($env:GITHUB_OUTPUT) {
    @(
        "portable_bytes=$zipBytes"
        "installer_bytes=$installerBytes"
        "entry_count=$entryCount"
        "portable_sha256=$($checksums[[IO.Path]::GetFileName($zipPath)])"
    ) | Add-Content -LiteralPath $env:GITHUB_OUTPUT -Encoding utf8
}
[PSCustomObject]@{
    Version = $Version
    Commit = $ExpectedCommit
    PortableBytes = $zipBytes
    InstallerBytes = $installerBytes
    ArchiveEntries = $entryCount
    PortableSHA256 = $checksums[[IO.Path]::GetFileName($zipPath)]
    Verified = $true
} | ConvertTo-Json
