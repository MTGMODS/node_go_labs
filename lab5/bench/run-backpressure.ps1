param(
    [ValidateRange(1, 1024)]
    [int]$SizeMB = 100,
    [ValidateRange(0, 100)]
    [int]$SlowDelayMS = 2
)

$ErrorActionPreference = "Stop"

function Wait-Job([string]$BaseUrl, [string]$JobId) {
    do {
        Start-Sleep -Milliseconds 200
        $job = Invoke-RestMethod -Uri "$BaseUrl/streams/jobs/$JobId"
    } while ($job.status -eq "running")
    if ($job.status -ne "completed") {
        throw "Job $JobId failed: $($job.error)"
    }
    return $job
}

function Run-Scenario([string]$Runtime, [string]$BaseUrl, [string]$Mode) {
    if ($Runtime -eq "Node.js") {
        $body = @{ mode = $Mode; slow_delay_ms = $SlowDelayMS; high_water_mark_kb = 64 } | ConvertTo-Json
    } else {
        $body = @{ mode = $Mode; slow_delay_ms = $SlowDelayMS; chunk_size_kb = 64 } | ConvertTo-Json
    }
    $started = Invoke-RestMethod -Method Post -Uri "$BaseUrl/streams/process" -ContentType "application/json" -Body $body
    $job = Wait-Job $BaseUrl $started.id
    $memoryBytes = if ($null -ne $job.rssGrowthBytes) { $job.rssGrowthBytes } else { $job.allocationGrowthBytes }
    return [pscustomobject]@{
        Runtime = $Runtime
        Mode = $Mode
        SizeMB = [math]::Round($job.sourceBytes / 1MB, 2)
        DurationMS = [math]::Round($job.durationMs, 2)
        MemoryGrowthMB = [math]::Round($memoryBytes / 1MB, 2)
        Lines = $job.lineCount
    }
}

$sourceBody = @{ size_mb = $SizeMB } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:3005/streams/source" -ContentType "application/json" -Body $sourceBody | Out-Null
Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8085/streams/source" -ContentType "application/json" -Body $sourceBody | Out-Null

$results = @(
    Run-Scenario "Node.js" "http://127.0.0.1:3005" "safe"
    Run-Scenario "Node.js" "http://127.0.0.1:3005" "unsafe"
    Run-Scenario "Go" "http://127.0.0.1:8085" "safe"
    Run-Scenario "Go" "http://127.0.0.1:8085" "unsafe"
)

$lines = @(
    "# Lab 5 Backpressure Results",
    "",
    "Source size: $SizeMB MB. Slow consumer delay: $SlowDelayMS ms per chunk.",
    "",
    "| Runtime | Mode | Size MB | Duration ms | Memory growth MB | Lines |",
    "| --- | --- | ---: | ---: | ---: | ---: |"
)
foreach ($result in $results) {
    $lines += "| $($result.Runtime) | $($result.Mode) | $($result.SizeMB) | $($result.DurationMS) | $($result.MemoryGrowthMB) | $($result.Lines) |"
}
$report = Join-Path $PSScriptRoot "results.md"
$lines | Set-Content -LiteralPath $report -Encoding utf8
$results | Format-Table -AutoSize
Write-Host "Report: $report"
