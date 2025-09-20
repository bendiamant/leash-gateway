#!/bin/bash
echo "Fixing Prometheus port in .env.local..."
if [ -f .env.local ]; then
    sed -i.bak 's/GATEWAY_METRICS_URL=http:\/\/localhost:9090/GATEWAY_METRICS_URL=http:\/\/localhost:9091/' .env.local
    echo "Updated GATEWAY_METRICS_URL to port 9091"
    echo "Restart the dev server for changes to take effect"
else
    echo ".env.local not found"
fi
