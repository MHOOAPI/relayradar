# 本地预览：构建前端（如缺失）→ 启动 Go API → 单进程托管 dist
# 用法：
#   .\scripts\preview.ps1                # memory 模式，http://127.0.0.1:8080
#   .\scripts\preview.ps1 -Port 8090
#   .\scripts\preview.ps1 -Store postgres # 需要 DATABASE_URL 指向可用库
param(
  [int]$Port = 8080,
  [ValidateSet('memory', 'postgres')][string]$Store = 'memory'
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'web\dist'

if (-not (Test-Path (Join-Path $dist 'index.html'))) {
  Write-Host '未找到前端构建产物，正在构建…' -ForegroundColor Yellow
  Push-Location (Join-Path $root 'web')
  try {
    if (-not (Test-Path 'node_modules')) { npm.cmd install }
    npm.cmd run build
  } finally { Pop-Location }
}

$env:ADDR = ":$Port"
$env:STORE = $Store
$env:WEB_DIR = $dist

Push-Location (Join-Path $root 'backend')
try {
  $exe = Join-Path $root '.cache\relayradar-api.exe'
  if (-not (Test-Path $exe)) {
    New-Item -ItemType Directory -Force -Path (Split-Path $exe) | Out-Null
    go build -o $exe ./cmd/api
  }
  Write-Host "RelayRadar → http://127.0.0.1:$Port  (store=$Store)" -ForegroundColor Green
  & $exe
} finally { Pop-Location }
