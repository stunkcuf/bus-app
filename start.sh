#!/bin/bash
set -e

echo "🔄 Pulling latest changes from GitHub..."
git fetch origin
git reset --hard origin/main

# Ensure cloudflared is available
if [ ! -f ./cloudflared ]; then
  echo "⬇️ Downloading cloudflared..."
  wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -O cloudflared
  chmod +x cloudflared
else
  echo "✅ cloudflared already exists"
fi

echo "🚀 Starting Cloudflare Tunnel..."
./cloudflared tunnel --config ~/.cloudflared/config.yml run replit-tunnel >> tunnel.log 2>&1 &

sleep 2

echo "🟢 Launching Go app with restart loop..."
while true; do
  echo "▶️ Starting main.go..."
  go run main.go >> startup.log 2>&1
  echo "🔁 App exited. Restarting in 5s..."
  sleep 5
done
