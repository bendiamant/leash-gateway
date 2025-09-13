'use client';

import { useState } from 'react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ChatInterfaceV5 } from '@/components/chat/chat-interface-v5';
import { SimpleChat } from '@/components/chat/simple-chat';
import { MetricsDashboard } from '@/components/metrics/metrics-dashboard';
import { 
  Shield, 
  MessageSquare, 
  BarChart3, 
  Zap,
  Lock,
  Activity,
  Github,
  ExternalLink,
  Sparkles,
  ChevronRight
} from 'lucide-react';
import { cn } from '@/lib/utils';

export default function Home() {
  const [activeTab, setActiveTab] = useState('chat');
  const [useSimpleChat, setUseSimpleChat] = useState(false); // Use the fixed v5 chat by default

  const features = [
    {
      icon: <Shield className="h-5 w-5" />,
      title: "Security & Governance",
      description: "Centralized policy enforcement, rate limiting, and content filtering for all LLM traffic"
    },
    {
      icon: <Zap className="h-5 w-5" />,
      title: "Multi-Provider Support",
      description: "Seamlessly switch between OpenAI, Anthropic, and Google with no code changes"
    },
    {
      icon: <Activity className="h-5 w-5" />,
      title: "Real-time Monitoring",
      description: "Complete observability with metrics, health checks, and cost tracking"
    },
    {
      icon: <Lock className="h-5 w-5" />,
      title: "Enterprise Ready",
      description: "Production-grade infrastructure with <4ms P50 latency overhead"
    }
  ];

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted/20">
      {/* Header */}
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-primary flex items-center justify-center">
                <Shield className="h-6 w-6 text-primary-foreground" />
              </div>
              <div>
                <h1 className="text-xl font-bold">Leash Security Gateway</h1>
                <p className="text-sm text-muted-foreground">Enterprise LLM Security & Governance</p>
              </div>
            </div>
            
            <div className="flex items-center gap-4">
              <Badge variant="outline" className="gap-1">
                <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                Gateway Online
              </Badge>
              <Button variant="outline" size="sm" asChild>
                <a href="https://github.com/leash-security/gateway" target="_blank" rel="noopener noreferrer">
                  <Github className="h-4 w-4 mr-2" />
                  GitHub
                  <ExternalLink className="h-3 w-3 ml-1" />
                </a>
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="container mx-auto px-4 py-8">
        <Card className="border-primary/20 bg-gradient-to-r from-primary/5 to-primary/10">
          <CardContent className="p-8">
            <div className="flex items-start justify-between">
              <div className="space-y-4 max-w-2xl">
                <div className="flex items-center gap-2">
                  <Sparkles className="h-5 w-5 text-primary" />
                  <Badge>Live Demo</Badge>
                </div>
                <h2 className="text-3xl font-bold">
                  Secure Your AI Infrastructure with Zero Code Changes
                </h2>
                <p className="text-lg text-muted-foreground">
                  Experience the power of centralized LLM governance. Just change your base URL and get 
                  instant security, observability, and cost control for all your AI applications.
                </p>
                <div className="flex flex-wrap gap-3">
                  {features.map((feature, index) => (
                    <Badge key={index} variant="secondary" className="gap-1 py-1">
                      {feature.icon}
                      {feature.title}
                    </Badge>
                  ))}
                </div>
              </div>
              <div className="hidden lg:block">
                <Card className="p-4 bg-muted/50">
                  <pre className="text-xs">
                    <code>{`// Before: Direct provider calls
const client = new OpenAI({
  baseURL: "https://api.openai.com/v1"
});

// After: Through Leash Gateway
const client = new OpenAI({
  baseURL: "http://gateway.company.com/v1/openai"
});

// ✨ That's it! Full security & observability`}</code>
                  </pre>
                </Card>
              </div>
            </div>
          </CardContent>
        </Card>
      </section>

      {/* Main Content */}
      <section className="container mx-auto px-4 pb-8">
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
          <TabsList className="grid w-full max-w-md mx-auto grid-cols-3">
            <TabsTrigger value="chat" className="gap-2">
              <MessageSquare className="h-4 w-4" />
              Chat Demo
            </TabsTrigger>
            <TabsTrigger value="metrics" className="gap-2">
              <BarChart3 className="h-4 w-4" />
              Metrics
            </TabsTrigger>
            <TabsTrigger value="features" className="gap-2">
              <Shield className="h-4 w-4" />
              Features
            </TabsTrigger>
          </TabsList>

          <TabsContent value="chat" className="space-y-6">
            <div className="grid lg:grid-cols-3 gap-6">
              <div className="lg:col-span-2">
                {useSimpleChat ? (
                  <div>
                    <div className="mb-2 p-2 bg-yellow-100 rounded text-sm">
                      Debug Mode: Using SimpleChat (basic implementation)
                      <Button 
                        size="sm" 
                        variant="outline" 
                        className="ml-2"
                        onClick={() => setUseSimpleChat(false)}
                      >
                        Switch to Full Chat
                      </Button>
                    </div>
                    <SimpleChat />
                  </div>
                ) : (
                  <div>
                    <div className="mb-2 p-2 bg-green-100 rounded text-sm">
                      Using AI SDK v5 Chat with proper API implementation
                      <Button 
                        size="sm" 
                        variant="outline" 
                        className="ml-2"
                        onClick={() => setUseSimpleChat(true)}
                      >
                        Switch to Simple Chat
                      </Button>
                    </div>
                    <ChatInterfaceV5 />
                  </div>
                )}
              </div>
              <div className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Gateway Features</CardTitle>
                    <CardDescription>Active security modules</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {[
                      { name: 'Rate Limiter', status: 'active', description: '100 req/min per API key' },
                      { name: 'Content Filter', status: 'active', description: 'PII & sensitive data protection' },
                      { name: 'Cost Tracker', status: 'active', description: 'Real-time usage monitoring' },
                      { name: 'Request Logger', status: 'active', description: 'Full audit trail' }
                    ].map((module) => (
                      <div key={module.name} className="flex items-start gap-3">
                        <div className="w-2 h-2 rounded-full bg-green-500 mt-2" />
                        <div className="flex-1">
                          <div className="font-medium text-sm">{module.name}</div>
                          <div className="text-xs text-muted-foreground">{module.description}</div>
                        </div>
                      </div>
                    ))}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-lg">Try These Scenarios</CardTitle>
                    <CardDescription>See the gateway in action</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <Button variant="outline" size="sm" className="w-full justify-start">
                      <ChevronRight className="h-4 w-4 mr-2" />
                      Test rate limiting
                    </Button>
                    <Button variant="outline" size="sm" className="w-full justify-start">
                      <ChevronRight className="h-4 w-4 mr-2" />
                      Try PII filtering
                    </Button>
                    <Button variant="outline" size="sm" className="w-full justify-start">
                      <ChevronRight className="h-4 w-4 mr-2" />
                      Compare providers
                    </Button>
                  </CardContent>
                </Card>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="metrics">
            <MetricsDashboard />
          </TabsContent>

          <TabsContent value="features" className="space-y-6">
            <div className="grid md:grid-cols-2 gap-6">
              {features.map((feature, index) => (
                <Card key={index}>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      {feature.icon}
                      {feature.title}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-muted-foreground">{feature.description}</p>
                  </CardContent>
                </Card>
              ))}
            </div>

            <Card>
              <CardHeader>
                <CardTitle>Integration Example</CardTitle>
                <CardDescription>
                  Minimal changes required - just update your base URL
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid md:grid-cols-2 gap-6">
                  <div>
                    <h4 className="font-medium mb-2">OpenAI SDK</h4>
                    <pre className="text-sm bg-muted p-3 rounded-lg">
                      <code>{`const openai = new OpenAI({
  apiKey: process.env.OPENAI_API_KEY,
  baseURL: "http://gateway/v1/openai"
});`}</code>
                    </pre>
                  </div>
                  <div>
                    <h4 className="font-medium mb-2">LangChain</h4>
                    <pre className="text-sm bg-muted p-3 rounded-lg">
                      <code>{`const model = new ChatOpenAI({
  openAIApiKey: apiKey,
  configuration: {
    baseURL: "http://gateway/v1/openai"
  }
});`}</code>
                    </pre>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </section>

      {/* Footer */}
      <footer className="border-t mt-12">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Shield className="h-4 w-4" />
              <span>Leash Security Gateway - Apache 2.0 License</span>
            </div>
            <div className="flex items-center gap-4">
              <a href="https://github.com/leash-security/gateway" className="text-sm text-muted-foreground hover:text-foreground">
                Documentation
              </a>
              <a href="https://github.com/leash-security/gateway/issues" className="text-sm text-muted-foreground hover:text-foreground">
                Support
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}