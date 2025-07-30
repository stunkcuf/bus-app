# Test specific page appearance
$baseUrl = "http://localhost:8080"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# Login
$loginBody = @{ username = "admin"; password = "Headstart1" }
Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded" | Out-Null

# Check specific pages
$pagesToCheck = @(
    @{Name="Maintenance Records"; Url="/maintenance-records"},
    @{Name="Service Records"; Url="/service-records"},
    @{Name="Monthly Mileage"; Url="/monthly-mileage-reports"},
    @{Name="Fleet"; Url="/fleet"}
)

foreach ($page in $pagesToCheck) {
    Write-Host "Checking $($page.Name)..." -ForegroundColor Yellow
    
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl$($page.Url)" -WebSession $session -ErrorAction Stop
        
        # Save to file for inspection
        $filename = "$($page.Name -replace ' ','_')_inspection.html"
        $response.Content | Out-File -FilePath $filename -Encoding UTF8
        Write-Host "  Saved to $filename" -ForegroundColor Green
        
        # Quick checks
        if ($response.Content -match '<body[^>]*style="[^"]*background:\s*white') {
            Write-Host "  ⚠ Body has white background!" -ForegroundColor Red
        }
        
        if ($response.Content -match 'class="[^"]*bg-white[^"]*"[^>]*>(?!.*navbar)') {
            Write-Host "  ⚠ Found bg-white class (not navbar)" -ForegroundColor Yellow
        }
        
        # Check table count
        $tableCount = ([regex]::Matches($response.Content, '<table[^>]*>')).Count
        Write-Host "  Tables found: $tableCount" -ForegroundColor Gray
        
    } catch {
        Write-Host "  ERROR: $_" -ForegroundColor Red
    }
    Write-Host ""
}

# Open one page for visual inspection
Write-Host "Opening Maintenance Records in browser..." -ForegroundColor Cyan
Start-Process "http://localhost:8080/maintenance-records"