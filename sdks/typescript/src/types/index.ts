// Core types for Leash SDK

export interface LeashConfig {
  gatewayUrl?: string;
  apiKey?: string;
  timeout?: number;
  retryAttempts?: number;
  fallbackProviders?: string[];
  cacheEnabled?: boolean;
  debugMode?: boolean;
  defaultHeaders?: Record<string, string>;
}

export interface ChatCompletionParams {
  model: string;
  messages: Message[];
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  frequency_penalty?: number;
  presence_penalty?: number;
  stop?: string | string[];
  stream?: boolean;
  user?: string;
}

export interface Message {
  role: 'system' | 'user' | 'assistant' | 'function';
  content: string;
  name?: string;
  function_call?: FunctionCall;
}

export interface FunctionCall {
  name: string;
  arguments: string;
}

export interface ChatCompletionResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: Choice[];
  usage: Usage;
  system_fingerprint?: string;
}

export interface Choice {
  index: number;
  message: Message;
  finish_reason: string;
  logprobs?: any;
}

export interface Usage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cost_usd?: number;
}

export interface StreamingChatCompletionResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: StreamingChoice[];
  system_fingerprint?: string;
}

export interface StreamingChoice {
  index: number;
  delta: MessageDelta;
  finish_reason?: string;
}

export interface MessageDelta {
  role?: string;
  content?: string;
  function_call?: Partial<FunctionCall>;
}

// Error types
export interface LeashError extends Error {
  code: string;
  status?: number;
  provider?: string;
  requestId?: string;
  details?: Record<string, any>;
}

// Provider types
export interface ProviderInfo {
  name: string;
  endpoint: string;
  models: string[];
  healthy: boolean;
  lastCheck: string;
  responseTime?: number;
}

// Gateway metrics types
export interface GatewayMetrics {
  requestCount: number;
  errorRate: number;
  averageLatency: number;
  totalCost: number;
  providerHealth: Record<string, ProviderInfo>;
  moduleStatus: Record<string, ModuleStatus>;
}

export interface ModuleStatus {
  name: string;
  type: string;
  enabled: boolean;
  healthy: boolean;
  requestsProcessed: number;
  errorCount: number;
  averageLatency: number;
}

// Fallback configuration
export interface FallbackConfig {
  enabled: boolean;
  providers: string[];
  strategy: 'round-robin' | 'priority' | 'cost-optimized';
  maxRetries: number;
  retryDelay: number;
}

// Cache configuration
export interface CacheConfig {
  enabled: boolean;
  ttl: number;
  maxSize: number;
  keyGenerator?: (params: ChatCompletionParams) => string;
}

// Middleware types
export type RequestMiddleware = (
  params: ChatCompletionParams,
  config: LeashConfig
) => Promise<ChatCompletionParams>;

export type ResponseMiddleware = (
  response: ChatCompletionResponse,
  params: ChatCompletionParams,
  config: LeashConfig
) => Promise<ChatCompletionResponse>;

// Event types for monitoring
export interface RequestEvent {
  type: 'request';
  requestId: string;
  timestamp: string;
  provider: string;
  model: string;
  tokenCount?: number;
  cost?: number;
}

export interface ResponseEvent {
  type: 'response';
  requestId: string;
  timestamp: string;
  provider: string;
  model: string;
  statusCode: number;
  latency: number;
  tokenCount: number;
  cost: number;
}

export interface ErrorEvent {
  type: 'error';
  requestId: string;
  timestamp: string;
  provider?: string;
  error: LeashError;
}

export type LeashEvent = RequestEvent | ResponseEvent | ErrorEvent;

// Integration types
export interface LangChainConfig {
  modelName: string;
  temperature?: number;
  maxTokens?: number;
  streaming?: boolean;
}

export interface VercelAIConfig {
  model: string;
  apiKey?: string;
  baseURL?: string;
}
