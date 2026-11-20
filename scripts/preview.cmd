@echo off
rem Launch the preview script with the execution policy bypassed.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0preview.ps1" %*
