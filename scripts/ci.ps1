param(
  [switch]$SkipDocker,
  [switch]$SkipInstall
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$ws = Split-Path -Parent $root
Set-Location $root

Write-Host '== backend =='
Push-Location backend
try {
  $env:GOCACHE = "$ws\.cache\go-build"
  $env:GOTMPDIR = "$ws\.cache\gotmp"
  $env:GOMODCACHE = "$ws\.gomodcache"
  $env:GOFLAGS = '-mod=mod'
  go vet ./...
  if ($LASTEXITCODE -ne 0) { throw 'go vet failed' }
  go test ./...
  if ($LASTEXITCODE -ne 0) { throw 'go test failed' }
} finally {
  Pop-Location
}

Write-Host '== web =='
Push-Location web
try {
  $env:npm_config_cache = "$ws\.npmcache"
  $env:TMP = "$ws\.cache\nodetmp"
  $env:TEMP = "$ws\.cache\nodetmp"
  npm.cmd ci
  if ($LASTEXITCODE -ne 0 -and -not $SkipInstall) { throw 'npm ci failed' }
  if ($LASTEXITCODE -ne 0) { Write-Host 'npm ci failed (Windows file locks); continuing with installed deps.' }
  npm.cmd run build
  if ($LASTEXITCODE -ne 0) { throw 'npm run build failed' }
} finally {
  Pop-Location
}

if (-not $SkipDocker) {
  Write-Host '== compose =='
  $env:POSTGRES_PASSWORD = if ($env:POSTGRES_PASSWORD) { $env:POSTGRES_PASSWORD } else { 'ci-password' }
  docker compose config --quiet
  if ($LASTEXITCODE -ne 0) { throw 'compose config failed' }
  docker compose build
  if ($LASTEXITCODE -ne 0) { throw 'compose build failed' }
}

Write-Host ''
Write-Host 'Local CI passed.'
