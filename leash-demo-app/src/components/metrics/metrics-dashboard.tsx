'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell
} from 'recharts';
import {
  Activity,
  DollarSign,
  Zap,
  Shield,
  AlertCircle,
  CheckCircle,
  TrendingUp,
  TrendingDown,
  Clock,
  RefreshCw
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface MetricsDashboardProps {
  className?: string;
}

export function MetricsDashboard({ className }: MetricsDashboardProps) {
  const [metrics, setMetrics] = useState<any>(null);
  const [health, setHealth] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [metricsError, setMetricsError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
  
  // Baseline from when health checks were moved to direct connections
  // This represents historical health check traffic that should be excluded
  const HEALTH_CHECK_BASELINE = 168;

  useEffect(() => {
    fetchMetrics();
    fetchHealth();
    const interval = setInterval(() => {
      console.log('[MetricsDashboard] Refreshing metrics...');
      fetchMetrics();
      fetchHealth();
    }, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchMetrics = async () => {
    try {
      const response = await fetch('/api/metrics');
      const data = await response.json();
      console.log('[MetricsDashboard] Received metrics:', data.data?.requests?.count);
      setMetrics(data.data);
      setMetricsError(data.error || null);
      setLoading(false);
      setLastUpdate(new Date());
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
      setMetricsError('Failed to connect to metrics API');
      setLoading(false);
    }
  };

  const fetchHealth = async () => {
    try {
      const response = await fetch('/api/health');
      const data = await response.json();
      setHealth(data);
    } catch (error) {
      console.error('Failed to fetch health:', error);
    }
  };

  // Process metrics data for charts
  const getRequestData = () => {
    if (!metrics?.requests?.count) return [];
    const processed = metrics.requests.count.map((item: any) => {
      const rawCount = parseFloat(item.value[1]);
      // Subtract baseline health check traffic to show only real app requests
      const realRequests = Math.max(0, rawCount - HEALTH_CHECK_BASELINE);
      
      const result = {
        // Handle both formats: provider or envoy_cluster_name
        provider: item.metric.provider || 
                  item.metric.envoy_cluster_name?.replace('_cluster', '').replace('_api', ''),
        requests: realRequests,
        rawRequests: rawCount // Keep raw count for reference
      };
      
      console.log(`[MetricsDashboard] ${result.provider}: Raw=${rawCount}, Real=${realRequests}`);
      return result;
    });
    return processed;
  };

  const getLatencyData = () => {
    if (!metrics?.requests?.latency) return [];
    return metrics.requests.latency.map((item: any) => ({
      // Handle both formats: provider or envoy_cluster_name
      provider: item.metric.provider || 
                item.metric.envoy_cluster_name?.replace('_cluster', '').replace('_api', ''),
      latency: parseFloat(item.value[1]) * 1000 // Convert to ms
    }));
  };

  const getCostData = () => {
    if (!metrics?.cost) return [];
    return metrics.cost.map((item: any) => ({
      provider: item.metric.provider,
      model: item.metric.model,
      cost: parseFloat(item.value[1])
    }));
  };

  const getModuleStatus = () => {
    if (!health?.modules) return [];
    return health.modules;
  };

  const COLORS = {
    openai: '#10b981',
    anthropic: '#a855f7',
    google: '#3b82f6'
  };

  return (
    <div className={cn("space-y-6", className)}>
      {/* Status Messages */}
      {metricsError && (
        <div className="flex items-center gap-2 p-3 bg-red-50 border border-red-200 rounded-lg">
          <AlertCircle className="h-4 w-4 text-red-600" />
          <span className="text-sm text-red-800">
            Metrics Error: {metricsError}
          </span>
        </div>
      )}
      
      {!metricsError && metrics && getRequestData().length === 0 && (
        <div className="flex items-center gap-2 p-3 bg-blue-50 border border-blue-200 rounded-lg">
          <AlertCircle className="h-4 w-4 text-blue-600" />
          <span className="text-sm text-blue-800">
            No application traffic yet. Send requests through the gateway to see metrics.
            (Health checks are excluded from metrics)
          </span>
        </div>
      )}
      
      {!metricsError && metrics && getRequestData().some(d => d.requests > 0) && (
        <div className="flex items-center justify-between p-3 bg-green-50 border border-green-200 rounded-lg">
          <div className="flex items-center gap-2">
            <CheckCircle className="h-4 w-4 text-green-600" />
            <span className="text-sm text-green-800">
              Showing real application traffic only. Health check requests (168 baseline per provider) are excluded.
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="h-3 w-3 text-green-600 animate-pulse" />
            <span className="text-xs text-green-700">
              Updates every 10s • Last: {lastUpdate.toLocaleTimeString()}
            </span>
          </div>
        </div>
      )}

      {/* Manual Refresh Button */}
      <div className="flex justify-end mb-4">
        <Button
          onClick={() => {
            console.log('[MetricsDashboard] Manual refresh triggered');
            fetchMetrics();
            fetchHealth();
          }}
          size="sm"
          variant="outline"
          className="flex items-center gap-2"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh Now
        </Button>
      </div>

      {/* Health Status */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Gateway Status</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {health?.status === 'healthy' ? (
                <div className="flex items-center gap-2 text-green-600">
                  <CheckCircle className="h-5 w-5" />
                  Healthy
                </div>
              ) : health?.status === 'degraded' ? (
                <div className="flex items-center gap-2 text-yellow-600">
                  <AlertCircle className="h-5 w-5" />
                  Degraded
                </div>
              ) : (
                <div className="flex items-center gap-2 text-red-600">
                  <AlertCircle className="h-5 w-5" />
                  Unhealthy
                </div>
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {health?.services?.filter((s: any) => s.status === 'healthy').length || 0}/{health?.services?.length || 0} services online
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {(() => {
                const data = getRequestData();
                const total = data.reduce((sum, item) => sum + item.requests, 0);
                console.log('[MetricsDashboard] Total calculation:', data, '=', total);
                return total.toFixed(0);
              })()}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Across all providers
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {getLatencyData().length > 0 
                ? (getLatencyData().reduce((sum, item) => sum + item.latency, 0) / getLatencyData().length).toFixed(0)
                : 0}ms
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              P95 latency
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${getCostData().reduce((sum, item) => sum + item.cost, 0).toFixed(2)}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Since last reset
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Charts */}
      <Tabs defaultValue="requests" className="space-y-4">
        <TabsList>
          <TabsTrigger value="requests">Requests</TabsTrigger>
          <TabsTrigger value="latency">Latency</TabsTrigger>
          <TabsTrigger value="cost">Cost</TabsTrigger>
          <TabsTrigger value="modules">Modules</TabsTrigger>
        </TabsList>

        <TabsContent value="requests" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Request Distribution</CardTitle>
              <CardDescription>
                Requests per provider over time
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={getRequestData()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="provider" />
                  <YAxis />
                  <Tooltip />
                  <Bar dataKey="requests" fill="#3b82f6" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="latency" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Provider Latency</CardTitle>
              <CardDescription>
                Response time per provider (ms)
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={getLatencyData()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="provider" />
                  <YAxis />
                  <Tooltip />
                  <Bar dataKey="latency" fill="#10b981" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="cost" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Cost Breakdown</CardTitle>
              <CardDescription>
                Cumulative cost per provider
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={getCostData()}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={(entry) => `${entry.provider}: $${entry.cost.toFixed(2)}`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="cost"
                  >
                    {getCostData().map((entry: any, index: number) => (
                      <Cell key={`cell-${index}`} fill={COLORS[entry.provider as keyof typeof COLORS] || '#8884d8'} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="modules" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Module Status</CardTitle>
              <CardDescription>
                Active gateway modules
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {getModuleStatus().map((module: any) => (
                  <div key={module.name} className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className={cn(
                        "w-2 h-2 rounded-full",
                        module.healthy ? "bg-green-500" : "bg-red-500"
                      )} />
                      <span className="font-medium capitalize">{module.name}</span>
                    </div>
                    <Badge variant={module.enabled ? "default" : "secondary"}>
                      {module.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Service Health Grid */}
      <Card>
        <CardHeader>
          <CardTitle>Service Health</CardTitle>
          <CardDescription>
            Real-time health status of gateway components
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            {health?.services?.map((service: any) => (
              <div key={service.service} className="flex flex-col items-center p-3 rounded-lg border">
                <div className={cn(
                  "w-12 h-12 rounded-full flex items-center justify-center mb-2",
                  service.status === 'healthy' ? "bg-green-100" : 
                  service.status === 'degraded' ? "bg-yellow-100" : "bg-red-100"
                )}>
                  {service.status === 'healthy' ? (
                    <CheckCircle className={cn("h-6 w-6", "text-green-600")} />
                  ) : (
                    <AlertCircle className={cn(
                      "h-6 w-6",
                      service.status === 'degraded' ? "text-yellow-600" : "text-red-600"
                    )} />
                  )}
                </div>
                <span className="text-sm font-medium capitalize">
                  {service.service.replace('provider-', '')}
                </span>
                {service.latency && (
                  <span className="text-xs text-muted-foreground">
                    {service.latency}ms
                  </span>
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
