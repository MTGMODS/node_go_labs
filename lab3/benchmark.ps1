param(
  [int]$Tasks = 10000,
  [int]$Iterations = 10000
)

$ErrorActionPreference = "Stop"
$composeFile = Join-Path $PSScriptRoot "docker-compose.yml"
$reportFile = Join-Path $PSScriptRoot "results.md"
$dockerExe = (Get-Command docker -ErrorAction Stop).Source

$cases = @(
  [pscustomobject]@{ Workers=1;  Buffer=0 },
  [pscustomobject]@{ Workers=2;  Buffer=0 },
  [pscustomobject]@{ Workers=4;  Buffer=0 },
  [pscustomobject]@{ Workers=8;  Buffer=0 },
  [pscustomobject]@{ Workers=16; Buffer=0 },
  [pscustomobject]@{ Workers=4;  Buffer=10 },
  [pscustomobject]@{ Workers=4;  Buffer=100 },
  [pscustomobject]@{ Workers=4;  Buffer=1000 }
)

& $dockerExe compose -f $composeFile build pipeline
if ($LASTEXITCODE -ne 0) { throw "Docker build failed" }

$rows = foreach ($case in $cases) {
  Write-Host "Running workers=$($case.Workers), buffer=$($case.Buffer)..."
  $output = & $dockerExe compose -f $composeFile run --rm --no-deps pipeline `
    "-tasks=$Tasks" `
    "-workers=$($case.Workers)" `
    "-buffer=$($case.Buffer)" `
    "-iterations=$Iterations" `
    "-format=json"
  if ($LASTEXITCODE -ne 0) { throw "Benchmark case failed" }
  $stats = ($output -join "") | ConvertFrom-Json
  [pscustomobject]@{
    Tasks = $stats.tasks
    Workers = $stats.workers
    Buffer = $stats.buffer
    Completed = $stats.completed
    TimeMS = [math]::Round([double]$stats.duration_ms, 3)
    Throughput = [math]::Round([double]$stats.throughput_tasks_per_sec, 2)
    Checksum = $stats.checksum
  }
}

$lines = @(
  "# Lab 3 benchmark results",
  "",
  "Generated: $(Get-Date -Format o)",
  "",
  "Parameters: tasks=$Tasks; calculation iterations per task=$Iterations.",
  "",
  "| Tasks | Workers | Buffer | Completed | Time, ms | Throughput, tasks/s | Checksum |",
  "|---:|---:|---:|---:|---:|---:|---:|"
)
foreach ($row in $rows) {
  $lines += "| $($row.Tasks) | $($row.Workers) | $($row.Buffer) | $($row.Completed) | $($row.TimeMS) | $($row.Throughput) | $($row.Checksum) |"
}
$lines | Set-Content -LiteralPath $reportFile -Encoding utf8

Write-Host "Benchmark report: $reportFile"
$rows | Format-Table -AutoSize
