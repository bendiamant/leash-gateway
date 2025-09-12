# Leash Gateway Demo App

Interactive React demo showcasing the Leash Security Gateway capabilities.

## 🎯 Features

- **Multi-Provider Chat**: Switch between OpenAI, Anthropic, and Google seamlessly
- **Real-time Metrics**: Live cost tracking, performance monitoring, and usage analytics
- **Security Showcase**: Demonstrates rate limiting, content filtering, and policy enforcement
- **Provider Health**: Circuit breaker status and provider health monitoring
- **Beautiful UI**: Modern React interface with Tailwind CSS

## 🚀 Quick Start

### Prerequisites

- Node.js 16+
- Running Leash Gateway (see main repository)

### Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Open browser to http://localhost:3001
```

### Production Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

## 🏗️ Architecture

The demo app showcases the **configuration-based integration** approach:

```typescript
// Traditional approach (direct provider)
const openai = new OpenAI({
  apiKey: "sk-...",
  baseURL: "https://api.openai.com/v1"
});

// Leash Gateway approach (one line change!)
const openai = new OpenAI({
  apiKey: "sk-...",
  baseURL: "https://gateway.company.com/v1/openai"  // Through gateway
});

// Or use the Leash SDK for enhanced features
const leash = new LeashLLM({
  gatewayUrl: "https://gateway.company.com",
  fallbackProviders: ["openai", "anthropic", "google"]
});
```

## 📊 What the Demo Shows

### 1. **Provider Switching**
- Live switching between OpenAI, Anthropic, and Google
- Same interface, different providers
- No code changes required

### 2. **Security in Action**
- Rate limiting enforcement
- Content filtering (try sending "harmful" content)
- Cost tracking and alerting
- Request/response logging

### 3. **Real-time Monitoring**
- Request throughput and latency
- Cost accumulation over time
- Provider health status
- Module execution metrics

### 4. **Error Handling**
- Circuit breaker demonstrations
- Fallback logic between providers
- Policy violation handling
- Network error recovery

## 🎨 Components

### ChatInterface
- Multi-provider chat interface
- Real-time cost and latency display
- Error handling and retry logic
- Quick prompt suggestions

### MetricsDashboard
- Live charts and graphs
- Provider health monitoring
- Module status display
- Cost and usage analytics

## 🔧 Configuration

### Environment Variables

```bash
# Gateway connection
VITE_GATEWAY_URL=http://localhost:8080
VITE_API_KEY=your-api-key

# Debug mode
VITE_DEBUG=true
```

### Proxy Configuration

The Vite dev server proxies requests to the gateway:

```typescript
proxy: {
  '/v1': 'http://localhost:8080',      // Gateway API
  '/metrics': 'http://localhost:9090', // Prometheus metrics
  '/health': 'http://localhost:8081'   // Health checks
}
```

## 🧪 Testing the Demo

### 1. **Basic Chat Test**
```bash
# Start the gateway
make dev-up

# Start the demo app
cd demo-app && npm run dev

# Open http://localhost:3001
# Send a message and watch it route through the gateway
```

### 2. **Provider Switching Test**
- Send a message with OpenAI selected
- Switch to Anthropic and send another message
- Notice the provider change in the response metadata

### 3. **Security Policy Test**
- Try sending a message with "harmful" content
- Watch it get blocked by the content filter
- Check the metrics dashboard for policy violations

### 4. **Cost Tracking Test**
- Send multiple messages
- Watch the real-time cost accumulation
- Check the cost breakdown by provider

## 📱 Screenshots

The demo app showcases:
- Beautiful chat interface with provider selection
- Real-time metrics dashboard with charts
- Security policy enforcement demonstrations
- Provider health and circuit breaker status

## 🚀 Deployment

### Docker Deployment

```bash
# Build demo app image
docker build -t leash-demo-app .

# Run with gateway
docker-compose -f docker-compose.demo.yaml up
```

### Static Hosting

```bash
# Build static files
npm run build

# Deploy to any static hosting service
# (Vercel, Netlify, AWS S3, etc.)
```

## 🤝 Contributing

This demo app is part of the larger Leash Security Gateway project. See the main repository for contribution guidelines.

## 📄 License

Licensed under the Apache License, Version 2.0. See [LICENSE](../../LICENSE) for details.
