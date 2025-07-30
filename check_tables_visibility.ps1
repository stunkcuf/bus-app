# Check all pages with tables for visibility issues
$baseUrl = "http://localhost:8080"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# Login
Write-Host "Logging in..." -ForegroundColor Yellow
$loginBody = @{
    username = "admin"
    password = "Headstart1"
}
Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded" | Out-Null
Write-Host "Login successful`n" -ForegroundColor Green

# Pages to check
$pages = @(
    "/fleet",
    "/company-fleet", 
    "/maintenance-records",
    "/service-records",
    "/monthly-mileage-reports",
    "/manage-drivers",
    "/manage-users",
    "/approve-users",
    "/assign-routes",
    "/students",
    "/ecse-students",
    "/route-assignments",
    "/driver-logs",
    "/fuel-records"
)

$issuesFound = @()

foreach ($pageUrl in $pages) {
    Write-Host "Checking $pageUrl..." -ForegroundColor Yellow
    
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl$pageUrl" -WebSession $session -ErrorAction Stop
        $content = $response.Content
        
        # Check for common visibility issues
        $issues = @()
        
        # Tables found
        $tableCount = ([regex]::Matches($content, '<table[^>]*>')).Count
        if ($tableCount -gt 0) {
            Write-Host "  Tables: $tableCount" -ForegroundColor Gray
            
            # Check for hidden tables
            if ($content -match 'table[^{]*\{[^}]*display:\s*none') {
                $issues += "Table hidden with display:none"
            }
            
            # Check for white text on white background
            if ($content -match 'color:\s*(white|#fff|#ffffff)[^}]*\}[^{]*table' -or
                $content -match 'table[^{]*\{[^}]*color:\s*(white|#fff|#ffffff)') {
                $issues += "Possible white text in table"
            }
            
            # Check table rows
            $rowCount = ([regex]::Matches($content, '<tbody[^>]*>.*?<tr[^>]*>', "Singleline")).Count
            Write-Host "  Data rows: $rowCount" -ForegroundColor Gray
            
            # Check for glass effects causing readability issues
            if ($content -match 'glass-card.*table|table.*glass') {
                Write-Host "  Has glass effect" -ForegroundColor Cyan
            }
        }
        
        # Report issues
        if ($issues.Count -gt 0) {
            Write-Host "  ⚠ ISSUES FOUND:" -ForegroundColor Red
            foreach ($issue in $issues) {
                Write-Host "    - $issue" -ForegroundColor Red
                $issuesFound += "$pageUrl : $issue"
            }
        } else {
            Write-Host "  ✓ No visibility issues" -ForegroundColor Green
        }
        
    } catch {
        Write-Host "  ✗ ERROR: $_" -ForegroundColor Red
    }
    Write-Host ""
}

if ($issuesFound.Count -gt 0) {
    Write-Host "`nSUMMARY OF ISSUES:" -ForegroundColor Red
    foreach ($issue in $issuesFound) {
        Write-Host "  $issue" -ForegroundColor Yellow
    }
} else {
    Write-Host "`nNo visibility issues found!" -ForegroundColor Green
}