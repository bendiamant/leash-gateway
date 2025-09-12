// Provider detection logic based on model names

export class ProviderDetector {
  private modelToProviderMap: Record<string, string> = {
    // OpenAI models
    'gpt-4o': 'openai',
    'gpt-4o-mini': 'openai',
    'gpt-4': 'openai',
    'gpt-4-turbo': 'openai',
    'gpt-3.5-turbo': 'openai',
    'text-davinci-003': 'openai',
    'text-davinci-002': 'openai',
    'code-davinci-002': 'openai',

    // Anthropic models
    'claude-3-opus-20240229': 'anthropic',
    'claude-3-sonnet-20240229': 'anthropic',
    'claude-3-haiku-20240307': 'anthropic',
    'claude-2.1': 'anthropic',
    'claude-2.0': 'anthropic',
    'claude-instant-1.2': 'anthropic',

    // Google models
    'gemini-1.5-pro': 'google',
    'gemini-1.5-flash': 'google',
    'gemini-pro': 'google',
    'gemini-pro-vision': 'google',
  };

  private providerPrefixes: Record<string, string> = {
    'gpt-': 'openai',
    'text-': 'openai',
    'code-': 'openai',
    'claude-': 'anthropic',
    'gemini-': 'google',
  };

  /**
   * Detects the provider for a given model
   */
  detectProvider(model: string): string {
    // Exact match first
    if (this.modelToProviderMap[model]) {
      return this.modelToProviderMap[model];
    }

    // Prefix matching
    for (const [prefix, provider] of Object.entries(this.providerPrefixes)) {
      if (model.startsWith(prefix)) {
        return provider;
      }
    }

    // Default fallback
    throw new Error(`Unknown model: ${model}. Unable to determine provider.`);
  }

  /**
   * Gets all supported models for a provider
   */
  getModelsForProvider(provider: string): string[] {
    const models: string[] = [];
    
    for (const [model, modelProvider] of Object.entries(this.modelToProviderMap)) {
      if (modelProvider === provider) {
        models.push(model);
      }
    }

    return models;
  }

  /**
   * Gets all supported providers
   */
  getSupportedProviders(): string[] {
    const providers = new Set(Object.values(this.modelToProviderMap));
    return Array.from(providers);
  }

  /**
   * Validates if a model is supported
   */
  isModelSupported(model: string): boolean {
    try {
      this.detectProvider(model);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Gets provider information
   */
  getProviderInfo(provider: string): { name: string; models: string[]; endpoint: string } {
    const models = this.getModelsForProvider(provider);
    
    const endpoints: Record<string, string> = {
      openai: '/v1/openai',
      anthropic: '/v1/anthropic',
      google: '/v1/google',
    };

    return {
      name: provider,
      models,
      endpoint: endpoints[provider] || `/v1/${provider}`,
    };
  }

  /**
   * Adds a custom model mapping
   */
  addModelMapping(model: string, provider: string): void {
    this.modelToProviderMap[model] = provider;
  }

  /**
   * Adds a custom provider prefix
   */
  addProviderPrefix(prefix: string, provider: string): void {
    this.providerPrefixes[prefix] = provider;
  }

  /**
   * Gets model capabilities (if known)
   */
  getModelCapabilities(model: string): {
    supportsStreaming: boolean;
    maxTokens: number;
    supportsImages: boolean;
    supportsFunctions: boolean;
  } {
    const capabilities = {
      supportsStreaming: true,
      maxTokens: 4096,
      supportsImages: false,
      supportsFunctions: false,
    };

    // Model-specific capabilities
    if (model.includes('gpt-4')) {
      capabilities.maxTokens = 8192;
      capabilities.supportsFunctions = true;
    }
    
    if (model.includes('vision')) {
      capabilities.supportsImages = true;
    }

    if (model.includes('claude-3')) {
      capabilities.maxTokens = 200000;
    }

    if (model.includes('gemini')) {
      capabilities.maxTokens = 30720;
      capabilities.supportsImages = model.includes('vision') || model.includes('pro');
    }

    return capabilities;
  }
}
