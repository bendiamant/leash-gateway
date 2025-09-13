# AI SDK v5 Migration Guide

## Overview

This document captures the complete journey of migrating from AI SDK v4 patterns to AI SDK v5, including the challenges encountered and solutions implemented. This migration was critical for the demo application's success.

## Migration Context

**When**: During demo app refactor  
**Challenge**: AI SDK v5 introduced breaking changes that weren't immediately apparent  
**Solution**: Used Context7 MCP to access official documentation and fix all issues  
**Result**: Fully functional chat interface with streaming support

## Breaking Changes Discovered

### 1. useChat Hook API Changes

#### AI SDK v4 Pattern (No Longer Works)
```typescript
const { 
  messages, 
  input,           // ❌ No longer provided
  handleInputChange, // ❌ No longer provided
  handleSubmit,     // ❌ No longer provided
  setInput,        // ❌ No longer provided
  isLoading 
} = useChat();
```

#### AI SDK v5 Pattern (Current)
```typescript
const { 
  messages,
  sendMessage,     // ✅ New method for sending messages
  status,          // ✅ New status field ('idle' | 'in_progress')
  regenerate,      // ✅ Method to regenerate last response
  error,
  // Other fields: id, setMessages, clearError, stop, addToolResult
} = useChat({
  transport: new DefaultChatTransport({ api: '/api/chat' })
});

// Input must be managed manually
const [input, setInput] = useState('');
```

### 2. Message Format Changes

#### Old Format
```typescript
// Simple messages array
messages: [
  { role: 'user', content: 'Hello' },
  { role: 'assistant', content: 'Hi there!' }
]
```

#### New Format (UIMessage)
```typescript
// UIMessage format with parts
messages: [
  {
    id: 'msg-1',
    role: 'user',
    content: 'Hello', // Can also have parts array
    parts: [{ type: 'text', text: 'Hello' }]
  }
]
```

### 3. API Route Response Methods

#### Old Method (v4)
```typescript
// These no longer exist
return new StreamingTextResponse(stream);
return result.toDataStreamResponse(); // ❌ Wrong method
```

#### New Method (v5)
```typescript
import { streamText, convertToModelMessages, UIMessage } from 'ai';

const result = streamText({
  model: openai('gpt-4o'),
  messages: convertToModelMessages(uiMessages), // ✅ Use convertToModelMessages
  temperature: 0.7,
  maxTokens: 1000
});

return result.toUIMessageStreamResponse(); // ✅ Correct method for UI
// Alternative: result.toTextStreamResponse() for plain text
```

### 4. Transport Configuration

#### New DefaultChatTransport Usage
```typescript
const { messages, sendMessage } = useChat({
  transport: new DefaultChatTransport({
    api: '/api/chat',
    // Custom request preparation
    prepareSendMessagesRequest: ({ messages, trigger }) => {
      return {
        body: {
          messages,
          provider: selectedProvider // Custom data
        }
      };
    }
  })
});
```

## Implementation Strategy

### 1. Debug Approach

Created multiple implementations to isolate issues:

1. **SimpleChat.tsx**: Basic fetch-based implementation (worked as baseline)
2. **ChatInterfaceV5.tsx**: Proper v5 implementation with useChat
3. **TestUseChat.tsx**: Debug component to inspect hook returns

### 2. Key Discoveries Process

```typescript
// Debug component revealed actual API
export function TestUseChat() {
  const chatHelpers = useChat({ api: '/api/chat' });
  
  console.log('useChat returns:', {
    hasMessages: !!chatHelpers.messages,
    hasInput: !!chatHelpers.input, // false - not provided!
    hasHandleSubmit: !!chatHelpers.handleSubmit, // false - not provided!
    hasSendMessage: !!chatHelpers.sendMessage, // true - this is the new way
    allKeys: Object.keys(chatHelpers) // Showed actual available methods
  });
}
```

### 3. Context7 MCP Integration

Used Context7 MCP to get accurate documentation:

```typescript
// Used Context7 to resolve library and get docs
mcp_context7_resolve-library-id({ libraryName: "vercel ai sdk" })
mcp_context7_get-library-docs({ 
  context7CompatibleLibraryID: "/vercel/ai",
  topic: "useChat hook react streaming"
})
```

This provided the exact, up-to-date API documentation needed.

## Final Working Implementation

### Chat Component (ChatInterfaceV5.tsx)
```typescript
export function ChatInterfaceV5() {
  const [input, setInput] = useState(''); // Manage input manually
  const [provider, setProvider] = useState('openai');
  
  const { messages, sendMessage, status, error, regenerate } = useChat({
    transport: new DefaultChatTransport({
      api: '/api/chat',
      prepareSendMessagesRequest: ({ messages }) => ({
        body: { messages, provider }
      })
    })
  });

  const isLoading = status === 'in_progress';

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (input.trim() && !isLoading) {
      sendMessage({ text: input }); // Use sendMessage, not handleSubmit
      setInput('');
    }
  };

  // ... render UI
}
```

### API Route (api/chat/route.ts)
```typescript
import { streamText, convertToModelMessages, UIMessage } from 'ai';

export const maxDuration = 30; // Important for streaming

export async function POST(req: Request) {
  const { messages = [], provider = 'openai' } = await req.json();
  
  // Convert to UIMessage format if needed
  const uiMessages: UIMessage[] = messages.map((msg: any) => ({
    id: msg.id || crypto.randomUUID(),
    role: msg.role,
    content: msg.content || msg.text || ''
  }));

  const result = streamText({
    model: /* provider model */,
    messages: convertToModelMessages(uiMessages),
    temperature: 0.7,
    maxTokens: 1000
  });

  return result.toUIMessageStreamResponse(); // Correct response method
}
```

## Lessons Learned

### 1. Documentation is Critical
- Official docs may be outdated during major version changes
- Context7 MCP provided accurate, up-to-date information
- Always verify actual API by inspecting returned objects

### 2. Progressive Debugging
- Start with simple implementation (fetch-based)
- Add complexity gradually
- Use debug components to inspect actual APIs

### 3. Breaking Changes Management
- Major version upgrades require careful migration
- Keep fallback implementations during transition
- Test thoroughly with different scenarios

### 4. Tool Selection
- Context7 MCP was invaluable for getting correct documentation
- AI SDK v5 is more explicit but also more flexible
- DefaultChatTransport provides good customization options

## Common Pitfalls to Avoid

1. **Don't assume old patterns work** - v5 is fundamentally different
2. **Don't use toDataStreamResponse()** - use toUIMessageStreamResponse()
3. **Don't expect input management** - handle it yourself with useState
4. **Don't use handleSubmit** - use sendMessage instead
5. **Don't forget maxDuration** - needed for streaming responses
6. **Don't use convertToCoreMessages** - use convertToModelMessages

## References

- [AI SDK v5 Documentation](https://sdk.vercel.ai)
- [Context7 MCP Tool](https://context7.com)
- Demo Implementation: `/leash-demo-app/src/components/chat/chat-interface-v5.tsx`
- API Implementation: `/leash-demo-app/src/app/api/chat/route.ts`

## Status

✅ **COMPLETE** - All AI SDK v5 migration issues resolved and documented
