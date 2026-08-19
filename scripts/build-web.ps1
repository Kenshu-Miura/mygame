$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$distDir = Join-Path $projectRoot "dist"
$resolvedProjectRoot = [System.IO.Path]::GetFullPath($projectRoot)
$resolvedDistDir = [System.IO.Path]::GetFullPath($distDir)
if (-not $resolvedDistDir.StartsWith($resolvedProjectRoot + [System.IO.Path]::DirectorySeparatorChar)) {
    throw "Build directory escaped the project root: $resolvedDistDir"
}

if (Test-Path -LiteralPath $distDir) {
    Remove-Item -LiteralPath $distDir -Recurse -Force
}
New-Item -ItemType Directory -Path $distDir | Out-Null

$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
Push-Location $projectRoot
try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"
    & go build -trimpath -ldflags="-s -w" -o (Join-Path $distDir "mygame.wasm") .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
}
finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    Pop-Location
}

$goRoot = (& go env GOROOT).Trim()
$wasmExec = Join-Path $goRoot "lib/wasm/wasm_exec.js"
if (-not (Test-Path -LiteralPath $wasmExec)) {
    $wasmExec = Join-Path $goRoot "misc/wasm/wasm_exec.js"
}
if (-not (Test-Path -LiteralPath $wasmExec)) {
    throw "wasm_exec.js was not found under $goRoot"
}

Copy-Item -LiteralPath $wasmExec -Destination $distDir
Get-ChildItem -LiteralPath (Join-Path $projectRoot "web") -Filter "*.html" -File | Copy-Item -Destination $distDir
Get-ChildItem -LiteralPath $projectRoot -File | Where-Object Extension -In ".png", ".wav", ".ogg" | Copy-Item -Destination $distDir

Write-Output "Web build completed: $distDir"
