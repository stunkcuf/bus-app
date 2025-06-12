#!/bin/bash

# Transportation App Health Check Script

echo "=== Transportation App Health Check ==="
echo ""

# Check if we're in the right directory
if [ ! -f "main.go" ]; then
    echo "❌ main.go not found. Are you in the app directory?"
    exit 1
fi

echo "✅ Found main.go"

# Check data directory
if [ ! -d "data" ]; then
    echo "❌ data/ directory doesn't exist"
    mkdir -p data
    echo "✅ Created data/ directory"
else
    echo "✅ data/ directory exists"
fi

# Check required JSON files
files=("users.json" "buses.json" "routes.json" "route_assignments.json" "students.json" "driver_logs.json" "maintenance.json")

echo ""
echo "=== JSON Files Check ==="
for file in "${files[@]}"; do
    path="data/$file"
    if [ ! -f "$path" ]; then
        echo "❌ Missing: $file"
    elif python3 -m json.tool "$path" > /dev/null 2>&1; then
        size=$(wc -c < "$path")
        echo "✅ Valid JSON: $file ($size bytes)"
    elif python -m json.tool "$path" > /dev/null 2>&1; then
        size=$(wc -c < "$path")
        echo "✅ Valid JSON: $file ($size bytes)"
    else
        echo "❌ Invalid JSON: $file"
        echo "   First few lines:"
        head -n 3 "$path" 2>/dev/null || echo "   Cannot read file"
    fi
done

# Check templates directory
echo ""
echo "=== Templates Check ==="
if [ ! -d "templates" ]; then
    echo "❌ templates/ directory doesn't exist"
    echo "   Creating templates directory..."
    mkdir -p templates
else
    echo "✅ templates/ directory exists"

    # Count HTML files
    html_files=$(find templates -name "*.html" 2>/dev/null)
    html_count=$(echo "$html_files" | grep -c "\.html$" 2>/dev/null || echo "0")

    if [ "$html_count" -eq 0 ]; then
        echo "❌ No .html files found in templates/"
        echo "   Templates are required for the app to work"
    else
        echo "✅ Found $html_count template files:"
        echo "$html_files" | sed 's/^/   /'
    fi
fi

# Check if port 5000 is available
echo ""
echo "=== Port Check ==="
if command -v netstat > /dev/null 2>&1; then
    if netstat -tuln 2>/dev/null | grep -q ":5000 "; then
        echo "⚠️  Port 5000 is already in use"
        echo "   Processes using port 5000:"
        netstat -tulpn 2>/dev/null | grep ":5000 " | sed 's/^/   /'
    else
        echo "✅ Port 5000 is available"
    fi
else
    echo "⚠️  netstat not available, cannot check port"
fi

# Check Go installation
echo ""
echo "=== Go Environment ==="
if command -v go > /dev/null 2>&1; then
    echo "✅ Go is installed: $(go version)"

    # Check if go.mod exists
    if [ -f "go.mod" ]; then
        echo "✅ go.mod exists"
        echo "   Module: $(head -n 1 go.mod)"
    else
        echo "⚠️  go.mod not found"
        echo "   You may need to run: go mod init your-app-name"
    fi

    # Check dependencies
    echo "   Checking dependencies..."
    if go mod tidy > /dev/null 2>&1; then
        echo "✅ Dependencies are good"
    else
        echo "❌ Dependency issues detected"
        echo "   Try running: go mod tidy"
    fi
else
    echo "❌ Go is not installed"
fi

# Check directory permissions
echo ""
echo "=== Permissions ==="
ls -ld . data/ templates/ 2>/dev/null | sed 's/^/   /'

# Show file sizes
echo ""
echo "=== File Overview ==="
if [ -d "data" ]; then
    ls -lah data/ | sed 's/^/   /'
else
    echo "   No data directory"
fi

# Try to identify common issues
echo ""
echo "=== Common Issues Check ==="

# Check for empty JSON files
for file in data/*.json 2>/dev/null; do
    if [ -f "$file" ] && [ ! -s "$file" ]; then
        echo "❌ Empty file detected: $(basename "$file")"
    fi
done

# Check for template syntax (basic)
for file in templates/*.html 2>/dev/null; do
    if [ -f "$file" ]; then
        if grep -q "{{" "$file" && ! grep -q "}}" "$file"; then
            echo "⚠️  Possible template syntax issue in $(basename "$file")"
        fi
    fi
done

echo ""
echo "=== Quick Start Debugging ==="
echo "1. To run the app: go run main.go"
echo "2. To see detailed logs: go run main.go 2>&1 | tee app.log"
echo "3. To test if server starts: curl -I http://localhost:5000/ (in another terminal)"
echo "4. To view app in browser: http://localhost:5000/"

echo ""
echo "=== Health check complete ==="