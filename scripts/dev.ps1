param(
  [switch]$Build,
  [switch]$Force
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$ownedPorts = @(80, 443, 8088)
$conflicts = @()
$ours = (& docker ps --filter 'name=relayradar' --format '{{.Ports}}' 2>$null) -join "`n"
foreach ($port in $ownedPorts) {
  $conn = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
  if (-not $conn) { continue }
  if ($ours -match ":$port->") { continue }
  $proc = Get-Process -Id $conn.OwningProcess -ErrorAction SilentlyContinue
  $conflicts += "  port $port  PID $($conn.OwningProcess)  $($proc.ProcessName)"
}
if ($conflicts.Count -gt 0 -and -not $Force) {
  Write-Host 'Port conflict: these ports are used by something other than RelayRadar.'
  $conflicts | ForEach-Object { Write-Host $_ }
  Write-Host ''
  Write-Host 'Stop that process, or run with -Force to start anyway (nginx may fail).'
  exit 1
}

if (-not (Test-Path '.env')) {
  Copy-Item '.env.example' '.env'
  $pw = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 24 | ForEach-Object { [char]$_ })
  (Get-Content '.env') -replace '^POSTGRES_PASSWORD=.*$', "POSTGRES_PASSWORD=$pw" | Set-Content '.env'
  (Get-Content '.env') -replace '^PUBLIC_BASE_URL=.*$', 'PUBLIC_BASE_URL=https://localhost' | Set-Content '.env'
  Write-Host 'Created .env with a generated database password.'
  Write-Host 'If a PostgreSQL volume already existed, run: docker compose down -v'
}

if ($Build) {
  Write-Host 'Building web bundle...'
  Push-Location web
  try {
    npm.cmd ci
    npm.cmd run build
  } finally {
    Pop-Location
  }
  Write-Host 'Building images...'
  docker compose build
}

Write-Host 'Starting stack...'
docker compose up -d

$https = 'https://localhost/'
$http = 'http://localhost:8088/'
$deadline = (Get-Date).AddSeconds(90)
$ok = $false
while ((Get-Date) -lt $deadline) {
  $code = & curl.exe -k -s -o NUL -w '%{http_code}' --max-time 3 $https 2>$null
  if ($code -eq '200') { $ok = $true; break }
  Start-Sleep -Seconds 2
}

Write-Host ''
if ($ok) {
  Write-Host 'RelayRadar is up:'
  Write-Host "  HTTPS  $https"
  Write-Host "  HTTP   $http"
} else {
  Write-Host 'Stack started but HTTPS did not answer 200 within 90s.'
  Write-Host 'Check: docker compose ps && docker compose logs api nginx'
}
Write-Host ''
& docker compose ps --format '{{.Service}} {{.Status}}'
