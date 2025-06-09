#!/bin/bash
set -euo pipefail

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

echo "🛑 Stopping any existing Go app..."
pkill -f "go run main.go" || echo "⚠️ No Go app running"

echo "🟢 Launching Go app..."
PORT=5000 nohup go run main.go > server.log 2>&1 &

for i in {1..15}; do
  if curl -s "http://localhost:5000" > /dev/null 2>&1; then
    echo "✅ Go server is responding"
    break
  fi
  echo "🔁 [$i] Still waiting for Go server..."
  sleep 2
done

if ! curl -s http://localhost:5000 > /dev/null 2>&1; then
  echo "❌ Server failed to start on port 5000"
  echo "📋 Checking server logs:"
  tail -20 server.log
fi
