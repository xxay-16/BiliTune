[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9A-Za-z][0-9A-Za-z._+-]*$')]
    [string]$Version,
    [string]$SourceDir,
    [string]$OutputDir,
    [switch]$SkipInstaller
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
if (-not $SourceDir) {
    $SourceDir = Join-Path $projectRoot 'build/windows/x64/runner/Release'
}
if (-not $OutputDir) {
    $OutputDir = Join-Path $projectRoot 'dist/windows'
}
$SourceDir = (Resolve-Path -LiteralPath $SourceDir).Path
$OutputDir = [System.IO.Path]::GetFullPath($OutputDir)
$packageName = "BiliTune-$Version-windows-x64"
$stageDir = Join-Path $OutputDir ".staging/$packageName"
$zipPath = Join-Path $OutputDir "$packageName-portable.zip"
$installerPath = Join-Path $OutputDir "BiliTune-Setup-$Version.exe"

foreach ($file in @(
    'bilitune.exe',
    'flutter_windows.dll',
    'libmpv-2.dll',
    'WebView2Loader.dll',
    'data/icudtl.dat',
    'data/app.so',
    'data/flutter_assets/AssetManifest.bin'
)) {
    if (-not (Test-Path -LiteralPath (Join-Path $SourceDir $file) -PathType Leaf)) {
        throw "Missing Windows build output: $file"
    }
}
foreach ($output in @($stageDir, $zipPath, $installerPath)) {
    if (Test-Path -LiteralPath $output) {
        throw "Package output already exists: $output. Use a fresh output directory."
    }
}

# Bundle the redistributable x64 runtime from the installed C++ toolchain.
$vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
if (-not (Test-Path -LiteralPath $vswhere)) {
    throw 'Visual Studio Installer/vswhere.exe was not found.'
}
$visualStudio = & $vswhere -latest -products '*' `
    -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
if ($LASTEXITCODE -ne 0 -or -not $visualStudio) {
    throw 'No Visual Studio C++ toolchain was found.'
}
$redistRoot = Join-Path $visualStudio 'VC/Redist/MSVC'
$runtime = $null
foreach ($toolset in (Get-ChildItem -LiteralPath $redistRoot -Directory |
    Where-Object { $_.Name -match '^\d+\.\d+\.\d+(\.\d+)?$' } |
    Sort-Object { [version]$_.Name } -Descending)) {
    $runtime = Get-ChildItem -LiteralPath (Join-Path $toolset.FullName 'x64') `
        -Directory -Filter 'Microsoft.VC*.CRT' -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if ($runtime) { break }
}
if (-not $runtime) {
    throw 'The x64 Visual C++ redistributable runtime was not found.'
}

$compiler = $null
if (-not $SkipInstaller) {
    $command = Get-Command ISCC.exe -ErrorAction SilentlyContinue
    if ($command) { $compiler = $command.Source }
    foreach ($programRoot in @(${env:ProgramFiles(x86)}, $env:ProgramFiles)) {
        if ($compiler) { break }
        foreach ($setupVersion in @('7', '6')) {
            $candidate = Join-Path $programRoot "Inno Setup $setupVersion/ISCC.exe"
            if (Test-Path -LiteralPath $candidate) {
                $compiler = $candidate
                break
            }
        }
    }
    if (-not $compiler) {
        throw 'Inno Setup compiler not found. Install Inno Setup or use -SkipInstaller.'
    }
}

New-Item -ItemType Directory -Path $stageDir -Force | Out-Null
Copy-Item -Path (Join-Path $SourceDir '*') -Destination $stageDir -Recurse
Copy-Item -Path (Join-Path $runtime.FullName '*.dll') -Destination $stageDir
@"
BiliTune Windows x64 ($Version)

完整解压后运行 bilitune.exe，请保留同目录中的 DLL 文件和 data 文件夹。
启动前请先从系统托盘退出已运行的旧版 BiliTune。
"@ | Set-Content -LiteralPath (Join-Path $stageDir 'README.txt') -Encoding utf8

if (-not $SkipInstaller) {
    & $compiler (Join-Path $projectRoot 'windows/installer/bilitune.iss') `
        "/DAppVersion=$Version" "/DSourceDir=$stageDir" "/DOutputDir=$OutputDir"
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $installerPath)) {
        throw 'Inno Setup did not produce the Windows installer.'
    }
}

Compress-Archive -LiteralPath $stageDir -DestinationPath $zipPath -CompressionLevel Optimal
$archive = [System.IO.Compression.ZipFile]::OpenRead($zipPath)
try {
    if (-not $archive.GetEntry("$packageName/bilitune.exe")) {
        throw 'Portable ZIP is missing bilitune.exe.'
    }
    if (-not $archive.GetEntry("$packageName/data/app.so")) {
        throw 'Portable ZIP is missing the compiled application.'
    }
} finally {
    $archive.Dispose()
}

$packages = @($zipPath)
if (-not $SkipInstaller) { $packages += $installerPath }
$checksums = foreach ($package in $packages) {
    $hash = Get-FileHash -LiteralPath $package -Algorithm SHA256
    "$($hash.Hash.ToLowerInvariant())  $([System.IO.Path]::GetFileName($package))"
}
$checksums | Set-Content -LiteralPath (Join-Path $OutputDir 'SHA256SUMS.txt') -Encoding utf8
Get-Item -LiteralPath $packages | Select-Object FullName, Length
