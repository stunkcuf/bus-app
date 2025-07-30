# Test Fleet Management System fixes

$baseUrl = "http://localhost:8080"

# Create session and login
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

Write-Host "1. Testing login..." -ForegroundColor Yellow
$loginBody = @{
    username = "admin"
    password = "Headstart1"
}

try {
    $loginResponse = Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded"
    Write-Host "   Login successful" -ForegroundColor Green
} catch {
    Write-Host "   Login failed: $_" -ForegroundColor Red
    exit 1
}

# Test fleet page
Write-Host "`n2. Testing fleet page edit buttons..." -ForegroundColor Yellow
try {
    $fleetPage = Invoke-WebRequest -Uri "$baseUrl/fleet" -WebSession $session
    $editButtonCount = ([regex]::Matches($fleetPage.Content, "editBus\(")).Count
    if ($editButtonCount -gt 0) {
        Write-Host "   Found $editButtonCount edit buttons" -ForegroundColor Green
        # Check if editBus function exists
        if ($fleetPage.Content -match "function editBus\(busId\)") {
            Write-Host "   editBus function found" -ForegroundColor Green
        } else {
            Write-Host "   editBus function NOT found" -ForegroundColor Red
        }
    } else {
        Write-Host "   No edit buttons found" -ForegroundColor Red
    }
} catch {
    Write-Host "   Fleet page error: $_" -ForegroundColor Red
}

# Test company fleet dropdown styling
Write-Host "`n3. Testing company fleet dropdown overlaps..." -ForegroundColor Yellow
try {
    $companyFleetPage = Invoke-WebRequest -Uri "$baseUrl/company-fleet" -WebSession $session
    if ($companyFleetPage.Content -match "z-index:\s*1050\s*!important") {
        Write-Host "   Dropdown z-index fix found" -ForegroundColor Green
    } else {
        Write-Host "   Dropdown z-index fix NOT found" -ForegroundColor Red
    }
} catch {
    Write-Host "   Company fleet page error: $_" -ForegroundColor Red
}

# Test fleet vehicle edit
Write-Host "`n4. Testing fleet vehicle edit pages..." -ForegroundColor Yellow
try {
    # Try to access a fleet vehicle edit page
    $vehicleEditPage = Invoke-WebRequest -Uri "$baseUrl/fleet-vehicle/edit/11" -WebSession $session
    if ($vehicleEditPage.Content -notmatch "Vehicle not found") {
        Write-Host "   Fleet vehicle edit page loads successfully" -ForegroundColor Green
    } else {
        Write-Host "   Fleet vehicle edit shows 'Vehicle not found'" -ForegroundColor Red
    }
} catch {
    Write-Host "   Fleet vehicle edit error: $_" -ForegroundColor Red
}

# Test route assignment wizard
Write-Host "`n5. Testing route assignment wizard..." -ForegroundColor Yellow
try {
    $routeWizardPage = Invoke-WebRequest -Uri "$baseUrl/route-assignment-wizard" -WebSession $session
    # Check if drivers dropdown has options (excluding empty option)
    $driverOptions = ([regex]::Matches($routeWizardPage.Content, '<option[^>]*value="([^"]+)"[^>]*>')).Count - 1
    if ($driverOptions -gt 0) {
        Write-Host "   Driver dropdown has $driverOptions drivers" -ForegroundColor Green
    } else {
        Write-Host "   Driver dropdown is empty" -ForegroundColor Red
    }
} catch {
    Write-Host "   Route assignment wizard error: $_" -ForegroundColor Red
}

# Test monthly mileage reports
Write-Host "`n6. Testing monthly mileage reports..." -ForegroundColor Yellow
try {
    $mileagePage = Invoke-WebRequest -Uri "$baseUrl/monthly-mileage-reports" -WebSession $session
    # Check if backdrop-filter is commented out
    if ($mileagePage.Content -match "/\*\s*backdrop-filter:\s*blur") {
        Write-Host "   Backdrop blur is disabled (page should be clear)" -ForegroundColor Green
    } else {
        Write-Host "   Backdrop blur might still be active" -ForegroundColor Yellow
    }
} catch {
    Write-Host "   Monthly mileage reports error: $_" -ForegroundColor Red
}

Write-Host "`nTest completed!" -ForegroundColor Cyan