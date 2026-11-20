@echo off
setlocal
set ROOT=%~dp0..
if "%1"=="" goto help

if /I "%1"=="dev"   ( call "%ROOT%\scripts\dev.cmd" & goto end )
if /I "%1"=="up"    ( docker compose -f "%ROOT%\docker-compose.yml" up -d & goto end )
if /I "%1"=="build" ( call "%ROOT%\scripts\dev.cmd" -Build & goto end )
if /I "%1"=="logs"  ( docker compose -f "%ROOT%\docker-compose.yml" logs -f --tail=100 & goto end )
if /I "%1"=="ps"    ( docker compose -f "%ROOT%\docker-compose.yml" ps & goto end )
if /I "%1"=="stop"  ( docker compose -f "%ROOT%\docker-compose.yml" down & goto end )
if /I "%1"=="test"  ( pushd "%ROOT%\backend" & go vet ./... & go test ./... & popd & pushd "%ROOT%\web" & call npm.cmd run build & popd & goto end )
if /I "%1"=="ci"    ( powershell -NoProfile -ExecutionPolicy Bypass -File "%ROOT%\scripts\ci.ps1" & goto end )

:help
echo Usage: scripts\task.cmd ^<dev^|up^|build^|logs^|ps^|stop^|test^>
exit /b 0

:end
