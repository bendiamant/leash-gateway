import { AxiosInstance } from 'axios';
import { LeashConfig, ChatCompletionResponse, ChatCompletionParams } from '../types';
import { LeashErrorHandler } from '../errors/LeashError';
import { ProviderDetector } from '../providers/detector';

export class FallbackManager {
  private config: LeashConfig;
  private httpClient: AxiosInstance;
  private providerDetector: ProviderDetector;
  private providerHealth: Map<string, ProviderHealthInfo>;

  constructor(config: LeashConfig, httpClient: AxiosInstance) {
    this.config = config;
    this.httpClient = httpClient;
    this.providerDetector = new ProviderDetector();
    this.providerHealth = new Map();
  }

  async executeWithFallback<T>(
    operation: (provider: string) => Promise<T>,
    primaryProvider: string,
    model: string
  ): Promise<T> {
    const providers = this.getProvidersForModel(model, primaryProvider);
    let lastError: Error;

    for (let i = 0; i < providers.length; i++) {
      const provider = providers[i];
      
      // Check if provider is healthy
      if (!this.isProviderHealthy(provider)) {
        continue;
      }

      try {
        const result = await this.executeWithRetry(operation, provider);
        this.recordSuccess(provider);
        return result;
      } catch (error) {
        this.recordFailure(provider, error);
        lastError = error;

        // Don't retry on client errors (4xx)
        if (LeashErrorHandler.isClientError(error as any)) {
          throw error;
        }

        // Don't retry on policy violations
        if ((error as any).code === 'POLICY_VIOLATION') {
          throw error;
        }

        // Continue to next provider for server errors
        continue;
      }
    }

    throw new Error(`All providers failed for model ${model}. Last error: ${lastError?.message}`);
  }

  private async executeWithRetry<T>(
    operation: (provider: string) => Promise<T>,
    provider: string
  ): Promise<T> {
    let lastError: Error;
    
    for (let attempt = 0; attempt <= this.config.retryAttempts; attempt++) {
      try {
        return await operation(provider);
      } catch (error) {
        lastError = error as Error;

        // Don't retry on client errors
        if (LeashErrorHandler.isClientError(error as any)) {
          throw error;
        }

        // Don't retry on last attempt
        if (attempt === this.config.retryAttempts) {
          break;
        }

        // Exponential backoff
        const delay = Math.min(1000 * Math.pow(2, attempt), 10000);
        await this.sleep(delay);
      }
    }

    throw lastError;
  }

  private getProvidersForModel(model: string, primaryProvider: string): string[] {
    const providers: string[] = [];
    
    // Add primary provider first
    providers.push(primaryProvider);
    
    // Add fallback providers that support the model
    for (const fallbackProvider of this.config.fallbackProviders) {
      if (fallbackProvider !== primaryProvider && this.providerSupportsModel(fallbackProvider, model)) {
        providers.push(fallbackProvider);
      }
    }

    return providers;
  }

  private providerSupportsModel(provider: string, model: string): boolean {
    try {
      const detectedProvider = this.providerDetector.detectProvider(model);
      return detectedProvider === provider;
    } catch {
      return false;
    }
  }

  private isProviderHealthy(provider: string): boolean {
    const health = this.providerHealth.get(provider);
    if (!health) {
      return true; // Assume healthy if no data
    }

    const now = Date.now();
    const timeSinceLastFailure = now - health.lastFailureTime;
    
    // Consider provider unhealthy if recent failures
    if (health.consecutiveFailures >= 3 && timeSinceLastFailure < 60000) {
      return false;
    }

    return true;
  }

  private recordSuccess(provider: string): void {
    const health = this.providerHealth.get(provider) || {
      consecutiveFailures: 0,
      lastFailureTime: 0,
      lastSuccessTime: 0,
      totalRequests: 0,
      totalFailures: 0,
    };

    health.consecutiveFailures = 0;
    health.lastSuccessTime = Date.now();
    health.totalRequests++;

    this.providerHealth.set(provider, health);
  }

  private recordFailure(provider: string, error: Error): void {
    const health = this.providerHealth.get(provider) || {
      consecutiveFailures: 0,
      lastFailureTime: 0,
      lastSuccessTime: 0,
      totalRequests: 0,
      totalFailures: 0,
    };

    health.consecutiveFailures++;
    health.lastFailureTime = Date.now();
    health.totalRequests++;
    health.totalFailures++;

    this.providerHealth.set(provider, health);
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Public methods for monitoring
  getProviderHealth(): Record<string, ProviderHealthInfo> {
    const result: Record<string, ProviderHealthInfo> = {};
    
    for (const [provider, health] of this.providerHealth.entries()) {
      result[provider] = { ...health };
    }

    return result;
  }

  resetProviderHealth(provider?: string): void {
    if (provider) {
      this.providerHealth.delete(provider);
    } else {
      this.providerHealth.clear();
    }
  }
}

interface ProviderHealthInfo {
  consecutiveFailures: number;
  lastFailureTime: number;
  lastSuccessTime: number;
  totalRequests: number;
  totalFailures: number;
}
