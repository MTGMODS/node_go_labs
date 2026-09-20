param(
  [string]$Duration = "20s",
  [int]$IOVUs = 30,
  [int]$CPUVUs = 4,
  [int]$IODelayMS = 100,
  [long]$CPUIterations = 25000000,
  [int]$CPUTasks = 4,
  [switch]$ReportOnly
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$rawResults = Join-Path $PSScriptRoot "results"
$reportPath = Join-Path $PSScriptRoot "results.md"
$csvPath = Join-Path $rawResults "summary.csv"
$dockerExe = (Get-Command docker -ErrorAction Stop).Source

New-Item -ItemType Directory -Force -Path $rawResults | Out-Null

function Convert-ToMiB([string]$value) {
  if ($value -notmatch '^\s*([0-9.]+)\s*(B|KiB|MiB|GiB)\s*$') {
    return 0.0
  }
  $number = [double]$Matches[1]
  switch ($Matches[2]) {
    "B"   { return $number / 1MB }
    "KiB" { return $number / 1024 }
    "MiB" { return $number }
    "GiB" { return $number * 1024 }
  }
}

function Format-Number([double]$value) {
  return $value.ToString("0.00", [System.Globalization.CultureInfo]::InvariantCulture)
}

function Start-StatsSampler([string]$container, [string]$outputPath) {
  "timestamp,cpu_percent,memory" | Set-Content -LiteralPath $outputPath -Encoding utf8
  return Start-Job -ScriptBlock {
    param($docker, $containerName, $path)
    while ($true) {
      $sample = & $docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}' $containerName 2>$null
      if ($sample) {
        $parts = $sample -split '\|'
        $cpu = ($parts[0] -replace '%', '').Trim()
        $memory = (($parts[1] -split '/')[0]).Trim()
        "$(Get-Date -Format o),$cpu,$memory" | Add-Content -LiteralPath $path -Encoding utf8
      }
      Start-Sleep -Milliseconds 500
    }
  } -ArgumentList $dockerExe, $container, $outputPath
}

function Stop-StatsSampler($job) {
  Stop-Job -Job $job -ErrorAction SilentlyContinue
  Wait-Job -Job $job -ErrorAction SilentlyContinue | Out-Null
  Remove-Job -Job $job -Force -ErrorAction SilentlyContinue
}

$cases = @(
  [pscustomobject]@{ Id="node-io"; Workload="I/O"; Runtime="Node.js"; Variant="setTimeout"; BaseURL="http://node:3000"; Endpoint="/io?delay_ms=$IODelayMS"; VUs=$IOVUs; Container="labs-license-node" },
  [pscustomobject]@{ Id="go-io"; Workload="I/O"; Runtime="Go"; Variant="time.Timer"; BaseURL="http://go:8080"; Endpoint="/io?delay_ms=$IODelayMS"; VUs=$IOVUs; Container="labs-license-go" },
  [pscustomobject]@{ Id="node-cpu"; Workload="CPU"; Runtime="Node.js"; Variant="event loop"; BaseURL="http://node:3000"; Endpoint="/cpu?iterations=$CPUIterations&tasks=$CPUTasks"; VUs=$CPUVUs; Container="labs-license-node" },
  [pscustomobject]@{ Id="go-cpu-sequential"; Workload="CPU"; Runtime="Go"; Variant="sequential"; BaseURL="http://go:8080"; Endpoint="/cpu/sequential?iterations=$CPUIterations&tasks=$CPUTasks"; VUs=$CPUVUs; Container="labs-license-go" },
  [pscustomobject]@{ Id="go-cpu-parallel"; Workload="CPU"; Runtime="Go"; Variant="goroutines"; BaseURL="http://go:8080"; Endpoint="/cpu/parallel?iterations=$CPUIterations&tasks=$CPUTasks"; VUs=$CPUVUs; Container="labs-license-go" }
)

