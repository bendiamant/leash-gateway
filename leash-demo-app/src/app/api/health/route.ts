import { NextResponse } from 'next/server';

export const runtime = 'edge';

const GATEWAY_URL = process.env.GATEWAY_URL || 'http://localhost:8080';
const MODULE_HOST_URL = process.env.GATEWAY_HEALTH_URL || 'http://localhost:8081';

interface HealthCheck {
  service: string;
  status: 'healthy' | 'unhealthy' | 'degraded';
  latency?: number;
  error?: string;
  metadata?: Record<string, any>;
}

async function checkServiceHealth(url: string, service: string): Promise<HealthCheck> {
  const startTime = Date.now();
  
  try {
    const response = await fetch(`${url}/health`, {
      signal: AbortSignal.timeout(5000) // 5 second timeout
    });
    
    const latency = Date.now() - startTime;
    
    if (response.ok) {
      const data = await response.json().catch(() => ({}));
      return {
        service,
        status: 'healthy',
        latency,
        metadata: data
      };
    } else {
      return {
        service,
        status: 'degraded',
        latency,
        error: `HTTP ${response.status}: ${response.statusText}`
      };
    }
  } catch (error) {
    return {
      service,
      status: 'unhealthy',
      latency: Date.now() - startTime,
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }
}

async function checkProviderHealth(provider: string): Promise<HealthCheck> {
  const startTime = Date.now();
  
  try {
    // Direct health check to providers (bypass gateway to avoid polluting metrics)
    const providerUrls: Record<string, string> = {
      openai: 'https://api.openai.com/v1/models',
      anthropic: 'https://api.anthropic.com/v1/models',
      google: 'https://generativelanguage.googleapis.com/v1beta/models'
    };
    
    const response = await fetch(providerUrls[provider] || `${GATEWAY_URL}/health`, {
      headers: {
        'Authorization': `Bearer ${process.env[`${provider.toUpperCase()}_API_KEY`] || 'test'}`
      },
      signal: AbortSignal.timeout(5000)
    });
    
    const latency = Date.now() - startTime;
    
    if (response.ok || response.status === 401) { // 401 means gateway is working but API key might be invalid
      return {
        service: `provider-${provider}`,
        status: response.ok ? 'healthy' : 'degraded',
        latency,
        metadata: { reachable: true }
      };
    } else {
      return {
        service: `provider-${provider}`,
        status: 'unhealthy',
        latency,
        error: `HTTP ${response.status}`
      };
    }
  } catch (error) {
    return {
      service: `provider-${provider}`,
      status: 'unhealthy',
      latency: Date.now() - startTime,
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }
}

export async function GET() {
  try {
    // Check all services in parallel
    const [gateway, moduleHost, openai, anthropic, google] = await Promise.all([
      checkServiceHealth(GATEWAY_URL, 'gateway'),
      checkServiceHealth(MODULE_HOST_URL, 'module-host'),
      checkProviderHealth('openai'),
      checkProviderHealth('anthropic'),
      checkProviderHealth('google')
    ]);

    const healthChecks = [gateway, moduleHost, openai, anthropic, google];
    
    // Calculate overall status
    const hasUnhealthy = healthChecks.some(check => check.status === 'unhealthy');
    const hasDegraded = healthChecks.some(check => check.status === 'degraded');
    
    let overallStatus: 'healthy' | 'degraded' | 'unhealthy';
    if (hasUnhealthy) {
      overallStatus = 'unhealthy';
    } else if (hasDegraded) {
      overallStatus = 'degraded';
    } else {
      overallStatus = 'healthy';
    }

    // Get module status (mock for now if module host is unavailable)
    const modules = moduleHost.status === 'healthy' ? 
      await getModuleStatus() : 
      getMockModuleStatus();

    return NextResponse.json({
      status: overallStatus,
      timestamp: new Date().toISOString(),
      services: healthChecks,
      modules,
      metadata: {
        version: '1.0.0',
        environment: process.env.NODE_ENV || 'development'
      }
    });
  } catch (error) {
    return NextResponse.json({
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      error: error instanceof Error ? error.message : 'Unknown error',
      services: [],
      modules: getMockModuleStatus()
    }, { status: 503 });
  }
}

async function getModuleStatus() {
  try {
    const response = await fetch(`${MODULE_HOST_URL}/modules/status`);
    if (response.ok) {
      return await response.json();
    }
  } catch (error) {
    console.error('Failed to fetch module status:', error);
  }
  return getMockModuleStatus();
}

function getMockModuleStatus() {
  return [
    { name: 'ratelimiter', enabled: true, healthy: true },
    { name: 'contentfilter', enabled: true, healthy: true },
    { name: 'costtracker', enabled: true, healthy: true },
    { name: 'logger', enabled: true, healthy: true }
  ];
}
