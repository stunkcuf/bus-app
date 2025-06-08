#!/bin/bash
set -euo pipefail

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

echo "🛑 Stopping any existing Go app..."
pkill -f "go run main.go" || echo "⚠️ No Go app running"

echo "🟢 Launching Go app..."
nohup go run main.go > server.log 2>&1 &

echo "⏳ Waiting for port 8080..."
for i in {1..15}; do
  if curl -s http://localhost:8080 > /dev/null; then
    echo "✅ Go server is responding"
    break
  fi
  echo "🔁 [$i] Still waiting for Go server..."
  sleep 2
done
