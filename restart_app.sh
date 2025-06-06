#!/bin/bash
set -euo pipefail

echo "🛑 Stopping existing Go app (if any)..."
pkill -f "go run main.go" || echo "⚠️ No Go app running"

echo "🟢 Starting Go app..."
nohup go run main.go > server.log 2>&1 &
sleep 2

echo "⏳ Checking port 8080..."
until curl -s http://localhost:8080 > /dev/null; do
  echo "🔁 Still waiting for Go server..."
  sleep 2
done

echo "✅ Go server is live on port 8080"
