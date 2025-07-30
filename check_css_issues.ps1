# Check for CSS issues in table pages
$baseUrl = "http://localhost:8080"
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

# Login
$loginBody = @{ username = "admin"; password = "Headstart1" }
Invoke-WebRequest -Uri "$baseUrl/" -Method POST -Body $loginBody -WebSession $session -ContentType "application/x-www-form-urlencoded" | Out-Null

# Check a few key pages
$testPages = @(
    @{Name="Fleet"; Url="/fleet"},
    @{Name="Maintenance Records"; Url="/maintenance-records"},
    @{Name="Monthly Mileage"; Url="/monthly-mileage-reports"}
)

foreach ($page in $testPages) {
    Write-Host "`nAnalyzing $($page.Name)..." -ForegroundColor Yellow
    $response = Invoke-WebRequest -Uri "$baseUrl$($page.Url)" -WebSession $session
    $content = $response.Content
    
    # Extract inline styles
    Write-Host "Checking for problematic CSS..." -ForegroundColor Cyan
    
    # Check table styling
    if ($content -match '<style[^>]*>([\s\S]*?)</style>') {
        $styles = $matches[1]
        
        # Check for white on white
        if ($styles -match 'table.*\{[^}]*color:\s*white' -and $styles -match 'background:\s*(white|#fff)') {
            Write-Host "  ⚠ White text on white background detected!" -ForegroundColor Red
        }
        
        # Check for glass card issues
        if ($styles -match 'glass-card.*\{[^}]*background:\s*rgba\(255,\s*255,\s*255,\s*0\.[0-9]+\)') {
            Write-Host "  ℹ Glass card with transparency detected" -ForegroundColor Cyan
            
            # Check if text color is set appropriately
            if ($styles -notmatch 'glass-card.*\{[^}]*color:\s*(black|#000|#[0-9a-f]{3,6})') {
                Write-Host "  ⚠ Glass card may have text visibility issues" -ForegroundColor Yellow
            }
        }
        
        # Check table specific styling
        if ($styles -match '\.table[^{]*\{([^}]*)\}' -or $styles -match 'table[^{]*\{([^}]*)\}') {
            $tableStyles = $matches[1]
            Write-Host "  Table styles found: $($tableStyles.Trim())" -ForegroundColor Gray
        }
    }
    
    # Check for Bootstrap dark theme conflicts
    if ($content -match 'table-dark' -and $content -match 'text-white.*bg-white') {
        Write-Host "  ⚠ Conflicting dark/light theme classes" -ForegroundColor Red
    }
    
    # Save page for manual inspection if issues found
    if ($content -match 'color:\s*white.*glass-card|table.*color:\s*white.*background.*white') {
        $filename = "inspection_$($page.Name -replace ' ','_').html"
        $content | Out-File -FilePath $filename
        Write-Host "  📄 Saved to $filename for inspection" -ForegroundColor Gray
    }
}

Write-Host "`nDone!" -ForegroundColor Green