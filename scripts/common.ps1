# Shared helpers for the Windows scripts (dot-sourced).
$ErrorActionPreference = "Stop"

# Repository root with an upper-case drive letter (Vitest/Vite on Windows can
# load two copies of a module when the working directory uses "c:" instead of "C:").
$Repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if ($Repo -match '^[a-z]:') { $Repo = $Repo.Substring(0, 1).ToUpper() + $Repo.Substring(1) }
Set-Location $Repo

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    if (Test-Path "C:\Program Files\Go\bin\go.exe") { $env:PATH = "C:\Program Files\Go\bin;" + $env:PATH }
    else { throw "Go toolchain not found (install Go 1.26+)" }
}
if (-not (Get-Command node -ErrorAction SilentlyContinue)) { throw "Node.js not found (install Node 24)" }

function Invoke-Step([string]$Name, [scriptblock]$Block) {
    Write-Host ""
    Write-Host "==> $Name" -ForegroundColor Cyan
    $start = Get-Date
    & $Block
    if ($LASTEXITCODE -ne 0) { throw "$Name failed (exit $LASTEXITCODE)" }
    Write-Host ("    ok ({0:N1}s)" -f ((Get-Date) - $start).TotalSeconds) -ForegroundColor Green
}

function Build-Wasm {
    $env:GOOS = "js"; $env:GOARCH = "wasm"
    try { go build -trimpath -ldflags="-s -w" -o web/public/crypto.wasm ./cmd/wasm }
    finally { Remove-Item Env:GOOS; Remove-Item Env:GOARCH }
    if ($LASTEXITCODE -ne 0) { throw "wasm build failed" }
    Copy-Item (Join-Path (go env GOROOT) "lib/wasm/wasm_exec.js") web/public/wasm_exec.js -Force
}

function Ensure-WebDeps {
    if (-not (Test-Path web/node_modules)) {
        Push-Location web; try { npm ci } finally { Pop-Location }
    }
}
