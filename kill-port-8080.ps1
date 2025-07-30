# Kill processes using port 8080
Write-Host "Checking for processes using port 8080..." -ForegroundColor Yellow

$connections = Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue

if ($connections) {
    foreach ($conn in $connections) {
        $processId = $conn.OwningProcess
        $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
        
        if ($process) {
            Write-Host "Found process '$($process.Name)' (PID: $processId) using port 8080" -ForegroundColor Red
            
            try {
                Stop-Process -Id $processId -Force
                Write-Host "Successfully killed process $processId" -ForegroundColor Green
            }
            catch {
                Write-Host "Failed to kill process $processId. May need admin privileges." -ForegroundColor Red
            }
        }
    }
    
    # Wait for port to be released
    Start-Sleep -Seconds 2
} else {
    Write-Host "Port 8080 is free!" -ForegroundColor Green
}

Write-Host "`nStarting Fleet Management System on port 8080..." -ForegroundColor Cyan

# Start the application
Set-Location "C:\Users\mycha\hs-bus"
$env:PORT = "8080"
$env:APP_ENV = "development"

& go run .