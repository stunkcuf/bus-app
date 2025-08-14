import re

# Read the file
with open('handlers_truly_missing.go', 'r') as f:
    content = f.read()

# Fix all getSessionUser calls
content = re.sub(r'username, role := getSessionUser\(r\)', 
                'username := getUsernameFromRequest(r)\n\t\trole := getRoleFromRequest(r)', 
                content)

# Fix all renderTemplate calls (add r parameter)
content = re.sub(r'renderTemplate\(w, (".*?"), ', 
                r'renderTemplate(w, r, \1, ', 
                content)

# Fix the ECSE student struct fields
content = content.replace('student.StudentName', 'student.Name')
content = content.replace('student.Program', 'student.Bus')

# Write back
with open('handlers_truly_missing.go', 'w') as f:
    f.write(content)

print("Fixed handlers_truly_missing.go")