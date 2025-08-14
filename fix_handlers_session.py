import re

# Read the file
with open('handlers_truly_missing.go', 'r') as f:
    content = f.read()

# Pattern to replace getUsernameFromRequest and getRoleFromRequest
pattern1 = r'username := getUsernameFromRequest\(r\)\n\t\trole := getRoleFromRequest\(r\)\n\t\tif username == ""'
replacement1 = 'user := getUserFromSession(r)\n\t\tif user == nil'

content = re.sub(pattern1, replacement1, content)

# For manager-only handlers
pattern2 = r'username := getUsernameFromRequest\(r\)\n\t\trole := getRoleFromRequest\(r\)\n\t\tif username == "" \|\| role != "manager"'
replacement2 = 'user := getUserFromSession(r)\n\t\tif user == nil || user.Role != "manager"'

content = re.sub(pattern2, replacement2, content)

# Fix the renderTemplate calls to use user object
content = re.sub(r'"Username": username,\n\t\t\t"Role":\s+role,', 
                '"User": user,', 
                content)

# Write back
with open('handlers_truly_missing.go', 'w') as f:
    f.write(content)

print("Fixed handlers_truly_missing.go session handling")