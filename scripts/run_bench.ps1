# Re-runs the Table 2.7 benchmark on Windows (equivalent of `make bench`):
# native Go metrics (keygen, client crypto, server evaluation, end-to-end, sizes)
# on both profiles, then the WASM client metrics under Node, then the summary.
# Output: bench/results/raw/*.csv, bench/results/summary.md, bench/ENVIRONMENT.md
#
# Usage: powershell -ExecutionPolicy Bypass -File scripts/run_bench.ps1 [-Runs 30] [-Warmup 5]
param([int]$Runs = 30, [int]$Warmup = 5)
. (Join-Path $PSScriptRoot "common.ps1")

$raw = Join-Path $Repo "bench/results/raw"
if (Test-Path $raw) { Get-ChildItem $raw -Exclude ".gitkeep" | Remove-Item -Force }

Invoke-Step "build crypto.wasm" { Build-Wasm; $global:LASTEXITCODE = 0 }
Invoke-Step "native benchmark (p1, p2)" { go run ./cmd/bench -runs $Runs -warmup $Warmup }
Invoke-Step "WASM benchmark (Node)" { node bench/wasm/bench.mjs "--runs=$Runs" "--warmup=$Warmup" }
Invoke-Step "summary" { go run ./cmd/bench -report }
Write-Host "Benchmark done: bench/results/summary.md" -ForegroundColor Green
