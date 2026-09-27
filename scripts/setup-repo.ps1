param(
    [Parameter(Mandatory = $true)]
    [string]$RepoPath,
    [string]$CoTicketRoot
)

$resolvedRepoPath = (Resolve-Path $RepoPath).Path
if ([string]::IsNullOrWhiteSpace($CoTicketRoot)) {
    $CoTicketRoot = Join-Path $PSScriptRoot ".."
}
$resolvedCoTicketRoot = (Resolve-Path $CoTicketRoot).Path
$coticketDir = Join-Path $resolvedRepoPath ".coticket"
$dbPath = Join-Path $coticketDir "coticket.db"

New-Item -ItemType Directory -Path $coticketDir -Force | Out-Null

$launcherPath = Join-Path $coticketDir "start-coticket-mcp.ps1"
$binaryPath = Join-Path $resolvedCoTicketRoot "coticket-mcp.exe"
$sourceEntrypoint = Join-Path $resolvedCoTicketRoot "cmd\coticket-mcp"

$launcherScript = @"
`$env:COTICKET_REPO_PATH = '$resolvedRepoPath'
`$env:COTICKET_DB_PATH = '$dbPath'
if (Test-Path '$binaryPath') {
    & '$binaryPath'
} else {
    go run '$sourceEntrypoint'
}
"@
Set-Content -Path $launcherPath -Value $launcherScript -Encoding UTF8

$configPath = Join-Path $coticketDir "mcp.coticket.example.json"
$mcpConfig = @{
    servers = @{
        coticket = @{
            command = "powershell"
            args    = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $launcherPath)
        }
    }
}
$mcpConfig | ConvertTo-Json -Depth 8 | Set-Content -Path $configPath -Encoding UTF8

Write-Host "Repository-scoped setup created:"
Write-Host "  Launcher: $launcherPath"
Write-Host "  DB path:  $dbPath"
Write-Host "  MCP config example: $configPath"
