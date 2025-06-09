#!/bin/bash
echo "🔄 Staging all changes..."
git add .

echo "📝 Committing changes..."
git commit -m "Auto-update from Replit $(date +'%Y-%m-%d %H:%M:%S')"

echo "🔃 Pulling latest changes (rebase)..."
git pull --rebase origin main

echo "🚀 Pushing to GitHub..."
git push origin main
