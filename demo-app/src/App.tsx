import React, { useState } from 'react';
import { ChatInterface } from './components/ChatInterface';
import { MetricsDashboard } from './components/MetricsDashboard';
import { Shield, BarChart3, MessageSquare, Github, ExternalLink } from 'lucide-react';

function App() {
  const [activeTab, setActiveTab] = useState<'chat' | 'metrics'>('chat');
  const [selectedProvider, setSelectedProvider] = useState('openai');
  const [metrics, setMetrics] = useState({
    totalRequests: 0,
    totalCost: 0,
    lastProvider: '',
    lastLatency: 0,
  });

  const handleProviderSwitch = (provider: string) => {
    setSelectedProvider(provider);
  };

  const handleMetricsUpdate = (newMetrics: any) => {
    setMetrics(prev => ({ ...prev, ...newMetrics }));
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo and Title */}
            <div className="flex items-center space-x-3">
              <div className="flex items-center justify-center w-10 h-10 bg-blue-600 rounded-lg">
                <Shield className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-gray-900">Leash Security Gateway</h1>
                <p className="text-sm text-gray-500">LLM Security & Governance Demo</p>
              </div>
            </div>

            {/* Navigation */}
            <nav className="flex space-x-4">
              <button
                onClick={() => setActiveTab('chat')}
                className={`flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'chat'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }`}
              >
                <MessageSquare className="w-4 h-4" />
                <span>Chat Demo</span>
              </button>
              <button
                onClick={() => setActiveTab('metrics')}
                className={`flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === 'metrics'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                }`}
              >
                <BarChart3 className="w-4 h-4" />
                <span>Metrics</span>
              </button>
            </nav>

            {/* Links */}
            <div className="flex items-center space-x-3">
              <a
                href="https://github.com/bendiamant/leash-gateway"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center space-x-1 text-gray-600 hover:text-gray-900"
              >
                <Github className="w-4 h-4" />
                <span className="text-sm">GitHub</span>
                <ExternalLink className="w-3 h-3" />
              </a>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Status Banner */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <Shield className="w-5 h-5 text-blue-600" />
              <div>
                <h3 className="text-sm font-medium text-blue-800">Gateway Status: Active</h3>
                <p className="text-sm text-blue-600">
                  All requests are being processed through the Leash Security Gateway with 
                  rate limiting, content filtering, and cost tracking enabled.
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-4 text-sm text-blue-600">
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>Gateway: Online</span>
              </div>
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>Modules: 4/4 Active</span>
              </div>
            </div>
          </div>
        </div>

        {/* Tab Content */}
        <div className="bg-white rounded-lg shadow-lg overflow-hidden">
          {activeTab === 'chat' && (
            <div className="h-[600px]">
              <ChatInterface
                selectedProvider={selectedProvider}
                onProviderSwitch={handleProviderSwitch}
                onMetricsUpdate={handleMetricsUpdate}
              />
            </div>
          )}

          {activeTab === 'metrics' && (
            <div className="p-6">
              <MetricsDashboard metrics={metrics} />
            </div>
          )}
        </div>

        {/* Feature Highlights */}
        <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-6">
          <FeatureCard
            icon={<MessageSquare className="w-6 h-6" />}
            title="Multi-Provider Support"
            description="Seamlessly switch between OpenAI, Anthropic, and Google without changing your code."
            color="blue"
          />
          <FeatureCard
            icon={<Shield className="w-6 h-6" />}
            title="Security & Governance"
            description="Built-in rate limiting, content filtering, and policy enforcement for all LLM traffic."
            color="green"
          />
          <FeatureCard
            icon={<BarChart3 className="w-6 h-6" />}
            title="Real-time Monitoring"
            description="Complete observability with cost tracking, performance metrics, and health monitoring."
            color="purple"
          />
        </div>

        {/* Integration Example */}
        <div className="mt-8 bg-gray-900 rounded-lg p-6 text-white">
          <h3 className="text-lg font-semibold mb-4">Integration Example</h3>
          <div className="bg-gray-800 rounded-md p-4 overflow-x-auto">
            <pre className="text-sm">
              <code>{`// Before: Direct provider call
const client = new OpenAI({
  apiKey: "sk-...",
  baseURL: "https://api.openai.com/v1"
});

// After: Route through Leash Gateway (one line change!)
const client = new OpenAI({
  apiKey: "sk-...",
  baseURL: "https://gateway.company.com/v1/openai"
});

// Application code remains identical
const response = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello!" }]
});`}</code>
            </pre>
          </div>
          <p className="text-gray-300 text-sm mt-3">
            ✨ <strong>Minimal integration:</strong> Just change the base URL and get centralized security, 
            governance, and observability for all your LLM traffic!
          </p>
        </div>
      </main>

      {/* Footer */}
      <footer className="bg-white border-t border-gray-200 py-8 mt-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Shield className="w-5 h-5 text-blue-600" />
              <span className="text-gray-600">Built with ❤️ by the Leash Security team</span>
            </div>
            <div className="text-sm text-gray-500">
              Licensed under Apache 2.0 | 
              <a href="https://github.com/bendiamant/leash-gateway" className="ml-1 text-blue-600 hover:text-blue-800">
                View on GitHub
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}

interface FeatureCardProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  color: 'blue' | 'green' | 'purple';
}

function FeatureCard({ icon, title, description, color }: FeatureCardProps) {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-600 border-blue-200',
    green: 'bg-green-50 text-green-600 border-green-200',
    purple: 'bg-purple-50 text-purple-600 border-purple-200',
  };

  return (
    <div className="bg-white p-6 rounded-lg shadow-md border border-gray-200">
      <div className={`inline-flex p-3 rounded-lg ${colorClasses[color]} mb-4`}>
        {icon}
      </div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-600">{description}</p>
    </div>
  );
}

export default App;
