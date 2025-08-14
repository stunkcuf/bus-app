import requests
import time

# Test key pages after fixes
test_pages = [
    "/manager-dashboard",
    "/driver-dashboard", 
    "/analytics-dashboard",
    "/fleet-vehicles",
    "/students",
    "/monthly-mileage-reports"
]

# Read cookies
cookies = {}
with open('cookies.txt', 'r') as f:
    for line in f:
        if 'session' in line:
            parts = line.strip().split('\t')
            if len(parts) >= 7:
                cookies[parts[5]] = parts[6]

session = requests.Session()
session.cookies.update(cookies)

print("FINAL TEST RESULTS AFTER FIXES")
print("=" * 50)

all_good = True
for page in test_pages:
    response = session.get(f"http://localhost:8080{page}")
    status = response.status_code
    
    # Check for issues
    has_jquery = 'jquery' in response.text.lower()
    has_bootstrap = 'bootstrap' in response.text.lower()
    has_console_error = 'console.error' in response.text
    
    status_icon = "OK" if status == 200 else "FAIL"
    jquery_icon = "OK" if has_jquery else "MISSING"
    bootstrap_icon = "OK" if has_bootstrap else "MISSING"
    console_icon = "OK" if not has_console_error else "FOUND"
    
    print(f"\n{page}:")
    print(f"  Status: {status} {status_icon}")
    print(f"  jQuery: {jquery_icon}")
    print(f"  Bootstrap: {bootstrap_icon}")
    print(f"  No console.error: {console_icon}")
    
    if status != 200 or not has_jquery or not has_bootstrap or has_console_error:
        all_good = False

print("\n" + "=" * 50)
if all_good:
    print("ALL TESTS PASSED!")
else:
    print("Some issues remain - see details above")