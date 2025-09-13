import { openai } from '@ai-sdk/openai';
import { anthropic } from '@ai-sdk/anthropic';
import { google } from '@ai-sdk/google';
import { streamText, convertToModelMessages, UIMessage } from 'ai';

export const runtime = 'edge';
export const maxDuration = 30; // Allow streaming responses up to 30 seconds

const GATEWAY_URL = process.env.GATEWAY_URL || 'http://localhost:8080';

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
    
    let { messages = [], provider = 'openai' } = body;

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
      };
    });

    const providerConfig = providers[provider as keyof typeof providers];
    if (!providerConfig) {
      return new Response(JSON.stringify({ error: 'Invalid provider' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Configure the provider with gateway URL
    const model = providerConfig.client(providerConfig.model, {
      baseURL: providerConfig.baseURL,
      apiKey: providerConfig.apiKey,
      headers: {
        'X-Gateway-Provider': provider,
        'X-Request-ID': crypto.randomUUID()
      }
    });

    // Stream the response
    const result = streamText({
      model,
      messages: convertToModelMessages(uiMessages),
      temperature: 0.7,
      maxTokens: 1000,
      onFinish: async ({ text, usage, finishReason }) => {
        // Metrics could be tracked here if needed
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
