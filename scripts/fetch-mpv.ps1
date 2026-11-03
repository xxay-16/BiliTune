[CmdletBinding()]
param([string]$OutputDir, [string]$CacheDir)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
if (-not $OutputDir) {$OutputDir=Join-Path $root 'bin'}
if (-not $CacheDir) {$CacheDir=Join-Path $root '.cache/mpv'}
New-Item -ItemType Directory -Path $OutputDir,$CacheDir -Force | Out-Null
$dllHash='0a5a0b476866c91a639a4e511e8153968f8fa23564461b0aa09acdba816164a4'
$archiveHash='583af5a291fc99ae2641794ede1955c368eb4c19dc05f4f0a9c7f9456edeb6a8'
$destination=Join-Path $OutputDir 'libmpv-2.dll'
if ((Test-Path -LiteralPath $destination) -and (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash.ToLowerInvariant() -eq $dllHash) {return}
$filename='mpv-dev-x86_64-20230924-git-652a1dd.7z'
$archive=Join-Path $CacheDir $filename
if (-not (Test-Path -LiteralPath $archive)) {
    Invoke-WebRequest -Uri "https://github.com/media-kit/libmpv-win32-audio-build/releases/download/2023-09-24/$filename" -OutFile $archive
}
if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $archiveHash) {throw 'libmpv archive checksum mismatch.'}
$seven=(Get-Command 7z.exe -ErrorAction SilentlyContinue).Source
if (-not $seven) {$seven=Join-Path $env:ProgramFiles '7-Zip/7z.exe'}
if (-not (Test-Path -LiteralPath $seven)) {throw '7-Zip is required to extract the pinned libmpv runtime.'}
& $seven x $archive 'libmpv-2.dll' "-o$OutputDir" -y | Out-Null
if ($LASTEXITCODE -ne 0 -or (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash.ToLowerInvariant() -ne $dllHash) {throw 'libmpv DLL verification failed.'}
