import axios, { AxiosInstance, AxiosResponse } from 'axios';
import EventSource from 'eventsource';
import {
  LeashConfig,
  ChatCompletionParams,
  ChatCompletionResponse,
  StreamingChatCompletionResponse,
  LeashError,
  GatewayMetrics,
  FallbackConfig,
  RequestMiddleware,
  ResponseMiddleware,
  LeashEvent
} from '../types';
import { LeashErrorHandler } from '../errors/LeashError';
import { ProviderDetector } from '../providers/detector';
import { FallbackManager } from '../middleware/fallback';
import { CacheManager } from '../middleware/cache';

export class LeashLLM {
  private config: Required<LeashConfig>;
  private httpClient: AxiosInstance;
  private providerDetector: ProviderDetector;
  private fallbackManager: FallbackManager;
  private cacheManager: CacheManager;
  private requestMiddleware: RequestMiddleware[] = [];
  private responseMiddleware: ResponseMiddleware[] = [];
  private eventListeners: ((event: LeashEvent) => void)[] = [];

  constructor(config: LeashConfig = {}) {
    this.config = {
      gatewayUrl: config.gatewayUrl || 'http://localhost:8080',
      apiKey: config.apiKey || '',
      timeout: config.timeout || 30000,
      retryAttempts: config.retryAttempts || 3,
      fallbackProviders: config.fallbackProviders || [],
      cacheEnabled: config.cacheEnabled || false,
      debugMode: config.debugMode || false,
      defaultHeaders: config.defaultHeaders || {},
    };

    this.httpClient = axios.create({
      baseURL: this.config.gatewayUrl,
      timeout: this.config.timeout,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': `leash-sdk-typescript/1.0.0`,
        ...this.config.defaultHeaders,
      },
    });

    // Add API key if provided
    if (this.config.apiKey) {
      this.httpClient.defaults.headers.common['Authorization'] = `Bearer ${this.config.apiKey}`;
    }

    this.providerDetector = new ProviderDetector();
    this.fallbackManager = new FallbackManager(this.config, this.httpClient);
    this.cacheManager = new CacheManager(this.config);

