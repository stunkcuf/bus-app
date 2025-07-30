#!/usr/bin/env python3
"""
Script to update all HTML templates with glass morphism design from fleet.html
"""

import os
import re
from pathlib import Path

# Templates that have already been updated
UPDATED_TEMPLATES = {
    'fleet.html', 'company_fleet.html', 'manage_users.html',
    'maintenance_records.html', 'add_bus_wizard.html', 'add_fuel_record.html',
    'add_student_wizard.html', 'analytics_dashboard.html', 'approve_users.html',
    'assign_routes.html', 'driver_dashboard.html'
}

# High priority templates to update first
HIGH_PRIORITY = [
    'edit_bus.html', 'emergency_dashboard.html', 'fleet_vehicle_add.html',
    'fleet_vehicle_edit.html', 'fleet_vehicles.html', 'fuel_records.html',
    'gps_tracking.html', 'maintenance_wizard.html', 'manager_dashboard.html',
    'manager_reports.html', 'monthly_mileage_reports.html', 'profile.html',
    'route_assignment_wizard.html', 'settings.html', 'students.html',
    'users.html', 'vehicle_maintenance.html'
]

# Glass morphism CSS to insert
GLASS_MORPHISM_CSS = '''  <style nonce="{{.CSPNonce}}">
    /* Ultimate beautiful design system */
    :root {
      --gradient-1: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      --gradient-2: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
      --gradient-3: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
      --gradient-4: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
      --gradient-5: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
      --gradient-6: linear-gradient(135deg, #30cfd0 0%, #330867 100%);
      --gradient-7: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
      --gradient-8: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
      --gradient-dark: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
    }
    
    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }
    
    body {
      background: #0f0c29;
      background: linear-gradient(to right, #24243e, #302b63, #0f0c29);
      min-height: 100vh;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      overflow-x: hidden;
      color: white;
    }
    
    /* Animated background */
    body::before {
      content: '';
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background-image: 
        radial-gradient(circle at 20% 80%, rgba(102, 126, 234, 0.3) 0%, transparent 50%),
        radial-gradient(circle at 80% 20%, rgba(240, 147, 251, 0.3) 0%, transparent 50%),
        radial-gradient(circle at 40% 40%, rgba(79, 172, 254, 0.2) 0%, transparent 50%);
      animation: backgroundShift 20s ease-in-out infinite;
      z-index: -1;
    }
    
    @keyframes backgroundShift {
      0%, 100% { transform: translate(0, 0) rotate(0deg); }
      33% { transform: translate(-20px, -20px) rotate(120deg); }
      66% { transform: translate(20px, -10px) rotate(240deg); }
    }
    
    /* Glassmorphism navigation */
    .navbar-glass {
      background: rgba(255, 255, 255, 0.1);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border-bottom: 1px solid rgba(255, 255, 255, 0.2);
      padding: 1rem 0;
      position: sticky;
      top: 0;
      z-index: 1000;
      box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    }
    
    .navbar-brand {
      color: white !important;
      font-weight: 700;
      font-size: 1.5rem;
      display: flex;
      align-items: center;
      gap: 0.75rem;
      text-decoration: none;
      transition: all 0.3s ease;
    }
    
    .navbar-brand:hover {
      transform: translateY(-2px);
      text-shadow: 0 5px 15px rgba(255, 255, 255, 0.3);
    }
    
    /* Floating orbs */
    .orb {
      position: absolute;
      border-radius: 50%;
      filter: blur(40px);
      opacity: 0.6;
      animation: float 20s infinite ease-in-out;
    }
    
    .orb1 {
      width: 300px;
      height: 300px;
      background: radial-gradient(circle, rgba(102, 126, 234, 0.8), transparent);
      top: -150px;
      left: -150px;
    }
    
    .orb2 {
      width: 400px;
      height: 400px;
      background: radial-gradient(circle, rgba(240, 147, 251, 0.6), transparent);
      bottom: -200px;
      right: -200px;
      animation-delay: -5s;
    }
    
    .orb3 {
      width: 250px;
      height: 250px;
      background: radial-gradient(circle, rgba(79, 172, 254, 0.7), transparent);
      top: 50%;
      left: 50%;
      animation-delay: -10s;
    }
    
    @keyframes float {
      0%, 100% { transform: translate(0, 0) scale(1); }
      33% { transform: translate(30px, -30px) scale(1.1); }
      66% { transform: translate(-20px, 20px) scale(0.9); }
    }
    
    /* Glass cards */
    .glass-card {
      background: rgba(255, 255, 255, 0.1);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border-radius: 30px;
      border: 1px solid rgba(255, 255, 255, 0.2);
      padding: 2rem;
      margin-bottom: 2rem;
      box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
      color: white;
    }
    
    /* Hero section */
    .hero-section {
      position: relative;
      padding: 2rem 0 1.5rem;
      text-align: center;
      color: white;
      overflow: hidden;
    }
    
    .hero-content h1 {
      font-size: 2.5rem;
      font-weight: 800;
      margin-bottom: 1rem;
      background: linear-gradient(to right, #fff, #a8edea, #fed6e3, #fff);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      animation: gradientShift 8s ease-in-out infinite;
    }
    
    @keyframes gradientShift {
      0%, 100% { background-position: 0% 50%; }
      50% { background-position: 100% 50%; }
    }
    
    /* Form styling */
    .form-control,
    .form-select {
      background: rgba(255, 255, 255, 0.1);
      border: 2px solid rgba(255, 255, 255, 0.2);
      border-radius: 15px;
      color: white;
      padding: 0.75rem 1rem;
      transition: all 0.3s ease;
    }
    
    .form-control:focus,
    .form-select:focus {
      background: rgba(255, 255, 255, 0.15);
      border-color: #667eea;
      box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.3);
      color: white;
      outline: none;
    }
    
    .form-control::placeholder {
      color: rgba(255, 255, 255, 0.7);
    }
    
    /* Button styling */
    .btn {
      padding: 0.75rem 1.5rem;
      border-radius: 25px;
      font-weight: 600;
      transition: all 0.3s ease;
      border: none;
      text-decoration: none;
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
    }
    
    .btn-primary {
      background: var(--gradient-1);
      color: white;
      box-shadow: 0 5px 20px rgba(102, 126, 234, 0.5);
    }
    
    .btn-primary:hover {
      transform: translateY(-3px);
      box-shadow: 0 10px 40px rgba(102, 126, 234, 0.7);
      color: white;
    }
    
    /* Tables */
    .table {
      color: white;
    }
    
    .table thead th {
      background: rgba(255, 255, 255, 0.15);
      color: white;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 1px;
      padding: 1rem;
      border: none;
    }
    
    .table tbody tr {
      background: rgba(255, 255, 255, 0.05);
      transition: all 0.3s ease;
    }
    
    .table tbody tr:hover {
      background: rgba(255, 255, 255, 0.1);
      transform: translateX(10px);
    }
    
    /* Text colors for dark theme */
    h1, h2, h3, h4, h5, h6, p, span, label, a, td, th, li {
      color: white !important;
    }
    
    .text-muted {
      color: rgba(255, 255, 255, 0.7) !important;
    }
    
    /* Card styling */
    .card {
      background: rgba(255, 255, 255, 0.1);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 20px;
      color: white;
    }
    
    .card-header {
      background: rgba(255, 255, 255, 0.15);
      border-bottom: 1px solid rgba(255, 255, 255, 0.2);
      color: white;
    }
    
    /* Modal styling */
    .modal-content {
      background: rgba(30, 30, 50, 0.95);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border: 1px solid rgba(255, 255, 255, 0.2);
      color: white;
    }
    
    .modal-header {
      border-bottom: 1px solid rgba(255, 255, 255, 0.2);
    }
    
    .modal-footer {
      border-top: 1px solid rgba(255, 255, 255, 0.2);
    }
    
    /* Animations */
    .fade-in {
      opacity: 0;
      transform: translateY(30px);
      animation: fadeInUp 0.8s ease-out forwards;
    }
    
    @keyframes fadeInUp {
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
  </style>'''

