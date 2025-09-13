# Leash Security Gateway Demo Application

A modern, standalone demo application showcasing the capabilities of the Leash Security Gateway - an enterprise-grade LLM security and governance platform.

![Leash Gateway Demo](https://img.shields.io/badge/Status-FULLY%20WORKING-brightgreen)
![AI SDK](https://img.shields.io/badge/AI%20SDK-v5-blue)
![License](https://img.shields.io/badge/License-Apache%202.0-blue)

> ✅ **Status**: Demo application is fully functional with AI SDK v5 integration complete!

## ✨ Features

### 🔐 Security & Governance
- **Centralized Policy Enforcement**: All LLM traffic flows through a single control plane
- **Rate Limiting**: Configurable limits per API key/tenant
- **Content Filtering**: PII detection and sensitive data redaction
- **Audit Logging**: Complete request/response trail for compliance

### ⚡ Multi-Provider Support
- **OpenAI**: GPT-4o Mini integration
- **Anthropic**: Claude 3.5 Sonnet support
- **Google**: Gemini 1.5 Flash compatibility
- **Seamless Switching**: Change providers without modifying application code

### 📊 Real-time Monitoring
- **Live Metrics Dashboard**: Request counts, latency, and error rates
- **Cost Tracking**: Real-time usage and spend monitoring
- **Health Monitoring**: Service status and module health checks
- **Performance Metrics**: P50/P95 latency tracking

### 🚀 Developer Experience
- **Zero Code Changes**: Just update your base URL
- **Streaming Support**: Real-time response streaming
- **Beautiful UI**: Modern interface built with shadcn/ui
- **Vercel AI SDK**: Production-ready AI integration

## 🛠️ Tech Stack

- **Frontend**: Next.js 14 (App Router), React, TypeScript
- **Backend**: Next.js API Routes with Vercel AI SDK
- **UI Components**: shadcn/ui, Radix UI, Tailwind CSS
- **Charts**: Recharts for metrics visualization
- **Gateway**: Envoy Proxy with gRPC module host
- **Storage**: PostgreSQL (config), Redis (cache/rate limiting)
- **Monitoring**: Prometheus + Grafana

## 📋 Prerequisites

- Node.js 18+ and npm/pnpm
- Docker and Docker Compose (for gateway services)
- API keys for at least one provider:
  - OpenAI API Key
  - Anthropic API Key
  - Google AI API Key

## 🚀 Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/leash-security/gateway.git
cd gateway/leash-demo-app
```

### 2. Install Dependencies

```bash
npm install
# or
pnpm install
```

### 3. Configure Environment

Copy the example environment file and add your API keys:

```bash
# Copy the example file
cp env.example .env.local

# Edit .env.local and add your actual API keys
```

The `.env.local` file should contain:

```env
# Gateway Configuration
GATEWAY_URL=http://localhost:8080
GATEWAY_METRICS_URL=http://localhost:9090
GATEWAY_HEALTH_URL=http://localhost:8081

# API Keys (at least one required for real LLM calls)
OPENAI_API_KEY=sk-your-actual-openai-api-key
ANTHROPIC_API_KEY=sk-ant-your-actual-anthropic-api-key
GOOGLE_API_KEY=your-actual-google-api-key

# Optional
NEXT_PUBLIC_APP_URL=http://localhost:3000
```

### 4. Start the Gateway Services

```bash
# Start all gateway services
docker-compose -f docker-compose.gateway.yaml up -d

# Or with monitoring (includes Grafana)
docker-compose -f docker-compose.gateway.yaml --profile monitoring up -d
```

Verify services are running:
- Gateway: http://localhost:8080
- Metrics: http://localhost:9091
- Module Host Health: http://localhost:8081/health
- Grafana (if enabled): http://localhost:3002 (admin/admin)

### 5. Start the Demo Application

```bash
# Development mode
npm run dev

# Production build
npm run build
npm run start
```

Open http://localhost:3000 in your browser.

## 🎮 Usage Guide

### Chat Interface
1. Select a provider from the dropdown (OpenAI, Anthropic, or Google)
2. Type your message and press Enter or click Send
3. Watch real-time streaming responses
4. See latency and cost metrics for each response

### Metrics Dashboard
- View real-time request counts per provider
- Monitor latency across different providers
- Track cumulative costs
- Check gateway and module health status

### Demo Scenarios

#### Test Rate Limiting
```javascript
// Rapid-fire requests to trigger rate limiting
for (let i = 0; i < 10; i++) {
  await sendMessage("Test " + i);
}
```

#### PII Filtering Demo
Try sending messages with personal information:
- "My SSN is 123-45-6789"
- "Call me at 555-123-4567"
- "My email is john@example.com"

#### Provider Comparison
Switch between providers mid-conversation to see:
- Response quality differences
- Latency variations
- Cost comparisons

## 🏗️ Architecture

```mermaid
graph TB
    subgraph "Demo App"
        UI[Next.js Frontend]
        API[API Routes + Vercel AI SDK]
    end
    
    subgraph "Gateway Layer"
        ENVOY[Envoy Proxy]
        MH[Module Host]
    end
    
    subgraph "Modules"
        RL[Rate Limiter]
        CF[Content Filter]
        CT[Cost Tracker]
        LOG[Logger]
    end
    
    subgraph "Providers"
        OAI[OpenAI]
        ANT[Anthropic]
        GOO[Google]
    end
    
    UI --> API
    API --> ENVOY
    ENVOY --> MH
    MH --> RL
    MH --> CF
    MH --> CT
    MH --> LOG
    ENVOY --> OAI
    ENVOY --> ANT
    ENVOY --> GOO
```

## 📦 Project Structure

```
leash-demo-app/
├── src/
│   ├── app/
│   │   ├── api/           # Backend API routes
│   │   │   ├── chat/      # LLM chat endpoint
│   │   │   ├── metrics/   # Gateway metrics
│   │   │   └── health/    # Health checks
│   │   ├── layout.tsx     # Root layout
│   │   └── page.tsx       # Main page
│   ├── components/
│   │   ├── chat/          # Chat components
│   │   ├── metrics/       # Dashboard components
│   │   └── ui/            # shadcn/ui components
│   └── lib/
│       └── utils.ts       # Utility functions
├── docker-compose.gateway.yaml
├── package.json
└── README.md
```

## 🔧 Configuration

### Gateway Modules

Modules can be configured in `configs/gateway/config.yaml`:

```yaml
modules:
  - name: ratelimiter
    enabled: true
    config:
      requests_per_minute: 100
      
  - name: contentfilter
    enabled: true
    config:
      detect_pii: true
      redact_patterns:
        - ssn
        - phone
        - email
```

### Provider Routing

The gateway automatically routes based on URL patterns:
- `/v1/openai/*` → OpenAI API
- `/v1/anthropic/*` → Anthropic API
- `/v1/google/*` → Google AI API

## 📊 Monitoring

### Prometheus Metrics

Key metrics available at http://localhost:9091:

- `leash_gateway_requests_total`: Total request count
- `leash_gateway_request_duration_seconds`: Request latency
- `leash_cost_usd_total`: Cumulative cost tracking
- `leash_ratelimit_hits_total`: Rate limit violations

### Grafana Dashboards

Import the provided dashboards for visualization:
1. Gateway Overview
2. Provider Performance
3. Cost Analysis
4. Module Health

## 🧪 Testing

```bash
# Run linting
npm run lint

# Type checking
npm run type-check

# Run in development with hot reload
npm run dev
```

## 🔧 Troubleshooting

### Common Issues and Solutions

1. **Gateway Connection Refused**
   ```bash
   # Check if services are running
   docker ps
   
   # View gateway logs
   docker logs leash-gateway-1
   
   # Restart services
   docker-compose -f docker-compose.gateway.yaml restart
   ```
   - Verify `GATEWAY_URL` in `.env.local` (should be `http://localhost:8080`)
   - Ensure ports 8080, 8081, 9090 are not in use

2. **Chat Not Responding / Streaming Issues**
   - Check browser console for errors (F12)
   - Verify API route logs in terminal running `npm run dev`
   - Try switching providers using the selector
   - Use SimpleChat mode as fallback (toggle button in UI)
   - For streaming issues, ensure you're using a modern browser

3. **API Key Errors**
   ```bash
   # Test your API keys directly
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer $OPENAI_API_KEY"
   ```
   - Verify keys in `.env.local` have no extra spaces/quotes
   - Check provider-specific formats:
     - OpenAI: `sk-...`
     - Anthropic: `sk-ant-...`
     - Google: API key from Google AI Studio

4. **No Metrics Displayed**
   - Generate some traffic first (send chat messages)
   - Check Prometheus directly: http://localhost:9090
   - Verify `GATEWAY_METRICS_URL` in `.env.local`
   - Wait 30 seconds for metrics to populate

5. **AI SDK v5 Migration Issues**
   If migrating from older code:
   - `useChat` API has completely changed
   - Input state must be managed manually with `useState`
   - Use `sendMessage` instead of `handleSubmit`
   - Use `toUIMessageStreamResponse()` not `toDataStreamResponse()`
   - See `.cursor/memory-bank/ai-sdk-v5-migration.md` for full guide

### Debug Mode

The app includes debug features:
- Toggle between SimpleChat and ChatInterfaceV5 in the UI
- Check API logs in the terminal
- Use browser DevTools Network tab to inspect streaming responses

## 🚢 Deployment

### Vercel Deployment

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel
```

### Docker Deployment

```bash
# Build the app
docker build -t leash-demo-app .

# Run with gateway
docker run -p 3000:3000 \
  -e GATEWAY_URL=http://gateway:8080 \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  leash-demo-app
```

## 🔒 Security Notes

- **API Keys**: Never commit API keys to version control
- **CORS**: Configure appropriate CORS settings for production
- **Rate Limiting**: Adjust limits based on your usage patterns
- **Content Filtering**: Customize PII patterns for your use case

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](../CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](../LICENSE) file for details.

## 🆘 Support

- **Documentation**: [Full Docs](https://github.com/leash-security/gateway/wiki)
- **Issues**: [GitHub Issues](https://github.com/leash-security/gateway/issues)
- **Discussions**: [GitHub Discussions](https://github.com/leash-security/gateway/discussions)

## 🙏 Acknowledgments

- Built with [Next.js](https://nextjs.org/) and [Vercel AI SDK](https://sdk.vercel.ai/)
- UI components from [shadcn/ui](https://ui.shadcn.com/)
- Powered by [Envoy Proxy](https://www.envoyproxy.io/)

---

Built with ❤️ by the Leash Security team