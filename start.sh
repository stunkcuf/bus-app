#!/bin/bash
set -euo pipefail

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

echo "🛑 Stopping any existing Go app..."
pkill -f "go run main.go" || echo "⚠️ No Go app running"

echo "🟢 Launching Go app..."
PORT=5000 exec go run mainv2.go
