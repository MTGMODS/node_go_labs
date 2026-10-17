param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("postgres", "mongo")]
    [string]$Type
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $projectRoot ".env"

if (-not (Test-Path -LiteralPath $envPath)) {
    throw ".env was not found. Create it before switching repositories."
}

$content = Get-Content -Raw -LiteralPath $envPath
if ($content -notmatch "(?m)^REPO_TYPE=") {
    throw "REPO_TYPE is missing from .env"
}
$updated = $content -replace "(?m)^REPO_TYPE=.*$", "REPO_TYPE=$Type"
[System.IO.File]::WriteAllText($envPath, $updated, [System.Text.UTF8Encoding]::new($false))

Push-Location $projectRoot
try {
    docker compose up -d --force-recreate node go
    docker compose ps
} finally {
    Pop-Location
}
