import React, { useState, useEffect, useRef } from 'react';
import { Send, Bot, AlertCircle, Zap, DollarSign } from 'lucide-react';

interface Message {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

interface ChatMessage extends Message {
  id: string;
  timestamp: Date;
  provider?: string;
  cost?: number;
  latency?: number;
  error?: string;
}

interface ChatCompletionParams {
  model: string;
  messages: Message[];
  temperature?: number;
  max_tokens?: number;
}

interface ChatCompletionResponse {
  choices: Array<{
    message: Message;
  }>;
  usage: {
    total_tokens: number;
    cost_usd?: number;
  };
}

interface LeashError extends Error {
  code: string;
  message: string;
}

// Mock LeashLLM for demo purposes
class LeashLLM {
  constructor(_config: any) {}
  
  async chatCompletions(params: ChatCompletionParams): Promise<ChatCompletionResponse> {
    // Mock implementation for demo
    return {
      choices: [{
        message: {
          role: 'assistant',
          content: `Mock response from ${params.model}. In a real implementation, this would route through the Leash Gateway to the actual provider.`
        }
      }],
      usage: {
        total_tokens: 50,
        cost_usd: 0.001
      }
    };
  }
  
  addEventListener(_listener: any) {}
  removeEventListener(_listener: any) {}
}

interface ChatInterfaceProps {
  selectedProvider: string;
  onProviderSwitch: (provider: string) => void;
  onMetricsUpdate: (metrics: any) => void;
}

export function ChatInterface({ selectedProvider, onProviderSwitch, onMetricsUpdate }: ChatInterfaceProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: '1',
      role: 'assistant',
      content: 'Hello! I\'m powered by the Leash Security Gateway. Try switching between different providers to see the magic! 🎉',
      timestamp: new Date(),
    }
  ]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [totalCost, setTotalCost] = useState(0);
  const [totalRequests, setTotalRequests] = useState(0);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Initialize Leash client
  const leashClient = new LeashLLM({
    gatewayUrl: import.meta.env.VITE_GATEWAY_URL || 'http://localhost:8080',
    apiKey: import.meta.env.VITE_API_KEY || 'demo-key',
    fallbackProviders: ['openai', 'anthropic', 'google'],
    debugMode: true,
  });

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Listen to SDK events for real-time metrics
  useEffect(() => {
    const handleEvent = (event: any) => {
      if (event.type === 'response') {
        setTotalRequests(prev => prev + 1);
        if (event.cost) {
          setTotalCost(prev => prev + event.cost);
        }
        onMetricsUpdate({
          totalRequests: totalRequests + 1,
          totalCost: totalCost + (event.cost || 0),
          lastProvider: event.provider,
          lastLatency: event.latency,
        });
      }
    };

    leashClient.addEventListener(handleEvent);
    return () => leashClient.removeEventListener(handleEvent);
  }, [totalRequests, totalCost, onMetricsUpdate]);

  const getModelForProvider = (provider: string): string => {
    const modelMap: Record<string, string> = {
      openai: 'gpt-4o-mini',
      anthropic: 'claude-3-sonnet-20240229',
      google: 'gemini-1.5-flash',
    };
    return modelMap[provider] || 'gpt-4o-mini';
  };

  const sendMessage = async () => {
    if (!input.trim() || loading) return;

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setLoading(true);
    setError(null);

    try {
      const model = getModelForProvider(selectedProvider);
      const startTime = Date.now();

      const params: ChatCompletionParams = {
        model,
        messages: messages.map(m => ({ role: m.role, content: m.content })).concat([
          { role: 'user', content: input }
        ]),
        temperature: 0.7,
        max_tokens: 1000,
      };

      const response = await leashClient.chatCompletions(params);
      const latency = Date.now() - startTime;

      const assistantMessage: ChatMessage = {
        id: Date.now().toString() + '_assistant',
        role: 'assistant',
        content: response.choices[0].message.content,
        timestamp: new Date(),
        provider: selectedProvider,
        cost: response.usage.cost_usd,
        latency,
      };

      setMessages(prev => [...prev, assistantMessage]);
      
      // Update metrics
      if (response.usage.cost_usd) {
        setTotalCost(prev => prev + response.usage.cost_usd!);
      }
      setTotalRequests(prev => prev + 1);

    } catch (err) {
      const leashError = err as LeashError;
      const errorMessage: ChatMessage = {
        id: Date.now().toString() + '_error',
        role: 'assistant',
        content: `❌ Error: ${leashError.message}`,
        timestamp: new Date(),
        provider: selectedProvider,
        error: leashError.code,
      };

      setMessages(prev => [...prev, errorMessage]);
      setError(leashError.message);
      
      console.error('Chat error:', leashError);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  const clearChat = () => {
    setMessages([{
      id: '1',
      role: 'assistant',
      content: 'Chat cleared! Ready for a new conversation. 🚀',
      timestamp: new Date(),
    }]);
    setError(null);
  };

  return (
    <div className="flex flex-col h-full bg-white rounded-lg shadow-lg">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200">
        <div className="flex items-center space-x-3">
          <Bot className="w-6 h-6 text-blue-600" />
          <h2 className="text-lg font-semibold text-gray-800">Leash Gateway Chat</h2>
        </div>
        
        <div className="flex items-center space-x-4">
          {/* Provider Selector */}
          <select
            value={selectedProvider}
            onChange={(e) => onProviderSwitch(e.target.value)}
            className="px-3 py-1 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="openai">OpenAI (GPT-4o-mini)</option>
            <option value="anthropic">Anthropic (Claude-3-Sonnet)</option>
            <option value="google">Google (Gemini-1.5-Flash)</option>
          </select>

          {/* Quick Stats */}
          <div className="flex items-center space-x-2 text-sm text-gray-600">
            <DollarSign className="w-4 h-4" />
            <span>${totalCost.toFixed(4)}</span>
            <span className="text-gray-400">|</span>
            <span>{totalRequests} requests</span>
          </div>

          <button
            onClick={clearChat}
            className="px-3 py-1 text-sm text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded-md transition-colors"
          >
            Clear
          </button>
        </div>
      </div>

      {/* Error Banner */}
      {error && (
        <div className="p-3 bg-red-50 border-l-4 border-red-400 text-red-700 text-sm">
          <div className="flex items-center">
            <AlertCircle className="w-4 h-4 mr-2" />
            {error}
          </div>
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.map((message) => (
          <div
            key={message.id}
            className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                message.role === 'user'
                  ? 'bg-blue-600 text-white'
                  : message.error
                  ? 'bg-red-100 text-red-800 border border-red-200'
                  : 'bg-gray-100 text-gray-800'
              }`}
            >
              {/* Message header for assistant */}
              {message.role === 'assistant' && !message.error && (
                <div className="flex items-center justify-between mb-2 text-xs text-gray-500">
                  <div className="flex items-center space-x-1">
                    <Bot className="w-3 h-3" />
                    <span>{message.provider || 'Assistant'}</span>
                  </div>
                  {message.latency && (
                    <div className="flex items-center space-x-1">
                      <Zap className="w-3 h-3" />
                      <span>{message.latency}ms</span>
                    </div>
                  )}
                </div>
              )}

              {/* Message content */}
              <div className="whitespace-pre-wrap">{message.content}</div>

              {/* Message footer */}
              <div className="flex items-center justify-between mt-2 text-xs opacity-70">
                <span>{message.timestamp.toLocaleTimeString()}</span>
                {message.cost && (
                  <span className="flex items-center space-x-1">
                    <DollarSign className="w-3 h-3" />
                    <span>${message.cost.toFixed(4)}</span>
                  </span>
                )}
              </div>
            </div>
          </div>
        ))}

        {/* Loading indicator */}
        {loading && (
          <div className="flex justify-start">
            <div className="bg-gray-100 text-gray-800 px-4 py-2 rounded-lg">
              <div className="flex items-center space-x-2">
                <div className="flex space-x-1">
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                </div>
                <span className="text-sm">Thinking via {selectedProvider}...</span>
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="p-4 border-t border-gray-200">
        <div className="flex space-x-3">
          <div className="flex-1">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder={`Type your message... (will be processed by ${selectedProvider})`}
              className="w-full px-3 py-2 border border-gray-300 rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              rows={input.split('\n').length || 1}
              disabled={loading}
            />
          </div>
          <button
            onClick={sendMessage}
            disabled={loading || !input.trim()}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center space-x-2"
          >
            <Send className="w-4 h-4" />
            <span>Send</span>
          </button>
        </div>

        {/* Quick Actions */}
        <div className="flex flex-wrap gap-2 mt-3">
          {[
            "Explain quantum computing",
            "Write a poem about AI",
            "Compare providers",
            "What's the weather like?",
            "Help me code a function"
          ].map((prompt) => (
            <button
              key={prompt}
              onClick={() => setInput(prompt)}
              className="px-3 py-1 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
              disabled={loading}
            >
              {prompt}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