def update_template(file_path):
    """Update a single template file with glass morphism design"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Skip if already updated (has glass morphism markers)
        if 'navbar-glass' in content or 'glass-card' in content:
            return False, "Already updated"
        
        # 1. Replace existing style tags or add new ones
        if '<style' in content:
            # Find the last </style> tag and insert our CSS after it
            last_style_end = content.rfind('</style>')
            if last_style_end != -1:
                insert_pos = last_style_end + len('</style>')
                content = content[:insert_pos] + '\n' + GLASS_MORPHISM_CSS + content[insert_pos:]
            else:
                # No closing style tag found, add before </head>
                head_end = content.find('</head>')
                if head_end != -1:
                    content = content[:head_end] + GLASS_MORPHISM_CSS + '\n' + content[head_end:]
        else:
            # No style tag found, add before </head>
            head_end = content.find('</head>')
            if head_end != -1:
                content = content[:head_end] + GLASS_MORPHISM_CSS + '\n' + content[head_end:]
        
        # 2. Add floating orbs after body tag
        body_tag = re.search(r'<body[^>]*>', content)
        if body_tag:
            orbs_html = '''  <!-- Floating orbs -->
  <div class="orb orb1"></div>
  <div class="orb orb2"></div>
  <div class="orb orb3"></div>
  
'''
            insert_pos = body_tag.end()
            content = content[:insert_pos] + '\n' + orbs_html + content[insert_pos:]
        
        # 3. Update navigation to glass style
        # Find navbar patterns
        navbar_patterns = [
            r'<nav[^>]*class="[^"]*navbar[^"]*"[^>]*>',
            r'<div[^>]*class="[^"]*navbar[^"]*"[^>]*>'
        ]
        
        for pattern in navbar_patterns:
            navbar_match = re.search(pattern, content)
            if navbar_match:
                old_nav = navbar_match.group(0)
                # Replace navbar classes with navbar-glass
                new_nav = re.sub(r'class="[^"]*"', 'class="navbar navbar-glass"', old_nav)
                content = content.replace(old_nav, new_nav)
                
                # Add logout button if not present
                if '/logout' not in content:
                    # Find end of navbar content
                    nav_end_pattern = r'</nav>|</div>\s*<!--\s*end navbar\s*-->'
                    nav_end = re.search(nav_end_pattern, content)
                    if nav_end:
                        logout_html = '''          <form method="POST" action="/logout" class="m-0">
            <button type="submit" class="btn btn-outline-light btn-sm">
              <i class="bi bi-box-arrow-right"></i> Logout
            </button>
          </form>
'''
                        insert_pos = nav_end.start()
                        content = content[:insert_pos] + logout_html + '\n' + content[insert_pos:]
        
        # 4. Update hero sections
        hero_patterns = [
            r'<section[^>]*class="[^"]*hero[^"]*"[^>]*>',
            r'<div[^>]*class="[^"]*hero[^"]*"[^>]*>',
            r'<div[^>]*class="[^"]*jumbotron[^"]*"[^>]*>'
        ]
        
        for pattern in hero_patterns:
            hero_match = re.search(pattern, content)
            if hero_match:
                old_hero = hero_match.group(0)
                new_hero = re.sub(r'class="[^"]*"', 'class="hero-section"', old_hero)
                content = content.replace(old_hero, new_hero)
        
        # 5. Update container divs to use glass-card
        content = re.sub(r'<div class="container mt-5">', '<div class="container">\n    <div class="glass-card fade-in">', content)
        content = re.sub(r'<div class="card">', '<div class="glass-card">', content)
        
        # 6. Update buttons
        content = re.sub(r'btn-outline-primary', 'btn-primary', content)
        content = re.sub(r'btn-secondary', 'btn-primary', content)
        
        # 7. Add dark theme text link before </head>
        if '/static/dark_theme_text.css' not in content:
            head_end = content.find('</head>')
            if head_end != -1:
                dark_theme_link = '    <!-- Dark Theme Text Colors -->\n    <link rel="stylesheet" href="/static/dark_theme_text.css">\n'
                content = content[:head_end] + dark_theme_link + content[head_end:]
        
        # 8. Ensure Bootstrap and Bootstrap Icons are included
        if 'bootstrap@5' not in content:
            head_end = content.find('</head>')
            if head_end != -1:
                bootstrap_links = '''  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.0/font/bootstrap-icons.css">
'''
                content = content[:head_end] + bootstrap_links + '\n' + content[head_end:]
        
        # Write updated content
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        return True, "Successfully updated"
    
    except Exception as e:
        return False, f"Error: {str(e)}"

def main():
    templates_dir = Path('C:\\Users\\mycha\\hs-bus\\templates')
    
    # Get all HTML files
    all_templates = list(templates_dir.glob('*.html'))
    
    # Filter out already updated templates
    templates_to_update = [t for t in all_templates if t.name not in UPDATED_TEMPLATES]
    
    # Sort by priority
    priority_templates = []
    other_templates = []
    
    for template in templates_to_update:
        if template.name in HIGH_PRIORITY:
            priority_templates.append(template)
        else:
            other_templates.append(template)
    
    # Process templates
    all_templates_to_process = priority_templates + other_templates
    
    print(f"Found {len(all_templates_to_process)} templates to update")
    print(f"Processing {len(priority_templates)} high priority templates first...\n")
    
    success_count = 0
    skip_count = 0
    error_count = 0
    
    for i, template_path in enumerate(all_templates_to_process):
        print(f"[{i+1}/{len(all_templates_to_process)}] Processing {template_path.name}...", end=' ')
        
        success, message = update_template(template_path)
        
        if success:
            print(f"[OK] {message}")
            success_count += 1
        elif "Already updated" in message:
            print(f"[SKIP] {message}")
            skip_count += 1
        else:
            print(f"[ERROR] {message}")
            error_count += 1
    
    print(f"\n\nSummary:")
    print(f"  Successfully updated: {success_count}")
    print(f"  Already updated: {skip_count}")
    print(f"  Errors: {error_count}")
    print(f"  Total processed: {len(all_templates_to_process)}")

if __name__ == "__main__":
    main()