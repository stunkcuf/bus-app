#!/bin/bash
set -euo pipefail

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

# Ensure cloudflared is downloaded
if [ ! -f ./cloudflared ]; then
  echo "⬇️ Downloading cloudflared..."
  curl -L -o cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
  chmod +x cloudflared
fi

# Start Go app
echo "🟢 Launching Go app..."
nohup go run main.go > server.log 2>&1 &
sleep 2

# Wait for Go app to start
echo "⏳ Waiting for port 8080..."
for i in {1..15}; do
  if curl -s http://localhost:8080 > /dev/null; then
    echo "✅ Go server is responding"
    break
  fi
  echo "🔁 [$i] Still waiting for Go server..."
  sleep 2
done

# Start tunnel only if not running
if ! pgrep -f "cloudflared.*run" > /dev/null; then
  echo "🚀 Starting Cloudflare Tunnel..."
  nohup ./cloudflared tunnel --config ~/.cloudflared/config.yml run > tunnel.log 2>&1 &
else
  echo "✅ Tunnel already running"
fi
