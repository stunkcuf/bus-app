#!/usr/bin/env python3
"""
Fix remaining templates that need glass morphism updates
"""

import os
from pathlib import Path

# Templates that need fixes
PARTIAL_TEMPLATES = {
    'driver_reports.html': ['floating_orbs', 'animated_bg', 'gradients'],
    'edit_bus.html': ['floating_orbs', 'animated_bg', 'dark_theme_css'],
    'edit_user.html': ['floating_orbs', 'animated_bg', 'dark_theme_css'],
    'gps_tracking.html': ['navbar_glass', 'floating_orbs', 'animated_bg', 'gradients'],
    'manage_users.html': ['navbar_glass', 'floating_orbs'],
    'manager_reports.html': ['floating_orbs', 'animated_bg', 'gradients'],
    'progress_indicator.html': ['floating_orbs', 'dark_theme_css'],
    'route_assignment_wizard_enhanced.html': ['floating_orbs', 'animated_bg']
}

NOT_UPDATED = ['db_pool_monitor.html', 'students_lazy.html']

# Features to add
FLOATING_ORBS = '''  <!-- Floating orbs -->
  <div class="orb orb1"></div>
  <div class="orb orb2"></div>
  <div class="orb orb3"></div>
  
'''

ANIMATED_BG_CSS = '''    /* Animated background */
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
    
'''

GRADIENTS_CSS = '''    /* Ultimate beautiful design system */
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
    
'''

DARK_THEME_LINK = '    <!-- Dark Theme Text Colors -->\n    <link rel="stylesheet" href="/static/dark_theme_text.css">\n'

def fix_template(file_path, missing_features):
    """Fix a template by adding missing features"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Add floating orbs if missing
        if 'floating_orbs' in missing_features and 'class="orb' not in content:
            body_match = content.find('<body')
            if body_match != -1:
                body_end = content.find('>', body_match)
                if body_end != -1:
                    content = content[:body_end+1] + '\n' + FLOATING_ORBS + content[body_end+1:]
        
        # Add animated background if missing
        if 'animated_bg' in missing_features and 'backgroundShift' not in content:
            style_end = content.find('</style>')
            if style_end != -1:
                content = content[:style_end] + ANIMATED_BG_CSS + content[style_end:]
        
        # Add gradients if missing
        if 'gradients' in missing_features and '--gradient-1' not in content:
            style_start = content.find('<style')
            if style_start != -1:
                style_tag_end = content.find('>', style_start)
                if style_tag_end != -1:
                    content = content[:style_tag_end+1] + '\n' + GRADIENTS_CSS + content[style_tag_end+1:]
        
        # Add dark theme CSS if missing
        if 'dark_theme_css' in missing_features and '/static/dark_theme_text.css' not in content:
            head_end = content.find('</head>')
            if head_end != -1:
                content = content[:head_end] + DARK_THEME_LINK + content[head_end:]
        
        # Fix navbar-glass if missing
        if 'navbar_glass' in missing_features:
            content = content.replace('class="navbar', 'class="navbar navbar-glass')
            content = content.replace('navbar navbar-glass navbar-glass', 'navbar navbar-glass')  # Avoid duplicates
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        return True, "Fixed missing features"
    
    except Exception as e:
        return False, f"Error: {str(e)}"

def main():
    templates_dir = Path('C:\\Users\\mycha\\hs-bus\\templates')
    
    print("Fixing partially updated templates...\n")
    
    # Fix partial templates
    for template_name, missing_features in PARTIAL_TEMPLATES.items():
        template_path = templates_dir / template_name
        print(f"Fixing {template_name} - Missing: {', '.join(missing_features)}...", end=' ')
        
        success, message = fix_template(template_path, missing_features)
        print("[OK]" if success else f"[ERROR] {message}")
    
    print("\n\nUpdating templates that weren't updated at all...")
    
    # For completely non-updated templates, run the full update
    from update_templates_glass import update_template
    
    for template_name in NOT_UPDATED:
        template_path = templates_dir / template_name
        print(f"Updating {template_name}...", end=' ')
        
        success, message = update_template(template_path)
        print("[OK]" if success else f"[ERROR] {message}")
    
    print("\n\nAll templates should now be updated with glass morphism design!")

if __name__ == "__main__":
    main()