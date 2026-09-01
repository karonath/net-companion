# Build Net-Companion Lite : frontend Vite -> go:embed -> binaire unique.
$root = $PSScriptRoot
$go = "C:\Program Files\Go\bin\go.exe"

# npm écrit des warnings sur stderr : on ne veut PAS que PowerShell les traite
# comme des erreurs terminantes. On vérifie explicitement les codes de sortie.
$ErrorActionPreference = "Continue"

Write-Host "==> Build frontend (Vite)"
Push-Location (Join-Path $root "frontend")
& npm install
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "npm install a échoué ($LASTEXITCODE)" }
& npm run build
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "npm run build a échoué ($LASTEXITCODE)" }
Pop-Location

Write-Host "==> Build backend (go:embed)"
Push-Location (Join-Path $root "backend")
& $go build -o net-companion.exe .
$code = $LASTEXITCODE
Pop-Location
if ($code -ne 0) { throw "go build a échoué ($code)" }

# Dépose le binaire prêt-à-copier (clé USB) dans release\
$release = Join-Path $root "release"
New-Item -ItemType Directory -Force -Path $release | Out-Null
Copy-Item (Join-Path $root "backend\net-companion.exe") (Join-Path $release "net-companion.exe") -Force

Write-Host "==> OK : backend\net-companion.exe"
Write-Host "==> Prêt pour clé USB : release\net-companion.exe"
