# Build Net-Companion Lite : frontend Vite -> go:embed -> binaire unique.
$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
$go = "C:\Program Files\Go\bin\go.exe"

Write-Host "==> Build frontend (Vite)"
Push-Location (Join-Path $root "frontend")
npm install
npm run build
Pop-Location

Write-Host "==> Build backend (go:embed)"
Push-Location (Join-Path $root "backend")
& $go build -o net-companion.exe .
Pop-Location

Write-Host "==> OK : backend\net-companion.exe"
