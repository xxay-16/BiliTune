[CmdletBinding()]
param([switch]$SkipFrontend, [switch]$Dev)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    if (-not $SkipFrontend) {
        Push-Location frontend
        try { & npm ci; if($LASTEXITCODE -ne 0){throw 'npm ci failed.'}; & npm run build; if($LASTEXITCODE -ne 0){throw 'Frontend build failed.'} } finally {Pop-Location}
    }
    New-Item -ItemType Directory -Path bin,.cache/tools -Force | Out-Null
    $wails=Join-Path $root '.cache/tools/wails3.exe'
    if (-not (Test-Path -LiteralPath $wails)) {
        $previousGobin=$env:GOBIN
        try {$env:GOBIN=Join-Path $root '.cache/tools'; & go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0; if($LASTEXITCODE -ne 0){throw 'Wails CLI install failed.'}} finally {$env:GOBIN=$previousGobin}
    }
    & $wails generate syso -arch amd64 -icon packaging/windows/icon.ico -manifest packaging/windows/wails.exe.manifest -info packaging/windows/info.json -out wails_windows_amd64.syso
    if ($LASTEXITCODE -ne 0) {throw 'Windows resources failed.'}
    $oldCgo=$env:CGO_ENABLED; $oldOS=$env:GOOS; $oldArch=$env:GOARCH
    try {
        $env:CGO_ENABLED='0';$env:GOOS='windows';$env:GOARCH='amd64'
        if($Dev){& go build -o bin/bilitune.exe .}else{& go build -trimpath -tags production -ldflags '-s -w -H windowsgui' -o bin/bilitune.exe .}
        if($LASTEXITCODE -ne 0){throw 'Go build failed.'}
    } finally {$env:CGO_ENABLED=$oldCgo;$env:GOOS=$oldOS;$env:GOARCH=$oldArch}
    & (Join-Path $PSScriptRoot 'fetch-mpv.ps1')
    Write-Output (Join-Path $root 'bin/bilitune.exe')
} finally {
    $resource=Join-Path $root 'wails_windows_amd64.syso'
    if (Test-Path -LiteralPath $resource) {Remove-Item -LiteralPath $resource}
    Pop-Location
}
