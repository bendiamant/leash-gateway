// Main entry point for Leash Security Gateway TypeScript SDK

export { LeashLLM } from './client/LeashLLM';

// Types
export type {
  LeashConfig,
  ChatCompletionParams,
  ChatCompletionResponse,
  Message,
  Choice,
  Usage,
  StreamingChatCompletionResponse,
  StreamingChoice,
  MessageDelta,
  LeashError,
  ProviderInfo,
  GatewayMetrics,
  ModuleStatus,
  FallbackConfig,
  CacheConfig,
  RequestMiddleware,
  ResponseMiddleware,
  LeashEvent,
  RequestEvent,
  ResponseEvent,
  ErrorEvent,
} from './types';

// Error classes
export {
  LeashErrorHandler,
  AuthenticationError,
  PolicyViolationError,
  RateLimitError,
  ProviderUnavailableError,
  NetworkError,
} from './errors/LeashError';

// Utilities
export { ProviderDetector } from './providers/detector';

// Middleware
export { FallbackManager } from './middleware/fallback';
export { CacheManager } from './middleware/cache';

// Version
export const VERSION = '1.0.0';

// Default configurations
export const DEFAULT_CONFIG: Partial<LeashConfig> = {
  gatewayUrl: 'http://localhost:8080',
  timeout: 30000,
  retryAttempts: 3,
  fallbackProviders: ['openai', 'anthropic'],
  cacheEnabled: false,
  debugMode: false,
};

// Convenience factory functions
export function createLeashClient(config?: LeashConfig): LeashLLM {
  return new LeashLLM(config);
}

export function createOpenAICompatibleClient(config?: LeashConfig): LeashLLM {
  // Create a client that mimics OpenAI SDK behavior
  const leashConfig = {
    ...config,
    // Ensure OpenAI compatibility mode
  };
  
  return new LeashLLM(leashConfig);
}

// Environment-based factory
export function createFromEnvironment(): LeashLLM {
  return LeashLLM.fromEnv();
}
