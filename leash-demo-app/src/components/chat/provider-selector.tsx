'use client';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface Provider {
  id: string;
  name: string;
  model: string;
  description: string;
  color: string;
  icon: string;
  costPer1k: number;
  latency: string;
}

const providers: Provider[] = [
  {
    id: 'openai',
    name: 'OpenAI',
    model: 'GPT-4o Mini',
    description: 'Fast and efficient for most tasks',
    color: 'bg-green-500',
    icon: '🤖',
    costPer1k: 0.00015,
    latency: '~200ms'
  },
  {
    id: 'anthropic',
    name: 'Anthropic',
    model: 'Claude 3.5 Sonnet',
    description: 'Advanced reasoning and analysis',
    color: 'bg-purple-500',
    icon: '🧠',
    costPer1k: 0.003,
    latency: '~300ms'
  },
  {
    id: 'google',
    name: 'Google',
    model: 'Gemini 1.5 Flash',
    description: 'Multimodal and context-aware',
    color: 'bg-blue-500',
    icon: '✨',
    costPer1k: 0.00035,
    latency: '~250ms'
  }
];

interface ProviderSelectorProps {
  value: string;
  onValueChange: (value: string) => void;
  disabled?: boolean;
}

export function ProviderSelector({ value, onValueChange, disabled }: ProviderSelectorProps) {
  const selectedProvider = providers.find(p => p.id === value) || providers[0];

  return (
    <Select value={value} onValueChange={onValueChange} disabled={disabled}>
      <SelectTrigger className="w-[280px]">
        <SelectValue>
          <div className="flex items-center gap-2">
            <div className={cn("w-2 h-2 rounded-full", selectedProvider.color)} />
            <span className="font-medium">{selectedProvider.name}</span>
            <Badge variant="secondary" className="text-xs">
              {selectedProvider.model}
            </Badge>
          </div>
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {providers.map((provider) => (
          <SelectItem key={provider.id} value={provider.id}>
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <div className={cn("w-2 h-2 rounded-full", provider.color)} />
                <span className="font-medium">{provider.name}</span>
                <Badge variant="outline" className="text-xs">
                  {provider.model}
                </Badge>
              </div>
              <div className="text-xs text-muted-foreground">
                {provider.description}
              </div>
              <div className="flex items-center gap-3 text-xs text-muted-foreground">
                <span>💰 ${provider.costPer1k}/1k tokens</span>
                <span>⚡ {provider.latency}</span>
              </div>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
