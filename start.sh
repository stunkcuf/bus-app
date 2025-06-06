#!/bin/bash
set -e

echo "🔄 Pulling latest changes from GitHub..."
git pull origin main || echo "⚠️ Git pull failed — check remote connection"

# ⬇️ Download cloudflared if missing
if [ ! -f ./cloudflared ]; then
  echo "⬇️ Downloading cloudflared..."
  curl -L -o cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
  chmod +x cloudflared
fi

# ✅ Start cloudflared tunnel if not already running
if ! pgrep -f "cloudflared.*run" > /dev/null; then
  echo "🚀 Starting Cloudflare Tunnel..."
  ./cloudflared tunnel --config ~/.cloudflared/config.yml run > tunnel.log 2>&1 &
else
  echo "✅ Tunnel already running"
fi

# ⏳ Wait for app port to be available
echo "⏳ Waiting for port 8080..."
until curl -s http://localhost:8080 > /dev/null; do
  echo "🔁 Still waiting for Go server..."
  sleep 2
done

# 🔁 Launch Go app with restart loop
echo "🟢 Launching Go app with auto-restart..."
while true; do
  echo "▶️ Starting main.go..."
  go run main.go > server.log 2>&1
  echo "⚠️ Go app crashed or stopped. Restarting in 5s..."
  sleep 5
done
