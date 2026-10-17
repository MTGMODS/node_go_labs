param(
    [string]$NodeBaseUrl = "http://127.0.0.1:3006",
    [string]$GoBaseUrl = "http://127.0.0.1:8086"
)

$ErrorActionPreference = "Stop"

function Assert-Equal($Actual, $Expected, [string]$Message) {
    if ($Actual -ne $Expected) {
        throw "$Message. Expected '$Expected', got '$Actual'."
    }
}

$nodeHealth = Invoke-RestMethod "$NodeBaseUrl/health"
$goHealth = Invoke-RestMethod "$GoBaseUrl/health"
Assert-Equal $nodeHealth.status "UP" "Node health check failed"
Assert-Equal $goHealth.status "UP" "Go health check failed"
Assert-Equal $nodeHealth.repository $goHealth.repository "Services use different repositories"

$key = "LAB6-SMOKE-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())"
$body = @{ key = $key; product = "MTG MODS VIP"; owner = "smoke-test" } | ConvertTo-Json
$created = Invoke-RestMethod -Method Post -Uri "$NodeBaseUrl/api/licenses" -ContentType "application/json" -Body $body
$found = Invoke-RestMethod "$GoBaseUrl/api/licenses/$($created.id)"
Assert-Equal $found.key $key "Cross-runtime database read failed"

$updated = Invoke-RestMethod -Method Put -Uri "$GoBaseUrl/api/licenses/$($created.id)" -ContentType "application/json" -Body (@{ status = "ACTIVE" } | ConvertTo-Json)
Assert-Equal $updated.status "ACTIVE" "Update failed"

$duplicate = Invoke-WebRequest -Method Post -Uri "$GoBaseUrl/api/licenses" -ContentType "application/json" -Body $body -SkipHttpErrorCheck
Assert-Equal $duplicate.StatusCode 409 "Duplicate key did not produce Conflict"

$events = Invoke-RestMethod "$NodeBaseUrl/api/events?limit=50"
if (($events | Where-Object { $_.data.key -eq $key }).Count -lt 2) {
    throw "MongoDB event logger did not persist create and update events."
}

$nodeExport = Invoke-RestMethod "$NodeBaseUrl/api/licenses/export?batch_size=1"
$goExport = Invoke-RestMethod "$GoBaseUrl/api/licenses/export?batch_size=1"
if ($nodeExport.processed -lt 1 -or $goExport.processed -lt 1) {
    throw "Database stream export did not process records."
}

$transfer = Invoke-RestMethod -Method Post -Uri "$NodeBaseUrl/api/accounts/transfer" -ContentType "application/json" -Body (@{
    from_account_id = 1; to_account_id = 2; amount = 0.01
} | ConvertTo-Json)
Assert-Equal $transfer.status "committed" "Successful transfer did not commit"
$afterCommit = Invoke-RestMethod "$GoBaseUrl/api/accounts"

$rollback = Invoke-WebRequest -Method Post -Uri "$GoBaseUrl/api/accounts/transfer" -ContentType "application/json" -Body (@{
    from_account_id = 2; to_account_id = 1; amount = 999999.00
} | ConvertTo-Json) -SkipHttpErrorCheck
Assert-Equal $rollback.StatusCode 409 "Insufficient balance did not produce Conflict"
$afterRollback = Invoke-RestMethod "$NodeBaseUrl/api/accounts"
Assert-Equal ($afterRollback | ConvertTo-Json -Compress) ($afterCommit | ConvertTo-Json -Compress) "Rollback changed account balances"

$deleted = Invoke-WebRequest -Method Delete -Uri "$GoBaseUrl/api/licenses/$($created.id)" -SkipHttpErrorCheck
Assert-Equal $deleted.StatusCode 204 "Delete failed"

[pscustomobject]@{
    Repository       = $nodeHealth.repository
    CrossRuntimeCRUD = "PASS"
    Duplicate409     = "PASS"
    MongoEventLog    = "PASS"
    Transaction      = "COMMIT + ROLLBACK PASS"
    NodeExport       = $nodeExport.filename
    GoExport         = $goExport.filename
} | Format-List
