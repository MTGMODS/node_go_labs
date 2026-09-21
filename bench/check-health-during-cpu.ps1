param(
  [long]$Iterations = 250000000,
  [int]$Tasks = 4,
  [int]$HealthRequests = 5
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$reportPath = Join-Path $PSScriptRoot "health-results.md"
$dockerExe = (Get-Command docker -ErrorAction Stop).Source

$cases = @(
  [pscustomobject]@{ Runtime="Node.js"; Variant="event loop"; CPU="http://localhost:3000/cpu?iterations=$Iterations&tasks=$Tasks"; Health="http://localhost:3000/health" },
  [pscustomobject]@{ Runtime="Go"; Variant="sequential"; CPU="http://localhost:8080/cpu/sequential?iterations=$Iterations&tasks=$Tasks"; Health="http://localhost:8080/health" },
  [pscustomobject]@{ Runtime="Go"; Variant="goroutines"; CPU="http://localhost:8080/cpu/parallel?iterations=$Iterations&tasks=$Tasks"; Health="http://localhost:8080/health" }
)

Push-Location $repoRoot
try {
  & $dockerExe compose up -d --build postgres node go
  if ($LASTEXITCODE -ne 0) { throw "docker compose up failed" }

  $rows = foreach ($case in $cases) {
    Write-Host "CPU then health: $($case.Runtime) $($case.Variant)..."
    $cpuJob = Start-Job -ScriptBlock {
      param($uri)
      $timer = [System.Diagnostics.Stopwatch]::StartNew()
      $response = Invoke-WebRequest -UseBasicParsing -Uri $uri -TimeoutSec 300
      $timer.Stop()
      [pscustomobject]@{ Status=[int]$response.StatusCode; DurationMS=[math]::Round($timer.Elapsed.TotalMilliseconds, 2) }
    } -ArgumentList $case.CPU

    Start-Sleep -Milliseconds 200
    $caseRows = for ($index = 1; $index -le $HealthRequests; $index++) {
      $timer = [System.Diagnostics.Stopwatch]::StartNew()
      $response = Invoke-WebRequest -UseBasicParsing -Uri $case.Health -TimeoutSec 300
      $timer.Stop()
      [pscustomobject]@{
        Runtime = $case.Runtime
        Variant = $case.Variant
        Request = $index
        Status = [int]$response.StatusCode
        HealthLatencyMS = [math]::Round($timer.Elapsed.TotalMilliseconds, 2)
      }
    }

    Wait-Job -Job $cpuJob | Out-Null
    $cpuResult = Receive-Job -Job $cpuJob
    Remove-Job -Job $cpuJob -Force
    Write-Host "CPU request completed in $($cpuResult.DurationMS) ms"
    foreach ($row in $caseRows) {
      $row | Add-Member -NotePropertyName CPURequestMS -NotePropertyValue $cpuResult.DurationMS
      $row
    }
  }

  $lines = @(
    "# Health latency while CPU work is running",
    "",
    "Generated: $(Get-Date -Format o)",
    "",
    "CPU workload: $Tasks x $Iterations increments.",
    "",
    "| Runtime | CPU variant | CPU request, ms | Health request | Status | Health latency, ms |",
    "|---|---|---:|---:|---:|---:|"
  )
  foreach ($row in $rows) {
    $lines += "| $($row.Runtime) | $($row.Variant) | $($row.CPURequestMS) | $($row.Request) | $($row.Status) | $($row.HealthLatencyMS) |"
  }
  $nodeFirst = $rows | Where-Object { $_.Runtime -eq "Node.js" -and $_.Request -eq 1 }
  $goSequentialFirst = $rows | Where-Object { $_.Runtime -eq "Go" -and $_.Variant -eq "sequential" -and $_.Request -eq 1 }
  $goParallelFirst = $rows | Where-Object { $_.Runtime -eq "Go" -and $_.Variant -eq "goroutines" -and $_.Request -eq 1 }
  $lines += @(
    "",
    "## Observations",
    "",
    "The first Node.js health request waited $($nodeFirst.HealthLatencyMS) ms, almost the full duration of the CPU request. Go responded while computation was running: $($goSequentialFirst.HealthLatencyMS) ms for sequential CPU and $($goParallelFirst.HealthLatencyMS) ms for goroutines."
  )
  $lines | Set-Content -LiteralPath $reportPath -Encoding utf8
  Write-Host "Health experiment report: $reportPath"
  $rows | Format-Table -AutoSize
}
finally {
  Pop-Location
}
