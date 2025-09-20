'use client';

import { useState, useRef, useEffect } from 'react';
import { useChat } from '@ai-sdk/react';
import { DefaultChatTransport } from 'ai';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { 
  Send, 
  Loader2, 
  Bot, 
  User,
  Sparkles,
  Zap,
  DollarSign,
  AlertCircle
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ProviderSelector } from './provider-selector';

interface ChatInterfaceV5Props {
  onMetricsUpdate?: (metrics: any) => void;
}

export function ChatInterfaceV5({ onMetricsUpdate }: ChatInterfaceV5Props) {
  const [provider, setProvider] = useState('openai');
  const [useGateway, setUseGateway] = useState(true); // Default to gateway mode
  const [input, setInput] = useState(''); // Manage input state ourselves
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  
  // Use the correct AI SDK v5 API with DefaultChatTransport
  const { messages, sendMessage, error, regenerate, status } = useChat({
    transport: new DefaultChatTransport({
      api: '/api/chat',
      // Prepare the request to include provider and gateway flag
      prepareSendMessagesRequest: ({ messages, trigger }) => {
        return {
          body: {
            messages,
            provider, // Include the current provider
            useGateway // Include gateway routing flag
          }
        };
      }
    }),
    onFinish: (message) => {
      if (onMetricsUpdate) {
        onMetricsUpdate({
          provider,
          timestamp: new Date().toISOString()
        });
      }
    }
  });

  const isLoading = status === 'in_progress';

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    }
  }, [messages]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (input.trim() && !isLoading) {
      // Send message using the v5 API format
      sendMessage({ 
        text: input
      });
      setInput('');
    }
  };

  const getProviderInfo = (provider: string) => {
    const providers = {
      openai: { 
        name: 'OpenAI', 
        model: 'GPT-4o Mini', 
        color: 'bg-green-500',
        icon: '🤖' 
      },
      anthropic: { 
        name: 'Anthropic', 
        model: 'Claude 3.5 Sonnet', 
        color: 'bg-purple-500',
        icon: '🧠' 
      },
      google: { 
        name: 'Google', 
        model: 'Gemini 1.5 Flash', 
        color: 'bg-blue-500',
        icon: '✨' 
      }
    };
    return providers[provider as keyof typeof providers] || providers.openai;
  };

  const providerInfo = getProviderInfo(provider);

  // Helper function to extract text from message parts
  const getMessageText = (message: any) => {
    if (!message.parts) return message.content || '';
    
    return message.parts
      .filter((part: any) => part.type === 'text')
      .map((part: any) => part.text || '')
      .join('');
  };

  return (
    <Card className="flex flex-col h-[700px] w-full">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-3">
          <div className={cn("w-10 h-10 rounded-lg flex items-center justify-center text-white", providerInfo.color)}>
            <Bot className="w-6 h-6" />
          </div>
          <div>
            <h2 className="text-lg font-semibold">Leash Gateway Chat</h2>
            <p className="text-sm text-muted-foreground">
              {providerInfo.name} ({providerInfo.model}) - {useGateway ? 'Via Gateway' : 'Direct'}
            </p>
          </div>
        </div>
        
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Mode:</span>
            <Button
              variant={useGateway ? "default" : "secondary"}
              size="sm"
              onClick={() => setUseGateway(!useGateway)}
              className={cn(
                "text-xs min-w-[100px]",
                useGateway ? "bg-green-600 hover:bg-green-700" : "bg-gray-600 hover:bg-gray-700"
              )}
            >
              {useGateway ? '🚦 Gateway' : '🔗 Direct'}
            </Button>
          </div>
          
          <ProviderSelector 
            value={provider} 
            onValueChange={setProvider}
            disabled={isLoading}
          />
        </div>
      </div>

      {/* Messages */}
      <ScrollArea ref={scrollAreaRef} className="flex-1 p-4">
        <div className="space-y-4">
          {messages.length === 0 && (
            <div className="text-center py-12">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 mb-4">
                <Sparkles className="w-8 h-8 text-primary" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Welcome to Leash Gateway Demo</h3>
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-muted mb-3">
                <div className={cn(
                  "w-2 h-2 rounded-full animate-pulse",
                  useGateway ? "bg-green-500" : "bg-blue-500"
                )} />
                <span className="text-xs font-medium">
                  {useGateway ? '🚦 Routing through Gateway' : '🔗 Direct to Provider'}
                </span>
              </div>
              <p className="text-muted-foreground max-w-md mx-auto">
                Experience secure, governed LLM interactions. 
                {useGateway ? ' Requests are being routed through the gateway.' : ' Currently in direct mode (bypassing gateway).'}
              </p>
            </div>
          )}
          
          {messages.map((message) => {
            const messageText = getMessageText(message);
            
            return (
              <div 
                key={message.id} 
                className={cn(
                  "flex gap-3",
                  message.role === 'user' ? "justify-end" : "justify-start"
                )}
              >
                {message.role === 'assistant' && (
                  <div className={cn("w-8 h-8 rounded-full flex items-center justify-center text-white shrink-0", providerInfo.color)}>
                    <Bot className="w-5 h-5" />
                  </div>
                )}
                
                <div className={cn(
                  "max-w-[70%] rounded-lg px-4 py-2",
                  message.role === 'user' 
                    ? "bg-primary text-primary-foreground" 
                    : "bg-muted"
                )}>
                  <div className="prose prose-sm dark:prose-invert max-w-none">
                    {messageText}
                  </div>
                </div>
                
                {message.role === 'user' && (
                  <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground shrink-0">
                    <User className="w-5 h-5" />
                  </div>
                )}
              </div>
            );
          })}
          
          {isLoading && (
            <div className="flex gap-3">
              <div className={cn("w-8 h-8 rounded-full flex items-center justify-center text-white shrink-0", providerInfo.color)}>
                <Bot className="w-5 h-5" />
              </div>
              <div className="bg-muted rounded-lg px-4 py-2">
                <Loader2 className="w-4 h-4 animate-spin" />
              </div>
            </div>
          )}
          
          {error && (
            <div className="flex items-center gap-2 p-3 bg-destructive/10 text-destructive rounded-lg">
              <AlertCircle className="w-5 h-5" />
              <div className="flex-1">
                <p className="font-medium">Error</p>
                <p className="text-sm">{error.message || 'Something went wrong'}</p>
              </div>
              {regenerate && (
                <Button 
                  variant="ghost" 
                  size="sm" 
                  onClick={() => regenerate()}
                  className="ml-auto"
                >
                  Retry
                </Button>
              )}
            </div>
          )}
        </div>
      </ScrollArea>

      <Separator />

      {/* Input */}
      <form onSubmit={handleSubmit} className="p-4">
        <div className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Type your message..."
            disabled={isLoading}
            className="flex-1 px-3 py-2 text-sm border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
          />
          <Button type="submit" disabled={isLoading || !input.trim()}>
            {isLoading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Send className="w-4 h-4" />
            )}
          </Button>
        </div>
        
        {/* Quick prompts */}
        <div className="flex flex-wrap gap-2 mt-3">
          {[
            "What is quantum computing?",
            "Write a haiku about AI",
            "Explain the gateway pattern"
          ].map((prompt) => (
            <Button
              key={prompt}
              variant="outline"
              size="sm"
              type="button"
              onClick={() => setInput(prompt)}
              disabled={isLoading}
              className="text-xs"
            >
              {prompt}
            </Button>
          ))}
        </div>
      </form>
    </Card>
  );
}
