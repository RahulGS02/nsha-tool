# NSHA Build Script for Windows PowerShell
# Builds executables for multiple platforms

$ErrorActionPreference = "Stop"

$VERSION = "1.0.0"
$APP_NAME = "nsha"
$BUILD_DIR = "build"

Write-Host "╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║           Building NSHA - Null SHA Fixer                  ║" -ForegroundColor Cyan
Write-Host "║                   Version: $VERSION                        ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Clean build directory
Write-Host "[CLEAN] Cleaning build directory..." -ForegroundColor Yellow
if (Test-Path $BUILD_DIR) {
    Remove-Item -Recurse -Force $BUILD_DIR
}
New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null

# Build flags for smaller binaries
$LDFLAGS = "-s -w -X main.Version=$VERSION"

Write-Host ""
Write-Host "[BUILD] Building executables..." -ForegroundColor Yellow
Write-Host ""

# Windows AMD64
Write-Host "  [BUILD] Building for Windows (amd64)..." -ForegroundColor Green
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="$LDFLAGS" -o "$BUILD_DIR\${APP_NAME}-windows-amd64.exe"
Write-Host "     [OK] $BUILD_DIR\${APP_NAME}-windows-amd64.exe" -ForegroundColor Green

# Windows ARM64
Write-Host "  [BUILD] Building for Windows (arm64)..." -ForegroundColor Green
$env:GOOS = "windows"
$env:GOARCH = "arm64"
go build -ldflags="$LDFLAGS" -o "$BUILD_DIR\${APP_NAME}-windows-arm64.exe"
Write-Host "     [OK] $BUILD_DIR\${APP_NAME}-windows-arm64.exe" -ForegroundColor Green

# Linux AMD64
Write-Host "  [BUILD] Building for Linux (amd64)..." -ForegroundColor Green
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="$LDFLAGS" -o "$BUILD_DIR\${APP_NAME}-linux-amd64"
Write-Host "     [OK] $BUILD_DIR\${APP_NAME}-linux-amd64" -ForegroundColor Green

# Linux ARM64
Write-Host "  [BUILD] Building for Linux (arm64)..." -ForegroundColor Green
$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -ldflags="$LDFLAGS" -o "$BUILD_DIR\${APP_NAME}-linux-arm64"
Write-Host "     [OK] $BUILD_DIR\${APP_NAME}-linux-arm64" -ForegroundColor Green

# macOS AMD64
Write-Host "  [BUILD] Building for macOS (amd64)..." -ForegroundColor Green
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -ldflags="$LDFLAGS" -o "$BUILD_DIR\${APP_NAME}-darwin-amd64"
Write-Host "     [OK] $BUILD_DIR\${APP_NAME}-darwin-amd64" -ForegroundColor Green

# macOS ARM64 (Apple Silicon)
Write-Host "  [BUILD] Building for macOS (arm64)..." -ForegroundColor Green
$env:GOOS = "darwin"
$env:GOARCH = "arm64"
go build -ldflags="$LDFLAGS" -o "$BUILD_DIR\${APP_NAME}-darwin-arm64"
Write-Host "     [OK] $BUILD_DIR\${APP_NAME}-darwin-arm64" -ForegroundColor Green

Write-Host ""
Write-Host "[SUCCESS] Build complete!" -ForegroundColor Green
Write-Host ""
Write-Host "[INFO] Executables are in the '$BUILD_DIR' directory:" -ForegroundColor Cyan
Get-ChildItem $BUILD_DIR | Format-Table Name, Length -AutoSize
Write-Host ""
Write-Host "[INFO] To install locally, copy the appropriate binary to your PATH" -ForegroundColor Yellow
Write-Host "   Example: Copy-Item $BUILD_DIR\${APP_NAME}-windows-amd64.exe C:\Windows\System32\nsha.exe" -ForegroundColor Yellow

