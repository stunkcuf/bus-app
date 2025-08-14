import os
import re

# Templates that need jQuery and Bootstrap JS
templates_to_fix = [
    "analytics_dashboard.html",
    "fleet_vehicles.html",
    "maintenance_records.html",
    "service_records.html",
    "route_assignment_wizard.html",
    "gps_tracking.html",
    "add_student_wizard.html",
    "monthly_mileage_reports.html",
    "report_builder.html",
    "approve_users.html",
    "manage_users.html",
    "change_password.html",
    "notification_preferences.html"
]

# The scripts that should be added before </body>
bootstrap_scripts = """  <!-- jQuery and Bootstrap JS -->
  <script src="https://code.jquery.com/jquery-3.7.1.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
"""

templates_dir = "templates"
fixed_count = 0

for template in templates_to_fix:
    filepath = os.path.join(templates_dir, template)
    
    if not os.path.exists(filepath):
        print(f"Template not found: {template}")
        continue
    
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Check if jQuery is already included
    if 'jquery' in content.lower():
        print(f"jQuery already present in {template}, skipping...")
        continue
    
    # Find the </body> tag and insert scripts before it
    if '</body>' in content:
        # Insert the scripts right before </body>
        new_content = content.replace('</body>', bootstrap_scripts + '</body>')
        
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(new_content)
        
        print(f"Fixed: {template}")
        fixed_count += 1
    else:
        print(f"No </body> tag found in {template}")

print(f"\nFixed {fixed_count} templates")