#!/bin/bash
set -euo pipefail

# Config
TUNNEL_NAME="replit-tunnel"
DOMAIN="hs-bus.org"
APP_PORT="8080"
CLOUDFLARED="./cloudflared"
CLOUDFLARE_DIR="$HOME/.cloudflared"

# Step 1: Ensure cloudflared binary
if [ ! -f "$CLOUDFLARED" ]; then
  echo "⬇️ Downloading cloudflared..."
  curl -L -o "$CLOUDFLARED" https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
  chmod +x "$CLOUDFLARED"
fi

# Step 2: Delete old tunnel if exists
echo "🧨 Deleting tunnel if exists..."
$CLOUDFLARED tunnel delete "$TUNNEL_NAME" || echo "🟡 No existing tunnel to delete."

# Step 3: Remove cert.pem
echo "🧹 Removing old cert.pem..."
rm -f "$CLOUDFLARE_DIR/cert.pem"

# Step 4: Login
echo "🔐 Logging into Cloudflare..."
$CLOUDFLARED tunnel login

# Step 5: Create tunnel
echo "🚧 Creating new tunnel..."
TUNNEL_CREATE_OUTPUT=$($CLOUDFLARED tunnel create "$TUNNEL_NAME")
TUNNEL_ID=$(echo "$TUNNEL_CREATE_OUTPUT" | grep -oE '[a-f0-9\-]{36}' | head -n 1)

if [ -z "$TUNNEL_ID" ]; then
  echo "❌ Tunnel creation failed. Could not extract ID."
  exit 1
fi

echo "✅ Tunnel created with ID: $TUNNEL_ID"

# Step 6: Write config.yml
CONFIG_YML="$CLOUDFLARE_DIR/config.yml"
CRED_FILE="$CLOUDFLARE_DIR/$TUNNEL_ID.json"

cat <<EOF > "$CONFIG_YML"
tunnel: $TUNNEL_ID
credentials-file: $CRED_FILE
no-autoupdate: true
no-quic: true

ingress:
  - hostname: $DOMAIN
    service: http://localhost:$APP_PORT
  - service: http_status:404
EOF

echo "📁 config.yml saved at $CONFIG_YML"
echo "🔑 Credentials saved at $CRED_FILE"

# Step 7: Show status
echo "🚀 To start the tunnel: ./cloudflared tunnel --config $CONFIG_YML run"
