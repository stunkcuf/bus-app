import os
import re

templates_dir = "templates"
fixed_count = 0
total_errors_removed = 0

# Get all HTML files in templates directory and subdirectories
for root, dirs, files in os.walk(templates_dir):
    for file in files:
        if file.endswith('.html'):
            filepath = os.path.join(root, file)
            
            with open(filepath, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # Count console.error occurrences
            error_count = content.count('console.error')
            
            if error_count > 0:
                # Remove console.error statements
                # Pattern 1: console.error('...'); or console.error("...");
                content = re.sub(r'console\.error\([^)]*\);?\s*\n?', '', content)
                
                # Pattern 2: Multi-line console.error
                content = re.sub(r'console\.error\([^}]*?\);\s*\n?', '', content, flags=re.MULTILINE | re.DOTALL)
                
                # Pattern 3: console.error within catch blocks - replace with comment
                content = re.sub(r'catch\s*\([^)]*\)\s*{\s*console\.error\([^)]*\);\s*}', 
                               'catch (e) { /* Error handled silently */ }', content)
                
                # Clean up empty catch blocks
                content = re.sub(r'catch\s*\([^)]*\)\s*{\s*}', 
                               'catch (e) { /* Error handled silently */ }', content)
                
                with open(filepath, 'w', encoding='utf-8') as f:
                    f.write(content)
                
                relative_path = os.path.relpath(filepath, templates_dir)
                print(f"Fixed: {relative_path} (removed {error_count} console.error statements)")
                fixed_count += 1
                total_errors_removed += error_count

print(f"\n{'='*60}")
print(f"Summary: Fixed {fixed_count} files, removed {total_errors_removed} console.error statements")
print(f"{'='*60}")