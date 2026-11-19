# Script generate protobuf stubs for Windows PowerShell
$ErrorActionPreference = "Stop"

Write-Host "==> Checking protoc compiler..." -ForegroundColor Cyan
if (-not (Get-Command protoc -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] 'protoc' compiler is not found in your PATH." -ForegroundColor Red
    Write-Host "Please download protoc from: https://github.com/protocolbuffers/protobuf/releases" -ForegroundColor Yellow
    Write-Host "Extract 'protoc.exe' into '$HOME\go\bin' or another folder in your PATH." -ForegroundColor Yellow
    exit 1
}

Write-Host "==> Ensuring output directories exist..." -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path "pb/common", "pb/inventory" | Out-Null

Write-Host "==> Generating Go protobuf files..." -ForegroundColor Cyan
protoc --proto_path=proto `
       --go_out=paths=source_relative:pb `
       --go-grpc_out=paths=source_relative:pb `
       proto/common/common.proto proto/inventory/inventory_service.proto

if ($LASTEXITCODE -eq 0) {
    Write-Host "[SUCCESS] Protobuf files generated successfully in ./pb/" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Protoc execution failed with exit code $LASTEXITCODE" -ForegroundColor Red
    exit $LASTEXITCODE
}