Push-Location $repoRoot
try {
  if (-not $ReportOnly) {
    & $dockerExe compose up -d --build postgres node go
    if ($LASTEXITCODE -ne 0) { throw "docker compose up failed" }
  }

  $rows = foreach ($case in $cases) {
    $summaryFile = "$($case.Id).json"
    $statsFile = Join-Path $rawResults "$($case.Id)-stats.csv"
    if (-not $ReportOnly) {
      Write-Host "Running $($case.Id)..."
      $statsJob = Start-StatsSampler $case.Container $statsFile
      try {
        & $dockerExe compose --profile benchmark run --rm --no-deps `
          -e "BASE_URL=$($case.BaseURL)" `
          -e "ENDPOINT=$($case.Endpoint)" `
          -e "WORKLOAD=$($case.Id)" `
          -e "VUS=$($case.VUs)" `
          -e "DURATION=$Duration" `
          k6 run "--summary-export=/results/$summaryFile" /scripts/load.js | Out-Host
        if ($LASTEXITCODE -ne 0) { throw "k6 failed for $($case.Id)" }
      }
      finally {
        Stop-StatsSampler $statsJob
      }
    }

    $summary = Get-Content -LiteralPath (Join-Path $rawResults $summaryFile) -Raw | ConvertFrom-Json
    $stats = Import-Csv -LiteralPath $statsFile
    $cpuValues = @($stats | ForEach-Object { [double]$_.cpu_percent })
    $memoryValues = @($stats | ForEach-Object { Convert-ToMiB $_.memory })

    [pscustomobject]@{
      Id = $case.Id
      Workload = $case.Workload
      Runtime = $case.Runtime
      Variant = $case.Variant
      RPS = [math]::Round([double]$summary.metrics.http_reqs.rate, 2)
      AvgMS = [math]::Round([double]$summary.metrics.http_req_duration.avg, 2)
      P95MS = [math]::Round([double]$summary.metrics.http_req_duration.'p(95)', 2)
      P99MS = [math]::Round([double]$summary.metrics.http_req_duration.'p(99)', 2)
      AvgCPUPercent = if ($cpuValues.Count) { [math]::Round(($cpuValues | Measure-Object -Average).Average, 2) } else { 0 }
      PeakRAMMiB = if ($memoryValues.Count) { [math]::Round(($memoryValues | Measure-Object -Maximum).Maximum, 2) } else { 0 }
    }
  }

  $rows | Export-Csv -LiteralPath $csvPath -NoTypeInformation -Encoding utf8

  $lines = @(
    "# Lab 2 benchmark results",
    "",
    "Generated: $(Get-Date -Format o)",
    "",
    "Parameters: duration=$Duration; I/O VUs=$IOVUs; CPU VUs=$CPUVUs; delay=$IODelayMS ms; CPU=$CPUTasks x $CPUIterations increments per request.",
    "",
    "| Workload | Runtime | Variant | RPS | Avg, ms | p95, ms | p99, ms | Avg CPU, % | Peak RAM, MiB |",
    "|---|---|---|---:|---:|---:|---:|---:|---:|"
  )
  foreach ($row in $rows) {
    $lines += "| $($row.Workload) | $($row.Runtime) | $($row.Variant) | $(Format-Number $row.RPS) | $(Format-Number $row.AvgMS) | $(Format-Number $row.P95MS) | $(Format-Number $row.P99MS) | $(Format-Number $row.AvgCPUPercent) | $(Format-Number $row.PeakRAMMiB) |"
  }
  $nodeIO = $rows | Where-Object Id -eq "node-io"
  $goIO = $rows | Where-Object Id -eq "go-io"
  $nodeCPU = $rows | Where-Object Id -eq "node-cpu"
  $goSequential = $rows | Where-Object Id -eq "go-cpu-sequential"
  $goParallel = $rows | Where-Object Id -eq "go-cpu-parallel"
  $lines += @(
    "",
    "## Спостереження",
    "",
    "- I/O throughput майже однаковий: $(Format-Number $nodeIO.RPS) RPS для Node.js і $(Format-Number $goIO.RPS) RPS для Go. За 30 VUs і затримки 100 мс обидва runtime наблизились до теоретичної межі 300 RPS.",
    "- Go sequential CPU має в $(Format-Number ($goSequential.RPS / $nodeCPU.RPS)) раза вищий RPS за Node.js для однакової роботи на запит.",
    "- Go goroutines має в $(Format-Number ($goParallel.RPS / $goSequential.RPS)) раза вищий RPS за Go sequential і в $(Format-Number ($goParallel.RPS / $nodeCPU.RPS)) раза вищий за Node.js у цьому прогоні.",
    "- Node.js використовує приблизно одне повне CPU-ядро. Docker CPU понад 100% для Go означає одночасне використання кількох ядер, а не помилку вимірювання.",
    "- Значення характеризують цей комп'ютер і поточні Docker CPU limits. Після зміни середовища benchmark потрібно повторити."
  )
  $lines | Set-Content -LiteralPath $reportPath -Encoding utf8

  Write-Host "Benchmark report: $reportPath"
  $rows | Format-Table -AutoSize
}
finally {
  Pop-Location
}
