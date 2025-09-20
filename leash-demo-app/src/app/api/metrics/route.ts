import { NextResponse } from 'next/server';

export const runtime = 'edge';

const PROMETHEUS_URL = process.env.GATEWAY_METRICS_URL || 'http://localhost:9091';

interface MetricQuery {
  metric: string;
  timeRange?: string;
  step?: string;
}

// Prometheus queries for gateway metrics - using actual Envoy metrics
const queries = {
  // Use Envoy's native metrics - these will now only show real app traffic
  // since health checks go direct to providers
  requestCount: 'sum(envoy_cluster_upstream_rq_total) by (envoy_cluster_name)',
  latency: 'histogram_quantile(0.95, sum(rate(envoy_cluster_upstream_rq_time_bucket[5m])) by (envoy_cluster_name, le))',
  errorRate: 'sum(rate(envoy_cluster_upstream_rq_xx{envoy_response_code_class="5"}[5m])) by (envoy_cluster_name)',
  // These metrics aren't available yet
  costTotal: 'sum(leash_cost_usd_total) by (provider, model)',
  activeModules: 'up{job="leash-module-host"}',
  rateLimitHits: 'sum(rate(envoy_cluster_upstream_rq_timeout[5m]))'
};

async function queryPrometheus(query: string, params?: Record<string, string>) {
  const searchParams = new URLSearchParams({
    query,
    ...params
  });

  try {
    const response = await fetch(`${PROMETHEUS_URL}/api/v1/query?${searchParams}`, {
      headers: {
        'Accept': 'application/json'
      }
    });

    if (!response.ok) {
      throw new Error(`Prometheus query failed: ${response.statusText}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data = await response.json();
      return data.data?.result || [];
    } else {
      // Prometheus might return text format, not JSON
      console.warn('Prometheus returned non-JSON response');
      return [];
    }
  } catch (error) {
    console.error('Prometheus query error:', error);
    return [];
  }
}

export async function GET(req: Request) {
  const { searchParams } = new URL(req.url);
  const metricType = searchParams.get('type') || 'all';

  try {
    let metrics: Record<string, any> = {};

    if (metricType === 'all' || metricType === 'requests') {
      const [requestCount, latency, errorRate] = await Promise.all([
        queryPrometheus(queries.requestCount),
        queryPrometheus(queries.latency),
        queryPrometheus(queries.errorRate)
      ]);

      metrics.requests = {
        count: requestCount || [],
        latency: latency || [],
        errorRate: errorRate || []
      };
    }

    if (metricType === 'all' || metricType === 'cost') {
      const costData = await queryPrometheus(queries.costTotal);
      metrics.cost = costData || [];
    }

    if (metricType === 'all' || metricType === 'modules') {
      const moduleData = await queryPrometheus(queries.activeModules);
      metrics.modules = moduleData || [];
    }

    if (metricType === 'all' || metricType === 'ratelimit') {
      const rateLimitData = await queryPrometheus(queries.rateLimitHits);
      metrics.rateLimit = rateLimitData || [];
    }

    // Return real data (even if empty)
    return NextResponse.json({
      success: true,
      data: metrics,
      timestamp: new Date().toISOString(),
      prometheusUrl: PROMETHEUS_URL
    });
  } catch (error) {
    console.error('Metrics API error:', error);
    
    // Return error with empty metrics
    return NextResponse.json({
      success: false,
      data: {
        requests: { count: [], latency: [], errorRate: [] },
        cost: [],
        modules: [],
        rateLimit: []
      },
      error: error instanceof Error ? error.message : 'Failed to fetch metrics from Prometheus',
      prometheusUrl: PROMETHEUS_URL,
      timestamp: new Date().toISOString()
    });
  }
}

function getMockMetrics() {
  return {
    requests: {
      count: [
        { metric: { provider: 'openai' }, value: [Date.now() / 1000, '125'] },
        { metric: { provider: 'anthropic' }, value: [Date.now() / 1000, '87'] },
        { metric: { provider: 'google' }, value: [Date.now() / 1000, '43'] }
      ],
      latency: [
        { metric: { provider: 'openai' }, value: [Date.now() / 1000, '0.245'] },
        { metric: { provider: 'anthropic' }, value: [Date.now() / 1000, '0.312'] },
        { metric: { provider: 'google' }, value: [Date.now() / 1000, '0.198'] }
      ],
      errorRate: [
        { metric: { provider: 'openai' }, value: [Date.now() / 1000, '0.002'] },
        { metric: { provider: 'anthropic' }, value: [Date.now() / 1000, '0.001'] },
        { metric: { provider: 'google' }, value: [Date.now() / 1000, '0.003'] }
      ]
    },
    cost: [
      { metric: { provider: 'openai', model: 'gpt-4o-mini' }, value: [Date.now() / 1000, '2.45'] },
      { metric: { provider: 'anthropic', model: 'claude-3-sonnet' }, value: [Date.now() / 1000, '1.87'] },
      { metric: { provider: 'google', model: 'gemini-1.5-flash' }, value: [Date.now() / 1000, '0.92'] }
    ],
    modules: [
      { metric: { module: 'ratelimiter' }, value: [Date.now() / 1000, '1'] },
      { metric: { module: 'contentfilter' }, value: [Date.now() / 1000, '1'] },
      { metric: { module: 'costtracker' }, value: [Date.now() / 1000, '1'] },
      { metric: { module: 'logger' }, value: [Date.now() / 1000, '1'] }
    ],
    rateLimit: [
      { metric: {}, value: [Date.now() / 1000, '3'] }
    ]
  };
}
