import requests
import time
import json
from datetime import datetime

# Read cookies from file
cookies = {}
with open('cookies.txt', 'r') as f:
    for line in f:
        if 'session' in line:
            parts = line.strip().split('\t')
            if len(parts) >= 7:
                cookies[parts[5]] = parts[6]

session = requests.Session()
session.cookies.update(cookies)

base_url = "http://localhost:8080"

# All pages to test based on the templates directory
pages_to_test = [
    # Core Dashboard Pages
    "/manager-dashboard",
    "/driver-dashboard", 
    "/ecse-dashboard",
    "/emergency-dashboard",
    "/parent-dashboard",
    "/analytics-dashboard",
    "/budget-dashboard",
    "/progress-dashboard",
    
    # Fleet Management
    "/fleet",
    "/company-fleet",
    "/fleet-vehicles",
    "/vehicle-maintenance",
    "/maintenance-records",
    "/service-records",
    "/maintenance-wizard",
    "/maintenance-alerts",
    
    # Routes & Assignments
    "/assign-routes",
    "/route-assignment-wizard",
    "/gps-tracking",
    "/parent-bus-tracking",
    
    # Students
    "/students",
    "/students-lazy",
    "/add-student-wizard",
    "/import-ecse",
    "/view-ecse-student",
    
    # Reports
    "/monthly-mileage-reports",
    "/driver-reports", 
    "/manager-reports",
    "/view-ecse-reports",
    "/driver-scorecards",
    "/report-builder",
    "/budget-report",
    
    # User Management
    "/approve-users",
    "/manage-users",
    "/change-password",
    "/driver-profile",
    "/notification-preferences",
    "/parent-notification-settings",
    
    # Data Management
    "/import-mileage",
    "/import-data-wizard",
    "/scheduled-exports",
    
    # Fuel & Records
    "/fuel-records",
    "/add-fuel-record",
    "/fuel-analytics",
    
    # System & Help
    "/help-center",
    "/getting-started",
    "/quick-reference",
    "/user-manual",
    "/video-tutorials",
    "/troubleshooting",
    
    # Messaging
    "/messaging",
    
    # Monitoring
    "/db-monitor",
    "/db-pool-monitor",
    
    # Bus Management
    "/add-bus-wizard"
]

print(f"\n{'='*80}")
print(f"COMPREHENSIVE PAGE TESTING - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
print(f"{'='*80}\n")

issues_found = []
working_pages = []
failed_pages = []

for page in pages_to_test:
    print(f"\nTesting: {page}")
    print("-" * 40)
    
    try:
        response = session.get(f"{base_url}{page}", timeout=10)
        status = response.status_code
        content_length = len(response.text)
        
        print(f"  Status Code: {status}")
        print(f"  Content Size: {content_length:,} bytes")
        
        # Check for common issues
        issues = []
        
        # Check status
        if status != 200:
            issues.append(f"HTTP {status} error")
            failed_pages.append(page)
        else:
            working_pages.append(page)
            
        # Check for error indicators in content
        content_lower = response.text.lower()
        
        if "error" in content_lower and "no error" not in content_lower:
            error_count = content_lower.count("error")
            issues.append(f"Contains {error_count} error references")
            
        if "undefined" in content_lower:
            undefined_count = content_lower.count("undefined")
            issues.append(f"Contains {undefined_count} 'undefined' references")
            
        if "null" in response.text and "null" not in page:
            null_count = response.text.count("null")
            if null_count > 5:  # Allow some nulls
                issues.append(f"Contains {null_count} 'null' values")
                
        if "<title></title>" in response.text or "<title> </title>" in response.text:
            issues.append("Empty page title")
            
        if "{{" in response.text and "}}" in response.text:
            template_vars = response.text.count("{{")
            issues.append(f"Unprocessed template variables: {template_vars}")
            
        if "<body></body>" in response.text or len(response.text) < 500:
            issues.append("Page appears empty or minimal content")
            
        # Check for common CSS/JS issues
        if "bootstrap.min.css" not in response.text:
            issues.append("Missing Bootstrap CSS")
            
        if "jquery" not in content_lower and "bootstrap.bundle" not in response.text:
            issues.append("Missing jQuery/Bootstrap JS")
            
        # Check for console error scripts (common debugging left in)
        if "console.error" in response.text:
            issues.append("Contains console.error statements")
            
        if issues:
            print(f"  ISSUES FOUND:")
            for issue in issues:
                print(f"      - {issue}")
            issues_found.append({"page": page, "issues": issues})
        else:
            print(f"  [OK] Page appears OK")
            
    except requests.exceptions.Timeout:
        print(f"  [TIMEOUT] Page took too long to load")
        issues_found.append({"page": page, "issues": ["Timeout - slow loading"]})
        failed_pages.append(page)
    except Exception as e:
        print(f"  [ERROR] {str(e)}")
        issues_found.append({"page": page, "issues": [f"Error: {str(e)}"]})
        failed_pages.append(page)
        
    time.sleep(0.5)  # Don't overwhelm the server

# Summary Report
print(f"\n{'='*80}")
print("TESTING SUMMARY")
print(f"{'='*80}\n")

print(f"Total Pages Tested: {len(pages_to_test)}")
print(f"Working Pages: {len(working_pages)}")
print(f"Failed Pages: {len(failed_pages)}")
print(f"Pages with Issues: {len(issues_found)}")

if failed_pages:
    print(f"\n[FAILED] PAGES ({len(failed_pages)}):")
    for page in failed_pages:
        print(f"  - {page}")

if issues_found:
    print(f"\n[ISSUES] PAGES WITH ISSUES ({len(issues_found)}):")
    for item in issues_found:
        print(f"\n  {item['page']}:")
        for issue in item['issues']:
            print(f"    - {issue}")

# Save detailed report
with open('page_test_report.json', 'w') as f:
    json.dump({
        'timestamp': datetime.now().isoformat(),
        'total_tested': len(pages_to_test),
        'working': working_pages,
        'failed': failed_pages,
        'issues': issues_found
    }, f, indent=2)

print(f"\n[REPORT] Detailed report saved to page_test_report.json")
print(f"\n{'='*80}")