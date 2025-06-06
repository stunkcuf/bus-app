#!/bin/bash
set -euo pipefail

echo "🛑 Stopping existing Go app (if any)..."
pkill -f "go run main.go" 2>/dev/null || echo "⚠️ No Go app running"

echo "🟢 Starting Go app..."
nohup go run main.go > server.log 2>&1 &
sleep 2

echo "⏳ Waiting for Go server on port 8080..."
for i in {1..10}; do
  if curl -s http://localhost:8080 > /dev/null; then
    echo "✅ Go server is live on port 8080"
    exit 0
  fi
  echo "🔁 Still waiting ($i)..."
  sleep 2
done

echo "❌ Go server failed to start after multiple attempts."
exit 1

