# Test navigation improvements
$baseUrl = "http://localhost:8080"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# Login
Write-Host "Logging in..." -ForegroundColor Yellow
$loginBody = @{ username = "admin"; password = "Headstart1" }
$loginResponse = Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded"

# Check maintenance records page
Write-Host "`nChecking Maintenance Records navigation..." -ForegroundColor Yellow
$maintenancePage = Invoke-WebRequest -Uri "$baseUrl/maintenance-records" -WebSession $session

# Check for navigation elements
$hasLogout = $maintenancePage.Content -match "logout|Logout"
$hasDashboard = $maintenancePage.Content -match "Dashboard"
$hasNavbar = $maintenancePage.Content -match "navbar.*glass"

Write-Host "  Has Logout button: $hasLogout" -ForegroundColor $(if($hasLogout){"Green"}else{"Red"})
Write-Host "  Has Dashboard link: $hasDashboard" -ForegroundColor $(if($hasDashboard){"Green"}else{"Red"})
Write-Host "  Has Navigation bar: $hasNavbar" -ForegroundColor $(if($hasNavbar){"Green"}else{"Red"})

# Check hero section
if ($maintenancePage.Content -match "padding:\s*2rem\s*0") {
    Write-Host "  ✓ Hero padding reduced" -ForegroundColor Green
} elseif ($maintenancePage.Content -match "padding:\s*5rem\s*0") {
    Write-Host "  ✗ Hero padding still large" -ForegroundColor Red
}

# Check blur
if ($maintenancePage.Content -match "/\*.*backdrop-filter:\s*blur") {
    Write-Host "  ✓ Blur is disabled" -ForegroundColor Green
} elseif ($maintenancePage.Content -match "backdrop-filter:\s*blur") {
    Write-Host "  ✗ Blur is still active" -ForegroundColor Red
}