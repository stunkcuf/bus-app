#!/bin/bash
set -e

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

# Download cloudflared if missing
if [ ! -f ./cloudflared ]; then
  echo "⬇️ Downloading cloudflared..."
  curl -L -o cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
  chmod +x cloudflared
fi

# Start Go app
echo "🟢 Launching Go app..."
go run main.go > server.log 2>&1 &
GO_PID=$!

# Wait for Go app to open port 8080
echo "⏳ Waiting for port 8080..."
for i in {1..15}; do
  if curl -s http://localhost:8080 > /dev/null; then
    echo "✅ Go server is responding"
    break
  fi
  echo "🔁 [$i] Still waiting for Go server..."
  sleep 2
done

if ! curl -s http://localhost:8080 > /dev/null; then
  echo "❌ Go server failed to start. Dumping server.log:"
  cat server.log
  kill $GO_PID
  exit 1
fi

# Start tunnel if not already running
if ! pgrep -f "cloudflared.*run" > /dev/null; then
  echo "🚀 Starting Cloudflare Tunnel..."
  ./cloudflared tunnel --config ~/.cloudflared/config.yml run > tunnel.log 2>&1 &
else
  echo "✅ Tunnel already running"
fi
