#!/bin/bash
set -e

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

# Download cloudflared if not already present
if [ ! -f ./cloudflared ]; then
  echo "⬇️ Downloading cloudflared..."
  curl -L -o cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
  chmod +x cloudflared
fi

# Start Go server in background and log output
echo "🟢 Launching Go app..."
go run main.go > server.log 2>&1 &
sleep 2

# Wait for Go server on port 8080
echo "⏳ Waiting for port 8080..."
for i in {1..15}; do
  if curl -s http://localhost:8080 > /dev/null; then
    echo "✅ Go server is responding"
    break
  else
    echo "🔁 [$i] Still waiting for Go server..."
    sleep 2
  fi
done

# Start Cloudflare tunnel if not running
if ! pgrep -f "cloudflared.*run" > /dev/null; then
  echo "🚀 Starting Cloudflare Tunnel..."
  ./cloudflared tunnel --config ~/.cloudflared/config.yml run > tunnel.log 2>&1 &
else
  echo "✅ Tunnel already running"
fi
