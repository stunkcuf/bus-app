# Detailed test of route assignment wizard

$baseUrl = "http://localhost:8080"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# Login
$loginBody = @{
    username = "admin"
    password = "Headstart1"
}

Write-Host "Logging in..." -ForegroundColor Yellow
$loginResponse = Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded"

# Get route assignment wizard page
Write-Host "`nFetching route assignment wizard page..." -ForegroundColor Yellow
$wizardPage = Invoke-WebRequest -Uri "$baseUrl/route-assignment-wizard" -WebSession $session

# Check for driver select element
Write-Host "`nChecking driver dropdown..." -ForegroundColor Yellow
if ($wizardPage.Content -match '<select[^>]*id="driver"[^>]*>([\s\S]*?)</select>') {
    $selectContent = $matches[1]
    Write-Host "Driver select found!" -ForegroundColor Green
    
    # Count options (excluding the placeholder)
    $optionMatches = [regex]::Matches($selectContent, '<option[^>]*value="([^"]+)"')
    $driverCount = 0
    
    foreach ($match in $optionMatches) {
        $value = $match.Groups[1].Value
        if ($value -ne "") {
            $driverCount++
            Write-Host "  - Found driver option: $value" -ForegroundColor Cyan
        }
    }
    
    Write-Host "`nTotal drivers in dropdown: $driverCount" -ForegroundColor Yellow
    
    # Also check if there's any .Data.Drivers reference
    if ($wizardPage.Content -match '\.Data\.Drivers') {
        Write-Host "Template still has .Data.Drivers reference" -ForegroundColor Red
    }
} else {
    Write-Host "Driver select element not found!" -ForegroundColor Red
}

# Save page content for inspection
$wizardPage.Content | Out-File -FilePath "route_wizard_page.html"
Write-Host "`nPage content saved to route_wizard_page.html for inspection" -ForegroundColor Gray