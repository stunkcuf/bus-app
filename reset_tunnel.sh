#!/bin/bash
set -euo pipefail

# CONFIG
TUNNEL_NAME="replit-tunnel"
DOMAIN="hs-bus.org"
APP_PORT="8080"

# Ensure clean tunnel state
echo "🧨 Deleting tunnel (if exists)..."
./cloudflared tunnel delete "$TUNNEL_NAME" || echo "🟡 No existing tunnel to delete."

# Clear cert.pem (if present)
echo "🧹 Removing old origin cert..."
rm -f ~/.cloudflared/cert.pem

# Login fresh
echo "🔐 Logging into Cloudflare..."
./cloudflared tunnel login

# Create new tunnel
echo "🚧 Creating new tunnel: $TUNNEL_NAME..."
TUNNEL_CREATE=$(./cloudflared tunnel create "$TUNNEL_NAME")
TUNNEL_ID=$(echo "$TUNNEL_CREATE" | grep -oE '[a-f0-9\-]{36}' | head -n 1)

# Validate tunnel ID was captured
if [[ -z "$TUNNEL_ID" ]]; then
  echo "❌ Failed to extract TUNNEL_ID. Aborting."
  exit 1
fi

echo "✅ Tunnel ID: $TUNNEL_ID"

# Config paths
CONFIG_PATH="$HOME/.cloudflared/config.yml"
CRED_FILE="$HOME/.cloudflared/$TUNNEL_ID.json"

# Write config
echo "📝 Writing config file..."
cat <<EOF > "$CONFIG_PATH"
tunnel: $TUNNEL_ID
credentials-file: $CRED_FILE
origincert: $HOME/.cloudflared/cert.pem
no-quic: true

ingress:
  - hostname: $DOMAIN
    service: http://localhost:$APP_PORT
  - service: http_status:404
EOF

# Final check
echo "📁 config.yml saved at $CONFIG_PATH"
echo "🔑 Credentials expected at $CRED_FILE"

# Start the tunnel
echo "🚀 Launching tunnel..."
./cloudflared tunnel --config "$CONFIG_PATH" run
