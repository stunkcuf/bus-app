# Script to identify templates that need the modern glass design

$templatesPath = "templates"
$goodTemplates = @("fleet.html", "company_fleet.html", "manage_users.html")
$excludeTemplates = @("login.html", "register.html", "error.html", "components/*")

Write-Host "Analyzing template consistency..." -ForegroundColor Cyan
Write-Host "Good templates (reference design):" -ForegroundColor Green
$goodTemplates | ForEach-Object { Write-Host "  - $_" -ForegroundColor Green }

# Get all template files
$allTemplates = Get-ChildItem -Path $templatesPath -Filter "*.html" -File | Where-Object {
    $_.Name -notin $goodTemplates -and 
    $_.Name -notin @("login.html", "register.html", "error.html") -and
    $_.Name -notmatch "^components_"
}

Write-Host "`nTemplates needing update:" -ForegroundColor Yellow
$needsUpdate = @()

foreach ($template in $allTemplates) {
    $content = Get-Content $template.FullName -Raw
    
    # Check for missing design elements
    $hasOrbs = $content -match '<div class="orb'
    $hasGlassCard = $content -match 'glass-card'
    $hasHeroSection = $content -match 'hero-section'
    $hasAnimatedBg = $content -match 'body::before.*backgroundShift'
    $hasNavbarGlass = $content -match 'navbar-glass'
    
    if (-not ($hasOrbs -and $hasGlassCard -and $hasAnimatedBg)) {
        Write-Host "  - $($template.Name)" -ForegroundColor Yellow
        
        if (-not $hasOrbs) { Write-Host "    Missing: Floating orbs" -ForegroundColor Red }
        if (-not $hasGlassCard) { Write-Host "    Missing: Glass card styling" -ForegroundColor Red }
        if (-not $hasAnimatedBg) { Write-Host "    Missing: Animated background" -ForegroundColor Red }
        if (-not $hasNavbarGlass) { Write-Host "    Missing: Glass navbar" -ForegroundColor Red }
        
        $needsUpdate += $template.Name
    }
}

Write-Host "`nTotal templates needing update: $($needsUpdate.Count)" -ForegroundColor Cyan

# Key templates to update first
$priorityTemplates = @(
    "maintenance_records.html",
    "service_records.html", 
    "monthly_mileage_reports.html",
    "students.html",
    "driver_logs.html",
    "assign_routes.html",
    "approve_users.html"
)

Write-Host "`nPriority templates to update first:" -ForegroundColor Magenta
$priorityTemplates | Where-Object { $_ -in $needsUpdate } | ForEach-Object {
    Write-Host "  - $_" -ForegroundColor Magenta
}