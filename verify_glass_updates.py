#!/usr/bin/env python3
"""
Verify which templates have been properly updated with glass morphism design
"""

import os
from pathlib import Path

def check_template_features(file_path):
    """Check if a template has the required glass morphism features"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        features = {
            'navbar_glass': 'navbar-glass' in content,
            'glass_card': 'glass-card' in content,
            'floating_orbs': 'class="orb' in content,
            'animated_bg': 'backgroundShift' in content,
            'dark_theme_css': '/static/dark_theme_text.css' in content,
            'gradients': '--gradient-1' in content
        }
        
        return features
    except Exception as e:
        return None

def main():
    templates_dir = Path('C:\\Users\\mycha\\hs-bus\\templates')
    
    # Get all HTML files
    all_templates = list(templates_dir.glob('*.html'))
    
    print(f"Verifying {len(all_templates)} templates...\n")
    
    fully_updated = []
    partially_updated = []
    not_updated = []
    errors = []
    
    for template in all_templates:
        features = check_template_features(template)
        
        if features is None:
            errors.append(template.name)
            continue
        
        # Count implemented features
        implemented = sum(features.values())
        
        if implemented >= 5:  # Has most/all features
            fully_updated.append((template.name, features))
        elif implemented >= 2:  # Has some features
            partially_updated.append((template.name, features))
        else:  # Has few/no features
            not_updated.append((template.name, features))
    
    # Report results
    print(f"FULLY UPDATED ({len(fully_updated)} templates):")
    print("-" * 50)
    for name, _ in sorted(fully_updated):
        print(f"  [OK] {name}")
    
    print(f"\n\nPARTIALLY UPDATED ({len(partially_updated)} templates):")
    print("-" * 50)
    for name, features in sorted(partially_updated):
        missing = [k for k, v in features.items() if not v]
        print(f"  [PARTIAL] {name} - Missing: {', '.join(missing)}")
    
    print(f"\n\nNOT UPDATED ({len(not_updated)} templates):")
    print("-" * 50)
    for name, _ in sorted(not_updated):
        print(f"  [NO] {name}")
    
    if errors:
        print(f"\n\nERRORS ({len(errors)} templates):")
        print("-" * 50)
        for name in sorted(errors):
            print(f"  [ERROR] {name}")
    
    print(f"\n\nSUMMARY:")
    print(f"  Fully Updated: {len(fully_updated)}")
    print(f"  Partially Updated: {len(partially_updated)}")
    print(f"  Not Updated: {len(not_updated)}")
    print(f"  Errors: {len(errors)}")
    print(f"  Total: {len(all_templates)}")

if __name__ == "__main__":
    main()