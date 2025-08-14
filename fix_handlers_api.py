import re

# Read the file
with open('handlers_missing_api.go', 'r') as f:
    content = f.read()

# Fix function signatures - remove the db parameter and extra return
content = re.sub(r'func (\w+Handler)\(db \*sql\.DB\) http\.HandlerFunc \{\s*return func\(w http\.ResponseWriter, r \*http\.Request\) \{',
                r'func \1(w http.ResponseWriter, r *http.Request) {\n\tdb := getDB(r)',
                content)

# Remove the extra closing brace
content = re.sub(r'\}\s*\}$', '}', content)

# Fix any remaining db parameter functions
content = re.sub(r'func (\w+Handler)\(w http\.ResponseWriter, r \*http\.Request\) \{',
                r'func \1(w http.ResponseWriter, r *http.Request) {',
                content)

# Write back
with open('handlers_missing_api.go', 'w') as f:
    f.write(content)

print("Fixed handlers_missing_api.go")