#!/bin/bash

# Script to rebuild and restart demo app when source code changes
# Usage: ./scripts/rebuild-demo.sh

echo "🔄 Rebuilding demo app with latest source code..."

# Stop demo app
docker-compose -f docker/docker-compose.demo.yaml stop demo-app

# Rebuild without cache
echo "🏗️ Building demo app container..."
docker-compose -f docker/docker-compose.demo.yaml build --no-cache demo-app

# Restart demo app
echo "🚀 Starting demo app..."
docker-compose -f docker/docker-compose.demo.yaml start demo-app

echo ""
echo "✅ Demo app rebuilt and restarted!"
echo "📱 Open: http://localhost:3001"
echo "🔄 Hard refresh in browser (Cmd+Shift+R) to see changes"
echo ""
echo "💡 Tip: Use this script whenever you change demo app source code"
