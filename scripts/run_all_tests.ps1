# Re-runs every test from a clean state on Windows (equivalent of `make test-all`):
#   1. Go vet + unit tests (crypto core, server handlers/store, bridge, bench stats)
#   2. End-to-end U01-U12 + 5 transaction chains on p1 and p2 against the real
#      server process -> tests/results/{RESULTS.md, results.json, chains.csv, samples/}
#   3. Web: lint, typecheck, Vitest
#   4. WASM bridge build + Playwright GUI flows -> report/figures/gui, tests/results/screenshots
#
# Usage: powershell -ExecutionPolicy Bypass -File scripts/run_all_tests.ps1 [-SkipUI]
param([switch]$SkipUI)
. (Join-Path $PSScriptRoot "common.ps1")

$results = Join-Path $Repo "tests/results"
if (Test-Path $results) { Get-ChildItem $results -Exclude ".gitkeep" | Remove-Item -Recurse -Force }
New-Item -ItemType Directory -Force $results | Out-Null

Invoke-Step "go vet" { go vet ./... }
Invoke-Step "go unit tests" { go test -count=1 -cover ./... }
Invoke-Step "e2e U01-U12 + chains (p1, p2)" { go test -tags e2e -count=1 -v ./tests/e2e/ }

Ensure-WebDeps
Push-Location web
try {
    Invoke-Step "web lint" { npm run lint }
    Invoke-Step "web typecheck" { npm run typecheck }
    Invoke-Step "web unit tests (vitest)" { npm test }
} finally { Pop-Location }

if (-not $SkipUI) {
    Invoke-Step "build crypto.wasm" { Build-Wasm; $global:LASTEXITCODE = 0 }
    Push-Location web
    try {
        $env:GO_BIN = (Get-Command go).Source
        Invoke-Step "Playwright GUI tests" { npx playwright test }
    } finally { Pop-Location }
    $shots = Join-Path $results "screenshots"
    New-Item -ItemType Directory -Force $shots | Out-Null
    Copy-Item report/figures/gui/*.png $shots -Force
}

Write-Host ""
Write-Host "All tests passed. Results in tests/results (RESULTS.md, chains.csv, samples/, screenshots/)." -ForegroundColor Green