    this.setupInterceptors();
  }

  // Main chat completions method (OpenAI compatible)
  async chatCompletions(params: ChatCompletionParams): Promise<ChatCompletionResponse> {
    const requestId = this.generateRequestId();
    const startTime = Date.now();

    try {
      // Apply request middleware
      let processedParams = params;
      for (const middleware of this.requestMiddleware) {
        processedParams = await middleware(processedParams, this.config);
      }

      // Check cache first
      if (this.config.cacheEnabled && !processedParams.stream) {
        const cached = await this.cacheManager.get(processedParams);
        if (cached) {
          this.emitEvent({
            type: 'response',
            requestId,
            timestamp: new Date().toISOString(),
            provider: 'cache',
            model: processedParams.model,
            statusCode: 200,
            latency: Date.now() - startTime,
            tokenCount: cached.usage.total_tokens,
            cost: cached.usage.cost_usd || 0,
          });
          return cached;
        }
      }

      // Detect provider and build URL
      const provider = this.providerDetector.detectProvider(processedParams.model);
      const url = this.buildProviderUrl(provider);

      this.emitEvent({
        type: 'request',
        requestId,
        timestamp: new Date().toISOString(),
        provider,
        model: processedParams.model,
      });

      // Make request with fallback support
      const response = await this.fallbackManager.executeWithFallback(
        async (selectedProvider: string) => {
          const providerUrl = this.buildProviderUrl(selectedProvider);
          return this.makeRequest(providerUrl, processedParams);
        },
        provider,
        processedParams.model
      );

      // Apply response middleware
      let processedResponse = response;
      for (const middleware of this.responseMiddleware) {
        processedResponse = await middleware(processedResponse, processedParams, this.config);
      }

      // Cache response if enabled
      if (this.config.cacheEnabled && !processedParams.stream) {
        await this.cacheManager.set(processedParams, processedResponse);
      }

      this.emitEvent({
        type: 'response',
        requestId,
        timestamp: new Date().toISOString(),
        provider,
        model: processedParams.model,
        statusCode: 200,
        latency: Date.now() - startTime,
        tokenCount: processedResponse.usage.total_tokens,
        cost: processedResponse.usage.cost_usd || 0,
      });

      return processedResponse;
    } catch (error) {
      const leashError = LeashErrorHandler.handleError(error, {
        requestId,
        provider: this.providerDetector.detectProvider(params.model),
        model: params.model,
      });

      this.emitEvent({
        type: 'error',
        requestId,
        timestamp: new Date().toISOString(),
        provider: this.providerDetector.detectProvider(params.model),
        error: leashError,
      });

      throw leashError;
    }
  }

  // Streaming chat completions
  async streamChatCompletions(
    params: ChatCompletionParams,
    onChunk: (chunk: StreamingChatCompletionResponse) => void,
    onError?: (error: LeashError) => void,
    onComplete?: () => void
  ): Promise<void> {
    const requestId = this.generateRequestId();
    const provider = this.providerDetector.detectProvider(params.model);
    const url = this.buildProviderUrl(provider);

    const streamingParams = { ...params, stream: true };

    try {
      this.emitEvent({
        type: 'request',
        requestId,
        timestamp: new Date().toISOString(),
        provider,
        model: params.model,
      });

      const response = await this.httpClient.post(url, streamingParams, {
        responseType: 'stream',
        headers: {
          'Accept': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
      });

      this.processEventStream(response.data, onChunk, onError, onComplete);
    } catch (error) {
      const leashError = LeashErrorHandler.handleError(error, {
        requestId,
        provider,
        model: params.model,
      });

      if (onError) {
        onError(leashError);
      } else {
        throw leashError;
      }
    }
  }

  // Gateway metrics and monitoring
  async getMetrics(): Promise<GatewayMetrics> {
    try {
      const response = await this.httpClient.get('/metrics');
      return response.data;
    } catch (error) {
      throw LeashErrorHandler.handleError(error, { requestId: this.generateRequestId() });
    }
  }

  // Provider health check
  async getProviderHealth(): Promise<Record<string, any>> {
    try {
      const response = await this.httpClient.get('/health/providers');
      return response.data;
    } catch (error) {
      throw LeashErrorHandler.handleError(error, { requestId: this.generateRequestId() });
    }
  }

  // Configuration methods
  updateConfig(newConfig: Partial<LeashConfig>): void {
    this.config = { ...this.config, ...newConfig };
    
    // Update HTTP client if necessary
    if (newConfig.gatewayUrl) {
      this.httpClient.defaults.baseURL = newConfig.gatewayUrl;
    }
    if (newConfig.timeout) {
      this.httpClient.defaults.timeout = newConfig.timeout;
    }
    if (newConfig.apiKey) {
      this.httpClient.defaults.headers.common['Authorization'] = `Bearer ${newConfig.apiKey}`;
    }
  }

  getConfig(): LeashConfig {
    return { ...this.config };
  }

  // Middleware management
  addRequestMiddleware(middleware: RequestMiddleware): void {
    this.requestMiddleware.push(middleware);
  }

  addResponseMiddleware(middleware: ResponseMiddleware): void {
    this.responseMiddleware.push(middleware);
  }

  // Event handling
  addEventListener(listener: (event: LeashEvent) => void): void {
    this.eventListeners.push(listener);
  }

  removeEventListener(listener: (event: LeashEvent) => void): void {
    const index = this.eventListeners.indexOf(listener);
    if (index > -1) {
      this.eventListeners.splice(index, 1);
    }
  }

  // Static factory methods
  static fromEnv(): LeashLLM {
    return new LeashLLM({
      gatewayUrl: process.env.LEASH_GATEWAY_URL,
      apiKey: process.env.LEASH_API_KEY,
      debugMode: process.env.LEASH_DEBUG === 'true',
    });
  }

  static async fromConfig(configPath: string): Promise<LeashLLM> {
    // In a real implementation, this would load from a config file
    throw new Error('Config file loading not implemented in this demo');
  }

  // Private methods
  private async makeRequest(url: string, params: ChatCompletionParams): Promise<ChatCompletionResponse> {
    const response: AxiosResponse<ChatCompletionResponse> = await this.httpClient.post(url, params);
    return response.data;
  }

  private buildProviderUrl(provider: string): string {
    return `/v1/${provider}/chat/completions`;
  }

  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private setupInterceptors(): void {
    // Request interceptor
    this.httpClient.interceptors.request.use(
      (config) => {
        if (this.config.debugMode) {
          console.log('[Leash SDK] Request:', config);
        }
        return config;
      },
      (error) => {
        if (this.config.debugMode) {
          console.error('[Leash SDK] Request Error:', error);
        }
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.httpClient.interceptors.response.use(
      (response) => {
        if (this.config.debugMode) {
          console.log('[Leash SDK] Response:', response);
        }
        return response;
      },
      (error) => {
        if (this.config.debugMode) {
          console.error('[Leash SDK] Response Error:', error);
        }
        return Promise.reject(error);
      }
    );
  }

  private processEventStream(
    stream: any,
    onChunk: (chunk: StreamingChatCompletionResponse) => void,
    onError?: (error: LeashError) => void,
    onComplete?: () => void
  ): void {
    let buffer = '';

    stream.on('data', (chunk: Buffer) => {
      buffer += chunk.toString();
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data === '[DONE]') {
            if (onComplete) onComplete();
            return;
          }

          try {
            const parsed: StreamingChatCompletionResponse = JSON.parse(data);
            onChunk(parsed);
          } catch (error) {
            if (onError) {
              onError(LeashErrorHandler.handleError(error, { requestId: this.generateRequestId() }));
            }
          }
        }
      }
    });

    stream.on('error', (error: Error) => {
      if (onError) {
        onError(LeashErrorHandler.handleError(error, { requestId: this.generateRequestId() }));
      }
    });

    stream.on('end', () => {
      if (onComplete) onComplete();
    });
  }

  private emitEvent(event: LeashEvent): void {
    for (const listener of this.eventListeners) {
      try {
        listener(event);
      } catch (error) {
        if (this.config.debugMode) {
          console.error('[Leash SDK] Event listener error:', error);
        }
      }
    }
  }

  // Cleanup
  destroy(): void {
    this.eventListeners = [];
    this.requestMiddleware = [];
    this.responseMiddleware = [];
  }
}
