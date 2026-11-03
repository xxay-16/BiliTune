[CmdletBinding()]
param(
    [ValidatePattern('^[0-9A-Za-z][0-9A-Za-z._+-]*$')][string]$Version,
    [string]$SourceDir,
    [string]$OutputDir,
    [switch]$SkipInstaller
)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
if (-not $SourceDir) { $SourceDir = Join-Path $projectRoot 'bin' }
if (-not $OutputDir) { $OutputDir = Join-Path $projectRoot 'dist/windows' }
if (-not $Version) {
    $source = Get-Content (Join-Path $projectRoot 'internal/model/model.go') -Raw
    if ($source -notmatch 'const Version = "([^"]+)"') { throw 'Cannot read app version.' }
    $Version = $Matches[1]
}
$SourceDir = (Resolve-Path -LiteralPath $SourceDir).Path
$OutputDir = [IO.Path]::GetFullPath($OutputDir)
$name = "BiliTune-$Version-windows-x64"
$stage = Join-Path $OutputDir ".staging/$name"
$zip = Join-Path $OutputDir "$name-portable.zip"
$installer = Join-Path $OutputDir "BiliTune-Setup-$Version.exe"
foreach ($file in @('bilitune.exe', 'libmpv-2.dll')) {
    if (-not (Test-Path -LiteralPath (Join-Path $SourceDir $file))) { throw "Missing $file. Run scripts/build-windows.ps1 first." }
}
foreach ($path in @($stage,$zip,$installer)) {
    if (Test-Path -LiteralPath $path) { throw "Output already exists: $path. Use a fresh version or output directory." }
}
New-Item -ItemType Directory -Path $stage -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $SourceDir 'bilitune.exe'),(Join-Path $SourceDir 'libmpv-2.dll') -Destination $stage
Copy-Item -LiteralPath (Join-Path $projectRoot 'THIRD_PARTY_NOTICES.md') -Destination $stage
Copy-Item -LiteralPath (Join-Path $projectRoot 'licenses') -Destination $stage -Recurse
@"
BiliTune Desktop $Version (Windows x64)
Go + Wails 3.0.0

完整解压后运行 bilitune.exe，并保留同目录的 libmpv-2.dll。
需要 Windows 10/11 与 Microsoft Edge WebView2 Runtime。
请先从托盘退出旧版 BiliTune，以免两个播放器同时响应媒体键。
首次启动会自动导入旧版登录、收藏夹显示设置、历史和下载记录。
"@ | Set-Content -LiteralPath (Join-Path $stage 'README.txt') -Encoding utf8
if (-not $SkipInstaller) {
    $compiler = (Get-Command ISCC.exe -ErrorAction SilentlyContinue).Source
    foreach ($root in @(${env:ProgramFiles(x86)}, $env:ProgramFiles, $env:LOCALAPPDATA)) {
        foreach ($relative in @('Inno Setup 7/ISCC.exe','Inno Setup 6/ISCC.exe','Programs/Inno Setup 7/ISCC.exe','Programs/Inno Setup 6/ISCC.exe')) {
            $candidate=Join-Path $root $relative
            if (-not $compiler -and (Test-Path -LiteralPath $candidate)) { $compiler=$candidate }
        }
    }
    if (-not $compiler) { throw 'Install Inno Setup 6/7 or specify -SkipInstaller.' }
    & $compiler (Join-Path $projectRoot 'packaging/windows/bilitune.iss') "/DAppVersion=$Version" "/DSourceDir=$stage" "/DOutputDir=$OutputDir"
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $installer)) { throw 'Installer compilation failed.' }
}
Compress-Archive -LiteralPath $stage -DestinationPath $zip -CompressionLevel Optimal
$archive=[IO.Compression.ZipFile]::OpenRead($zip)
try { foreach($file in @('bilitune.exe','libmpv-2.dll')) { if (-not $archive.GetEntry("$name/$file")) { throw "Archive missing $file" } } } finally { $archive.Dispose() }
$packages=@($zip)
if (-not $SkipInstaller) { $packages+=$installer }
foreach($file in $packages) {
    $hash=(Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $([IO.Path]::GetFileName($file))" | Set-Content -LiteralPath "$file.sha256" -Encoding ascii
    Write-Output $file
}
$sumLines=foreach($file in $packages){"$((Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant())  $([IO.Path]::GetFileName($file))"}
$sumLines | Set-Content -LiteralPath (Join-Path $OutputDir 'SHA256SUMS.txt') -Encoding ascii
