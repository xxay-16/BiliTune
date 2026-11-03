$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$cmakeCommand = Get-Command cmake.exe -ErrorAction SilentlyContinue
if ($cmakeCommand) {
    $cmake = $cmakeCommand.Source
} else {
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $visualStudio = & $vswhere -latest -products '*' `
        -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    $cmake = Join-Path $visualStudio 'Common7/IDE/CommonExtensions/Microsoft/CMake/CMake/bin/cmake.exe'
}
$buildDir = Join-Path $projectRoot 'build/native-hotkey-tests'
& $cmake -S (Join-Path $projectRoot 'test/windows') -B $buildDir -A x64
if ($LASTEXITCODE -ne 0) { throw 'Native hotkey test configuration failed.' }
& $cmake --build $buildDir --config Release
if ($LASTEXITCODE -ne 0) { throw 'Native hotkey tests did not compile.' }
$ctest = Join-Path (Split-Path -Parent $cmake) 'ctest.exe'
& $ctest --test-dir $buildDir -C Release --output-on-failure
if ($LASTEXITCODE -ne 0) { throw 'Native hotkey regression tests failed.' }
