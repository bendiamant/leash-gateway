import React, { useState, useEffect } from 'react';
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { Activity, DollarSign, Zap, Shield, CheckCircle } from 'lucide-react';

interface MetricsDashboardProps {
  metrics: {
    totalRequests: number;
    totalCost: number;
    lastProvider: string;
    lastLatency: number;
  };
}

export function MetricsDashboard({ metrics }: MetricsDashboardProps) {
  const [realtimeData, setRealtimeData] = useState<any[]>([]);
  const [providerData, setProviderData] = useState<any[]>([
    { name: 'OpenAI', requests: 0, cost: 0, latency: 0, status: 'healthy' },
    { name: 'Anthropic', requests: 0, cost: 0, latency: 0, status: 'healthy' },
    { name: 'Google', requests: 0, cost: 0, latency: 0, status: 'healthy' },
  ]);

  // Simulate real-time metrics updates
  useEffect(() => {
    const interval = setInterval(() => {
      const now = new Date();
      const timeLabel = now.toLocaleTimeString();
      
      setRealtimeData(prev => {
        const newData = [...prev, {
          time: timeLabel,
          requests: metrics.totalRequests,
          cost: metrics.totalCost,
          latency: metrics.lastLatency || 0,
        }].slice(-20); // Keep last 20 data points
        
        return newData;
      });

      // Update provider data
      if (metrics.lastProvider) {
        setProviderData(prev => prev.map(provider => {
          if (provider.name.toLowerCase() === metrics.lastProvider) {
            return {
              ...provider,
              requests: provider.requests + 1,
              cost: provider.cost + (metrics.totalCost - provider.cost),
              latency: metrics.lastLatency || provider.latency,
              status: 'healthy',
            };
          }
          return provider;
        }));
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [metrics]);

  const COLORS = ['#3B82F6', '#10B981', '#F59E0B', '#EF4444'];

  const moduleStatus = [
    { name: 'Rate Limiter', status: 'active', requests: metrics.totalRequests },
    { name: 'Content Filter', status: 'active', blocked: 0 },
    { name: 'Cost Tracker', status: 'active', cost: metrics.totalCost },
    { name: 'Logger', status: 'active', logs: metrics.totalRequests },
  ];

  return (
    <div className="space-y-6">
      {/* Key Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricCard
          title="Total Requests"
          value={metrics.totalRequests.toString()}
          icon={<Activity className="w-5 h-5" />}
          color="blue"
        />
        <MetricCard
          title="Total Cost"
          value={`$${metrics.totalCost.toFixed(4)}`}
          icon={<DollarSign className="w-5 h-5" />}
          color="green"
        />
        <MetricCard
          title="Avg Latency"
          value={`${metrics.lastLatency || 0}ms`}
          icon={<Zap className="w-5 h-5" />}
          color="yellow"
        />
        <MetricCard
          title="Security Status"
          value="Protected"
          icon={<Shield className="w-5 h-5" />}
          color="green"
        />
      </div>

      {/* Real-time Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Request Rate Chart */}
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-lg font-semibold text-gray-800 mb-4">Request Rate</h3>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={realtimeData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="requests" stroke="#3B82F6" strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Cost Tracking Chart */}
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-lg font-semibold text-gray-800 mb-4">Cost Over Time</h3>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={realtimeData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis />
              <Tooltip formatter={(value) => [`$${Number(value).toFixed(4)}`, 'Cost']} />
              <Bar dataKey="cost" fill="#10B981" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Provider Status */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Provider Distribution */}
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-lg font-semibold text-gray-800 mb-4">Provider Usage</h3>
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie
                data={providerData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={({ name, requests }) => `${name}: ${requests}`}
                outerRadius={80}
                fill="#8884d8"
                dataKey="requests"
              >
                {providerData.map((_, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>

        {/* Module Status */}
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-lg font-semibold text-gray-800 mb-4">Module Status</h3>
          <div className="space-y-3">
            {moduleStatus.map((module) => (
              <div key={module.name} className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <CheckCircle className="w-4 h-4 text-green-500" />
                  <span className="text-sm font-medium text-gray-700">{module.name}</span>
                </div>
                <div className="text-sm text-gray-500">
                  {module.name === 'Rate Limiter' && `${module.requests} requests`}
                  {module.name === 'Content Filter' && `${module.blocked} blocked`}
                  {module.name === 'Cost Tracker' && `$${module.cost?.toFixed(4)}`}
                  {module.name === 'Logger' && `${module.logs} logs`}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Provider Health Status */}
      <div className="bg-white p-6 rounded-lg shadow-md">
        <h3 className="text-lg font-semibold text-gray-800 mb-4">Provider Health</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {providerData.map((provider) => (
            <div key={provider.name} className="border border-gray-200 rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <h4 className="font-medium text-gray-700">{provider.name}</h4>
                <div className={`px-2 py-1 rounded-full text-xs ${
                  provider.status === 'healthy' 
                    ? 'bg-green-100 text-green-800' 
                    : 'bg-red-100 text-red-800'
                }`}>
                  {provider.status}
                </div>
              </div>
              <div className="space-y-1 text-sm text-gray-600">
                <div>Requests: {provider.requests}</div>
                <div>Cost: ${provider.cost.toFixed(4)}</div>
                <div>Latency: {provider.latency}ms</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

interface MetricCardProps {
  title: string;
  value: string;
  icon: React.ReactNode;
  color: 'blue' | 'green' | 'yellow' | 'red';
}

function MetricCard({ title, value, icon, color }: MetricCardProps) {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-600 border-blue-200',
    green: 'bg-green-50 text-green-600 border-green-200',
    yellow: 'bg-yellow-50 text-yellow-600 border-yellow-200',
    red: 'bg-red-50 text-red-600 border-red-200',
  };

  return (
    <div className="bg-white p-6 rounded-lg shadow-md border border-gray-200">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600">{title}</p>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
        </div>
        <div className={`p-3 rounded-full ${colorClasses[color]}`}>
          {icon}
        </div>
      </div>
    </div>
  );
}
