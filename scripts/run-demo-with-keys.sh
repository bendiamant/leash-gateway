#!/bin/bash

# Script to run the Leash Gateway demo with real API keys
# Usage: ./scripts/run-demo-with-keys.sh

echo "🚀 Starting Leash Gateway Demo with API Keys"
echo ""

# Check if API keys are provided
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  OPENAI_API_KEY not set. You can set it with:"
    echo "   export OPENAI_API_KEY='sk-your-key'"
    echo ""
fi

if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "⚠️  ANTHROPIC_API_KEY not set. You can set it with:"
    echo "   export ANTHROPIC_API_KEY='ant-your-key'"
    echo ""
fi

echo "📋 Current API key status:"
echo "   OpenAI: ${OPENAI_API_KEY:+✅ Set}${OPENAI_API_KEY:-❌ Not set}"
echo "   Anthropic: ${ANTHROPIC_API_KEY:+✅ Set}${ANTHROPIC_API_KEY:-❌ Not set}"
echo ""

echo "🐳 Starting Docker containers..."
docker-compose -f docker/docker-compose.demo.yaml down
docker-compose -f docker/docker-compose.demo.yaml up -d

echo ""
echo "⏳ Waiting for services to start..."
sleep 15

echo ""
echo "🧪 Testing gateway connectivity..."
curl -w "Gateway health: %{http_code}\n" http://localhost:8080/health 2>/dev/null
curl -w "Module Host health: %{http_code}\n" http://localhost:8081/health 2>/dev/null

echo ""
echo "🎉 Demo is ready!"
echo ""
echo "📱 Open the demo app: http://localhost:3001"
echo "📊 Grafana dashboard: http://localhost:3000 (admin/admin)"
echo "📈 Prometheus metrics: http://localhost:9091"
echo ""

if [ -n "$OPENAI_API_KEY" ]; then
    echo "✅ With your OpenAI key, you'll get real responses!"
else
    echo "ℹ️  Without API keys, you'll see 403 errors (which proves the gateway works!)"
fi

echo ""
echo "🛑 To stop the demo: make demo-down"
