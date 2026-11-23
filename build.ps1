param (
    [ValidateSet("dev", "test", "prod")]
    [string]$Env = "dev",

    [ValidateSet("windows", "linux", "all")]
    [string]$Target = "windows",

    [switch]$Test,
    [switch]$Frontend,
    [string]$Version = "0.1.0-alpha"
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  Tekton DevOps Platform Multi-Environment Build Tool" -ForegroundColor Cyan
Write-Host "  Target Environment : $Env" -ForegroundColor Yellow
Write-Host "  Target OS          : $Target" -ForegroundColor Yellow
Write-Host "  Version            : $Version" -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor Cyan

# Optional: Run Frontend Build
if ($Frontend) {
    Write-Host "`n[1/3] Building Frontend React App..." -ForegroundColor Green
    Push-Location (Join-Path $ScriptDir "frontend")
    try {
        npm run build
    } finally {
        Pop-Location
    }
}

# Optional: Run Tests
if ($Test) {
    Write-Host "`n[*] Running Backend Tests..." -ForegroundColor Green
    Push-Location (Join-Path $ScriptDir "backend")
    try {
        go test -tags $Env -v ./...
    } finally {
        Pop-Location
    }
}

# Build Backend
Write-Host "`n[2/3] Compiling Backend with Environment: '$Env'..." -ForegroundColor Green
$BackendDir = Join-Path $ScriptDir "backend"
$BinDir = Join-Path $BackendDir "bin"
if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
}

$LdFlags = "-s -w -X main.version=$Version -X devops-platform/internal/config.BuildEnv=$Env"

$BuildTag = $Env
if ($Env -eq "test") {
    $BuildTag = "testenv"
}

Push-Location $BackendDir
try {
    $env:CGO_ENABLED = "0"

    if ($Target -eq "windows" -or $Target -eq "all") {
        Write-Host "  -> Building Windows amd64 binary (platform-windows-amd64.exe)..." -ForegroundColor Gray
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        go build -tags $BuildTag -ldflags $LdFlags -o "bin/platform-windows-amd64.exe" ./cmd/platform
        Copy-Item -Path "bin/platform-windows-amd64.exe" -Destination "bin/platform.exe" -Force
        Write-Host "     [OK] Built: backend/bin/platform.exe" -ForegroundColor Green
    }

    if ($Target -eq "linux" -or $Target -eq "all") {
        Write-Host "  -> Cross-compiling Linux amd64 binary (platform-linux-amd64)..." -ForegroundColor Gray
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        go build -tags $BuildTag -ldflags $LdFlags -o "bin/platform-linux-amd64" ./cmd/platform
        Write-Host "     [OK] Built: backend/bin/platform-linux-amd64" -ForegroundColor Green
    }
} finally {
    Pop-Location
}

Write-Host "`n==========================================================" -ForegroundColor Cyan
Write-Host "  Build completed successfully for environment: '$Env'!" -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Cyan
