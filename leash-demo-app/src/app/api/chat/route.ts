import { openai } from '@ai-sdk/openai';
import { anthropic } from '@ai-sdk/anthropic';
import { google } from '@ai-sdk/google';
import { streamText, convertToModelMessages, UIMessage } from 'ai';

export const runtime = 'edge';
export const maxDuration = 30; // Allow streaming responses up to 30 seconds

const GATEWAY_URL = process.env.GATEWAY_URL || 'http://localhost:8080';

// Cost calculation based on provider pricing (per 1M tokens)
function calculateCost(provider: string, model: string, usage: any): number {
  if (!usage) return 0;
  
  const pricing: Record<string, Record<string, { input: number; output: number }>> = {
    openai: {
      'gpt-4o-mini': { input: 0.15, output: 0.60 },  // per 1M tokens
      'gpt-4o': { input: 5.00, output: 15.00 },
      'gpt-3.5-turbo': { input: 0.50, output: 1.50 }
    },
    anthropic: {
      'claude-3-5-sonnet-20241022': { input: 3.00, output: 15.00 },
      'claude-3-opus': { input: 15.00, output: 75.00 },
      'claude-3-haiku': { input: 0.25, output: 1.25 }
    },
    google: {
      'gemini-1.5-flash': { input: 0.075, output: 0.30 },
      'gemini-1.5-pro': { input: 1.25, output: 5.00 }
    }
  };
  
  const modelPricing = pricing[provider]?.[model] || { input: 0, output: 0 };
  const inputCost = (usage.promptTokens || 0) * modelPricing.input / 1_000_000;
  const outputCost = (usage.completionTokens || 0) * modelPricing.output / 1_000_000;
  
  return inputCost + outputCost;
}

// Provider configuration with gateway routing
const providers = {
  openai: {
    client: openai,
    model: 'gpt-4o-mini',
    baseURL: `${GATEWAY_URL}/v1/openai`,
    apiKey: process.env.OPENAI_API_KEY || 'demo-key'
  },
  anthropic: {
    client: anthropic,
    model: 'claude-3-5-sonnet-20241022',
    baseURL: `${GATEWAY_URL}/v1/anthropic`,
    apiKey: process.env.ANTHROPIC_API_KEY || 'demo-key'
  },
  google: {
    client: google,
    model: 'gemini-1.5-flash',
    baseURL: `${GATEWAY_URL}/v1/google`,
    apiKey: process.env.GOOGLE_API_KEY || 'demo-key'
  }
};

export async function POST(req: Request) {
  try {
    const body = await req.json();
    
    let { messages = [], provider = 'openai', useGateway = true } = body;

    // Ensure messages is an array
    if (!Array.isArray(messages)) {
      return new Response(JSON.stringify({ error: 'Messages must be an array' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Convert simple messages to UIMessage format if needed
    const uiMessages: UIMessage[] = messages.map((msg: any) => {
      // If it already has an id and parts, it's already a UIMessage
      if (msg.id && msg.parts) {
        return msg;
      }
      // Convert simple format to UIMessage
      return {
        id: msg.id || crypto.randomUUID(),
        role: msg.role,
        content: msg.content || msg.text || '',
        parts: msg.parts || (msg.content || msg.text ? [{
          type: 'text',
          text: msg.content || msg.text || ''
        }] : [])
      };
    });

    const providerConfig = providers[provider as keyof typeof providers];
    if (!providerConfig) {
      return new Response(JSON.stringify({ error: 'Invalid provider' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Log routing decision with clear visual separator
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`🤖 Provider: ${provider.toUpperCase()}`);
    console.log(`📍 Routing: ${useGateway ? '🚦 VIA GATEWAY' : '🔗 DIRECT TO PROVIDER'}`);
    if (useGateway) {
      console.log(`🌐 Gateway URL: ${providerConfig.baseURL}`);
    }
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    
    const baseURL = useGateway ? providerConfig.baseURL : undefined;

    // Configure the provider - with or without gateway
    const modelConfig: any = {
      apiKey: providerConfig.apiKey
    };
    
    if (useGateway) {
      modelConfig.baseURL = providerConfig.baseURL;
      modelConfig.headers = {
        'X-Gateway-Provider': provider,
        'X-Request-ID': crypto.randomUUID()
      };
      console.log(`✅ Gateway configuration applied: ${modelConfig.baseURL}`);
    } else {
      console.log(`⚠️ Direct mode - no gateway URL`);
    }
    
    const model = providerConfig.client(providerConfig.model, modelConfig);

    // Stream the response
    const startTime = Date.now();
    const requestId = crypto.randomUUID();
    const temperature = 0.7;
    const maxTokens = 1000;
    
    const result = streamText({
      model,
      messages: convertToModelMessages(uiMessages),
      temperature,
      maxTokens,
      onFinish: async ({ text, usage, finishReason }) => {
        // Write comprehensive metrics to PostgreSQL if using gateway
        if (useGateway) {
          const processingTime = Date.now() - startTime;
          try {
            // Write to enhanced audit logs
            await fetch('http://localhost:3002/api/metrics-pg/enhanced', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                audit: {
                  request_id: requestId,
                  provider,
                  model: providerConfig.model,
                  method: 'POST',
                  path: `/v1/${provider}/chat/completions`,
                  status_code: 200,
                  processing_time_ms: processingTime,
                  prompt_tokens: usage?.promptTokens || 0,
                  completion_tokens: usage?.completionTokens || 0,
                  total_tokens: usage?.totalTokens || 0,
                  temperature,
                  max_tokens: maxTokens,
                  cost_usd: calculateCost(provider, providerConfig.model, usage)
                },
                request_body: {
                  model: providerConfig.model,
                  messages: uiMessages,
                  temperature,
                  max_tokens: maxTokens
                },
                response_body: {
                  text,
                  usage,
                  finish_reason: finishReason
                },
                messages: uiMessages.concat([{
                  role: 'assistant',
                  content: text
                }])
              })
            });
            console.log(`📊 Enhanced metrics written: ${provider}/${providerConfig.model} - ${usage?.totalTokens || 0} tokens, $${calculateCost(provider, providerConfig.model, usage).toFixed(6)}`);
          } catch (error) {
            console.error('Failed to write metrics:', error);
          }
        }
      }
    });

    return result.toUIMessageStreamResponse();
  } catch (error) {
    console.error('Chat API error:', error);
    
    // Check if gateway is unavailable
    if (error instanceof Error && error.message.includes('ECONNREFUSED')) {
      return new Response(JSON.stringify({
        error: 'Gateway unavailable',
        message: 'The Leash Gateway is not running. Please start it with docker-compose.'
      }), {
        status: 503,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    return new Response(JSON.stringify({
      error: 'Internal server error',
      message: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}
