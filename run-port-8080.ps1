# Fleet Management System - Run on Port 8080
$env:PORT = "8080"
$env:APP_ENV = "development"

Write-Host "===========================================" -ForegroundColor Green
Write-Host "  FLEET MANAGEMENT SYSTEM" -ForegroundColor Green
Write-Host "  Starting on Port 8080..." -ForegroundColor Green
Write-Host "===========================================" -ForegroundColor Green
Write-Host ""

# Start in a new window
Start-Process cmd -ArgumentList "/k", "cd /d C:\Users\mycha\hs-bus && set PORT=8080 && set APP_ENV=development && echo Starting server on port 8080... && go run ."