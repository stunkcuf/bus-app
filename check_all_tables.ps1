# Comprehensive test of all pages with data tables
$baseUrl = "http://localhost:8080"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# Login first
Write-Host "Logging in..." -ForegroundColor Yellow
$loginBody = @{
    username = "admin"
    password = "Headstart1"
}
$loginResponse = Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded"
Write-Host "Login successful`n" -ForegroundColor Green

# List of pages with data tables to check
$pagesToCheck = @(
    @{Name="Fleet Overview"; Url="/fleet"; TableClass="fleet-table"},
    @{Name="Company Fleet"; Url="/company-fleet"; TableClass="vehicle-table"},
    @{Name="Maintenance Records"; Url="/maintenance-records"; TableClass="table"},
    @{Name="Service Records"; Url="/service-records"; TableClass="table"},
    @{Name="Monthly Mileage Reports"; Url="/monthly-mileage-reports"; TableClass="table"},
    @{Name="Driver Management"; Url="/manage-drivers"; TableClass="table"},
    @{Name="User Management"; Url="/manage-users"; TableClass="table"},
    @{Name="Approve Users"; Url="/approve-users"; TableClass="table"},
    @{Name="Assign Routes"; Url="/assign-routes"; TableClass="table"},
    @{Name="Students List"; Url="/students"; TableClass="table"},
    @{Name="ECSE Students"; Url="/ecse-students"; TableClass="table"},
    @{Name="Route Assignments"; Url="/route-assignments"; TableClass="table"},
    @{Name="Driver Logs"; Url="/driver-logs"; TableClass="table"},
    @{Name="Fuel Records"; Url="/fuel-records"; TableClass="table"}
)

foreach ($page in $pagesToCheck) {
    Write-Host "===========================================" -ForegroundColor Cyan
    Write-Host "Checking: $($page.Name) ($($page.Url))" -ForegroundColor Yellow
    Write-Host "===========================================" -ForegroundColor Cyan
    
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl$($page.Url)" -WebSession $session -ErrorAction Stop
        $content = $response.Content
        
        # Check if page loaded
        Write-Host "✓ Page loaded successfully" -ForegroundColor Green
        
        # Check for tables
        $tableCount = ([regex]::Matches($content, '<table[^>]*>')).Count
        Write-Host "  Tables found: $tableCount" -ForegroundColor White
        
        # Check for specific table issues
        
        # 1. Check table visibility CSS
        if ($content -match 'visibility:\s*hidden' -or $content -match 'display:\s*none.*table') {
            Write-Host "  ⚠ WARNING: Table might be hidden via CSS" -ForegroundColor Red
        }
        
        # 2. Check text color issues (white on white, etc)
        if ($content -match 'color:\s*(white|#fff|#ffffff).*background.*white' -or 
            $content -match 'color:\s*(white|#fff|#ffffff).*glass-card') {
            Write-Host "  ⚠ WARNING: Potential white text on white background" -ForegroundColor Red
        }
        
        # 3. Check for backdrop-filter blur
        if ($content -match 'backdrop-filter:\s*blur\(' -and $content -notmatch '/\*.*backdrop-filter:\s*blur') {
            Write-Host "  ⚠ WARNING: Active backdrop blur detected" -ForegroundColor Red
        }
        
        # 4. Check for table rows
        $rowCount = ([regex]::Matches($content, '<tr[^>]*>')).Count - 1  # Subtract header row
        Write-Host "  Table rows: $rowCount" -ForegroundColor White
        
        # 5. Check for buttons in tables
        $buttonCount = ([regex]::Matches($content, '<button[^>]*>|<a[^>]*class="[^"]*btn[^"]*"')).Count
        Write-Host "  Buttons found: $buttonCount" -ForegroundColor White
        
        # 6. Check for specific CSS classes that might cause issues
        if ($content -match 'text-(white|light).*bg-(white|light)') {
            Write-Host "  ⚠ WARNING: Potential text visibility issue" -ForegroundColor Red
        }
        
        # 7. Check table styling
        if ($content -match '<table[^>]*class="[^"]*glass[^"]*"') {
            Write-Host "  ℹ Glass effect applied to table" -ForegroundColor Cyan
        }
        
        # 8. Check for empty tables
        if ($tableCount -gt 0 -and $rowCount -eq 0) {
            Write-Host "  ⚠ WARNING: Table exists but has no data rows" -ForegroundColor Yellow
        }
        
        # Save problematic pages for inspection
        if ($content -match 'visibility:\s*hidden|display:\s*none.*table|color:\s*(white|#fff).*background.*white') {
            $filename = "$($page.Name -replace ' ', '_')_inspection.html"
            $content | Out-File -FilePath $filename
            Write-Host "  📄 Saved to $filename for inspection" -ForegroundColor Gray
        }
        
    } catch {
        Write-Host "✗ ERROR loading page: $_" -ForegroundColor Red
    }
    
    Write-Host ""
}

Write-Host "`nTest completed!" -ForegroundColor Cyan