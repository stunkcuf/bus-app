#!/bin/bash
set -e

# Git sync
git pull origin main || echo "⚠️ Git pull failed"

# Cloudflared check...
# (skipped here for brevity)

# Loop Go app
while true; do
  echo "🔁 Starting Go server..."
  go run main.go > server.log 2>&1
  echo "⚠️ Go app exited. Restarting in 5s..."
  sleep 5
done
