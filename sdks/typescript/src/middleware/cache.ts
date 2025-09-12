import { ChatCompletionParams, ChatCompletionResponse, LeashConfig } from '../types';

export class CacheManager {
  private cache: Map<string, CacheEntry>;
  private config: LeashConfig;
  private maxSize: number;
  private defaultTTL: number;

  constructor(config: LeashConfig) {
    this.config = config;
    this.cache = new Map();
    this.maxSize = 1000; // Default max cache entries
    this.defaultTTL = 300000; // 5 minutes default TTL
  }

  async get(params: ChatCompletionParams): Promise<ChatCompletionResponse | null> {
    if (!this.config.cacheEnabled) {
      return null;
    }

    const key = this.generateCacheKey(params);
    const entry = this.cache.get(key);

    if (!entry) {
      return null;
    }

    // Check if entry is expired
    if (Date.now() > entry.expiresAt) {
      this.cache.delete(key);
      return null;
    }

    // Update access time for LRU
    entry.lastAccessed = Date.now();
    return entry.response;
  }

  async set(params: ChatCompletionParams, response: ChatCompletionResponse): Promise<void> {
    if (!this.config.cacheEnabled) {
      return;
    }

    // Don't cache streaming responses
    if (params.stream) {
      return;
    }

    // Don't cache if response contains errors
    if (!response || !response.choices || response.choices.length === 0) {
      return;
    }

    const key = this.generateCacheKey(params);
    const now = Date.now();

    const entry: CacheEntry = {
      response,
      createdAt: now,
      lastAccessed: now,
      expiresAt: now + this.defaultTTL,
      accessCount: 0,
    };

    // Ensure cache doesn't exceed max size
    if (this.cache.size >= this.maxSize) {
      this.evictLRU();
    }

    this.cache.set(key, entry);
  }

  clear(): void {
    this.cache.clear();
  }

  getStats(): CacheStats {
    const now = Date.now();
    let totalSize = 0;
    let expiredEntries = 0;

    for (const [key, entry] of this.cache.entries()) {
      totalSize += this.estimateEntrySize(entry);
      if (now > entry.expiresAt) {
        expiredEntries++;
      }
    }

    return {
      totalEntries: this.cache.size,
      totalSizeBytes: totalSize,
      expiredEntries,
      maxSize: this.maxSize,
      hitRate: this.calculateHitRate(),
    };
  }

  private generateCacheKey(params: ChatCompletionParams): string {
    // Create a deterministic cache key from request parameters
    const keyData = {
      model: params.model,
      messages: params.messages,
      temperature: params.temperature,
      max_tokens: params.max_tokens,
      top_p: params.top_p,
      frequency_penalty: params.frequency_penalty,
      presence_penalty: params.presence_penalty,
      stop: params.stop,
      user: params.user,
    };

    // Sort and stringify for consistent keys
    const keyString = JSON.stringify(keyData, Object.keys(keyData).sort());
    
    // Simple hash function (in production, use a proper hash)
    let hash = 0;
    for (let i = 0; i < keyString.length; i++) {
      const char = keyString.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }

    return `cache_${Math.abs(hash).toString(36)}`;
  }

  private evictLRU(): void {
    let oldestKey: string | null = null;
    let oldestTime = Date.now();

    for (const [key, entry] of this.cache.entries()) {
      if (entry.lastAccessed < oldestTime) {
        oldestTime = entry.lastAccessed;
        oldestKey = key;
      }
    }

    if (oldestKey) {
      this.cache.delete(oldestKey);
    }
  }

  private estimateEntrySize(entry: CacheEntry): number {
    // Rough estimation of memory usage
    const responseSize = JSON.stringify(entry.response).length;
    return responseSize * 2; // Account for object overhead
  }

  private calculateHitRate(): number {
    // This would be tracked properly in a real implementation
    return 0; // Placeholder
  }

  // Configuration methods
  setMaxSize(maxSize: number): void {
    this.maxSize = maxSize;
    
    // Evict entries if over new limit
    while (this.cache.size > maxSize) {
      this.evictLRU();
    }
  }

  setDefaultTTL(ttl: number): void {
    this.defaultTTL = ttl;
  }

  // Cache warming (pre-populate common requests)
  async warmCache(commonRequests: ChatCompletionParams[]): Promise<void> {
    // This would make actual requests to warm the cache
    // Implementation would depend on having a working gateway
  }
}

interface CacheEntry {
  response: ChatCompletionResponse;
  createdAt: number;
  lastAccessed: number;
  expiresAt: number;
  accessCount: number;
}

interface CacheStats {
  totalEntries: number;
  totalSizeBytes: number;
  expiredEntries: number;
  maxSize: number;
  hitRate: number;
}
