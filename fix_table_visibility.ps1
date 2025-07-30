# Script to identify and document table visibility issues

$templates = @(
    "fleet.html",
    "company_fleet.html", 
    "maintenance_records.html",
    "service_records.html",
    "monthly_mileage_reports.html",
    "manage_users.html",
    "approve_users.html",
    "assign_routes.html",
    "students.html",
    "ecse_dashboard.html",
    "fuel_records.html",
    "driver_logs.html"
)

$issues = @()

foreach ($template in $templates) {
    $path = "templates\$template"
    if (Test-Path $path) {
        $content = Get-Content $path -Raw
        
        Write-Host "Checking $template..." -ForegroundColor Yellow
        
        # Check for potential issues
        $templateIssues = @()
        
        # 1. Check for white backgrounds in glass cards or tables
        if ($content -match "glass-card.*\{[^}]*background:\s*(white|#fff|#ffffff)") {
            $templateIssues += "Glass card with white background"
        }
        
        # 2. Check for missing color declarations in tables
        if ($content -match "<table" -and $content -notmatch "\.table.*\{[^}]*color:\s*white") {
            if ($content -notmatch "table.*style.*color.*white") {
                $templateIssues += "Table might be missing white color declaration"
            }
        }
        
        # 3. Check for active backdrop filters
        if ($content -match "backdrop-filter:\s*blur\([^)]+\)" -and $content -notmatch "/\*.*backdrop-filter:\s*blur") {
            $templateIssues += "Active backdrop blur detected"
        }
        
        # 4. Check for conflicting styles
        if ($content -match "text-dark.*glass-card|bg-light.*glass-card") {
            $templateIssues += "Conflicting Bootstrap classes with glass theme"
        }
        
        if ($templateIssues.Count -gt 0) {
            Write-Host "  Issues found:" -ForegroundColor Red
            foreach ($issue in $templateIssues) {
                Write-Host "    - $issue" -ForegroundColor Yellow
                $issues += "$template : $issue"
            }
        } else {
            Write-Host "  ✓ No issues found" -ForegroundColor Green
        }
    }
}

Write-Host "`n=== SUMMARY ===" -ForegroundColor Cyan
if ($issues.Count -gt 0) {
    Write-Host "Total issues found: $($issues.Count)" -ForegroundColor Red
    foreach ($issue in $issues) {
        Write-Host "  $issue" -ForegroundColor Yellow
    }
} else {
    Write-Host "No visibility issues found!" -ForegroundColor Green
}