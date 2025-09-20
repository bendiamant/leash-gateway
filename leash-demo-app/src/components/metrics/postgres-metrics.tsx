'use client';

import { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Activity,
  Clock,
  RefreshCw,
  Database,
  CheckCircle,
  AlertCircle
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface PostgresMetricsProps {
  className?: string;
}

export function PostgresMetrics({ className }: PostgresMetricsProps) {
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());

  const fetchMetrics = async () => {
    try {
      const response = await fetch('/api/metrics-pg');
      const data = await response.json();
      setMetrics(data.data);
      setLoading(false);
      setLastUpdate(new Date());
      console.log('[PostgresMetrics] Data received:', data);
    } catch (error) {
      console.error('Failed to fetch PostgreSQL metrics:', error);
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 2000); // Update every 2 seconds (much faster!)
    return () => clearInterval(interval);
  }, []);

  const formatLatency = (ms: number | null) => {
    if (ms === null || ms === undefined) return 'N/A';
    return `${Math.round(ms)}ms`;
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  };

  return (
    <div className={cn("space-y-4", className)}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Database className="h-5 w-5 text-blue-600" />
          <h3 className="text-lg font-semibold">Real-time Metrics (PostgreSQL)</h3>
          <Badge variant="outline" className="bg-green-50">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse mr-1" />
            Live
          </Badge>
        </div>
        
        <div className="flex items-center gap-3">
          <span className="text-xs text-muted-foreground">
            Updates every 2s • Last: {lastUpdate.toLocaleTimeString()}
          </span>
          <Button
            onClick={fetchMetrics}
            size="sm"
            variant="outline"
            className="flex items-center gap-2"
          >
            <RefreshCw className="h-3 w-3" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {metrics?.summary?.map((provider: any) => (
          <Card key={provider.provider} className="border-2">
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-medium capitalize">
                  {provider.provider}
                </CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </div>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground">Requests</span>
                <span className="text-lg font-bold">{provider.request_count}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground">Avg Latency</span>
                <span className="text-sm font-medium">{formatLatency(provider.avg_latency)}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-xs text-muted-foreground">Errors</span>
                <span className={cn(
                  "text-sm font-medium",
                  provider.error_count > 0 ? "text-red-600" : "text-green-600"
                )}>
                  {provider.error_count || 0}
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
        
        {(!metrics?.summary || metrics.summary.length === 0) && (
          <Card className="col-span-3 border-dashed">
            <CardContent className="flex flex-col items-center justify-center py-8">
              <AlertCircle className="h-8 w-8 text-muted-foreground mb-2" />
              <p className="text-sm text-muted-foreground">
                No requests yet. Send a message through the gateway to see real-time metrics!
              </p>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Recent Requests */}
      {metrics?.recent && metrics.recent.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Recent Requests</CardTitle>
            <CardDescription>Last 10 requests through the gateway</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {metrics.recent.map((req: any) => (
                <div key={req.request_id} className="flex items-center justify-between p-2 rounded-lg hover:bg-muted/50 transition-colors">
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      "w-2 h-2 rounded-full",
                      req.status_code < 400 ? "bg-green-500" : "bg-red-500"
                    )} />
                    <div>
                      <span className="text-sm font-medium capitalize">{req.provider}</span>
                      <span className="text-xs text-muted-foreground ml-2">{req.path}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <Badge variant="outline" className="text-xs">
                      {formatLatency(req.processing_time_ms)}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {formatTime(req.created_at)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

