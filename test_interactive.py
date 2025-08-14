import requests
import json

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
base_url = "http://localhost:8080"

print("TESTING INTERACTIVE FEATURES AND AJAX ENDPOINTS")
print("=" * 60)

# Test AJAX endpoints
ajax_endpoints = [
    ("/api/dashboard/analytics", "GET", None),
    ("/api/dashboard/fleet-status", "GET", None),
    ("/api/dashboard/maintenance-alerts", "GET", None),
    ("/api/notifications", "GET", None),
    ("/api/search/students", "GET", {"q": "test"}),
    ("/api/fleet/summary", "GET", None),
]

print("\n1. TESTING AJAX ENDPOINTS:")
print("-" * 40)

for endpoint, method, params in ajax_endpoints:
    try:
        if method == "GET":
            response = session.get(f"{base_url}{endpoint}", params=params, timeout=5)
        else:
            response = session.post(f"{base_url}{endpoint}", json=params, timeout=5)
        
        status = response.status_code
        content_type = response.headers.get('Content-Type', '')
        
        # Check if JSON response
        is_json = 'application/json' in content_type
        if is_json:
            try:
                data = response.json()
                has_data = bool(data)
            except:
                has_data = False
        else:
            has_data = len(response.text) > 0
        
        status_text = "OK" if status == 200 else f"ERROR ({status})"
        data_text = "Has Data" if has_data else "Empty"
        
        print(f"{endpoint}: {status_text} - {data_text}")
        
    except requests.exceptions.Timeout:
        print(f"{endpoint}: TIMEOUT")
    except Exception as e:
        print(f"{endpoint}: ERROR - {str(e)}")

# Test form pages
print("\n2. TESTING FORM PAGES:")
print("-" * 40)

form_pages = [
    "/add-student-wizard",
    "/add-bus-wizard", 
    "/change-password",
    "/maintenance-wizard",
    "/route-assignment-wizard"
]

for page in form_pages:
    try:
        response = session.get(f"{base_url}{page}", timeout=5)
        status = response.status_code
        
        # Check for form elements
        has_form = '<form' in response.text
        has_inputs = '<input' in response.text
        has_submit = 'type="submit"' in response.text or 'type=submit' in response.text
        
        status_text = "OK" if status == 200 else f"ERROR ({status})"
        form_text = "Has Form" if has_form else "No Form"
        input_text = "Has Inputs" if has_inputs else "No Inputs"
        submit_text = "Has Submit" if has_submit else "No Submit"
        
        print(f"{page}: {status_text} - {form_text}, {input_text}, {submit_text}")
        
    except Exception as e:
        print(f"{page}: ERROR - {str(e)}")

# Test data tables
print("\n3. TESTING DATA TABLES:")
print("-" * 40)

table_pages = [
    "/students",
    "/fleet",
    "/fleet-vehicles",
    "/maintenance-records",
    "/monthly-mileage-reports"
]

for page in table_pages:
    try:
        response = session.get(f"{base_url}{page}", timeout=5)
        status = response.status_code
        
        # Check for table elements
        has_table = '<table' in response.text
        has_tbody = '<tbody' in response.text
        has_rows = '<tr' in response.text
        
        # Count rows (rough estimate)
        row_count = response.text.count('<tr') - 1  # Subtract header row
        
        status_text = "OK" if status == 200 else f"ERROR ({status})"
        table_text = f"Has Table ({row_count} rows)" if has_table else "No Table"
        
        print(f"{page}: {status_text} - {table_text}")
        
    except Exception as e:
        print(f"{page}: ERROR - {str(e)}")

print("\n" + "=" * 60)
print("TESTING COMPLETE")