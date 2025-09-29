# RAGO JavaScript SDK

A comprehensive JavaScript/TypeScript SDK for interacting with RAGO - A powerful RAG (Retrieval-Augmented Generation) system with agent automation capabilities.

## Features

- 🔍 **RAG Operations**: Document ingestion, semantic search, and query processing
- 🤖 **LLM Integration**: Direct language model operations with streaming support
- 💬 **Conversations**: Manage chat conversations and history
- 🛠️ **MCP Tools**: Model Context Protocol integration for external tools
- 📊 **Analytics**: Comprehensive usage statistics and performance metrics
- 🤖 **Agents**: AI agent automation for complex workflows
- 🔧 **Tools**: Built-in tool management and execution
- 🚀 **TypeScript**: Full TypeScript support with comprehensive types
- 📡 **Streaming**: Real-time streaming for chat and generation

## Installation

```bash
npm install @rago/javascript-sdk
```

## Quick Start

```typescript
import { createRAGO } from '@rago/javascript-sdk';

// Initialize the SDK
const rago = createRAGO({
  baseURL: 'http://localhost:7127',
  // apiKey: 'your-api-key', // Optional
  timeout: 30000, // Optional
});

// Check server health
const isHealthy = await rago.healthCheck();
console.log('Server healthy:', isHealthy);

// Ingest a document
const ingestResult = await rago.rag.ingest({
  content: 'Your document content here...',
  filename: 'document.txt',
  metadata: { category: 'documentation' }
});

// Query the RAG system
const queryResult = await rago.rag.query({
  query: 'What is this document about?',
  top_k: 5,
  show_sources: true
});

console.log('Answer:', queryResult.answer);
console.log('Sources:', queryResult.sources);
```

## Core Modules

### RAG Module

Handle document ingestion, querying, and search operations:

```typescript
// Ingest documents
await rago.rag.ingest({
  content: 'Document content...',
  filename: 'doc.pdf',
  chunk_size: 1000,
  chunk_overlap: 200
});

// Query with RAG
const result = await rago.rag.query({
  query: 'What are the main topics?',
  top_k: 3,
  temperature: 0.7,
  show_sources: true
});

// Stream query responses
await rago.rag.queryStream({
  query: 'Explain the concept...',
  stream: true
}, {
  onData: (chunk) => console.log(chunk),
  onComplete: () => console.log('Done'),
  onError: (error) => console.error(error)
});

// Search without generation
const searchResults = await rago.rag.search({
  query: 'search term',
  top_k: 10
});

// List and manage documents
const documents = await rago.rag.listDocuments();
await rago.rag.deleteDocument('doc-id');
```

### LLM Module

Direct language model operations:

```typescript
// Generate text
const generated = await rago.llm.generate({
  prompt: 'Write a story about...',
  temperature: 0.8,
  max_tokens: 500
});

// Chat with the LLM
const chatResponse = await rago.llm.chat({
  messages: [
    { role: 'user', content: 'Hello!' },
    { role: 'assistant', content: 'Hi there!' },
    { role: 'user', content: 'How are you?' }
  ],
  temperature: 0.7
});

// Generate structured output
const structured = await rago.llm.generateStructured({
  prompt: 'Extract information from this text...',
  schema: {
    type: 'object',
    properties: {
      name: { type: 'string' },
      age: { type: 'number' },
      email: { type: 'string' }
    }
  }
});

// Streaming generation
await rago.llm.generateStream({
  prompt: 'Tell me about...'
}, {
  onData: (chunk) => process.stdout.write(chunk),
  onComplete: () => console.log('\nDone!')
});
```

### Conversations Module

Manage chat conversations:

```typescript
// Create a new conversation
const conversation = await rago.conversations.createNew();

// List all conversations
const conversations = await rago.conversations.list();

// Get a specific conversation
const conv = await rago.conversations.get('conversation-id');

// Save conversation updates
await rago.conversations.save({
  id: 'conversation-id',
  title: 'Updated Title',
  messages: [...messages]
});

// Search conversations
const searchResults = await rago.conversations.search('search query');
```

### MCP Module

Integrate with Model Context Protocol tools:

```typescript
// List available MCP tools
const tools = await rago.mcp.listTools();

// Call an MCP tool
const result = await rago.mcp.callTool('file_search', {
  pattern: '*.json',
  directory: '/path/to/search'
});

// Batch call multiple tools
const results = await rago.mcp.batchCallTools([
  { tool_name: 'tool1', arguments: { param: 'value1' } },
  { tool_name: 'tool2', arguments: { param: 'value2' } }
]);

// Chat with MCP integration
const mcpChat = await rago.mcp.chatWithMCP({
  messages: [{ role: 'user', content: 'Search for files...' }]
});

// Get server status
const servers = await rago.mcp.getServerStatus();
```

### Analytics Module

Access usage statistics and performance metrics:

```typescript
// Get overall usage stats
const usage = await rago.analytics.getUsageStats();
console.log('Total tokens:', usage.total_tokens);
console.log('Total cost:', usage.total_cost);

// Get RAG analytics
const ragAnalytics = await rago.analytics.getRAGAnalytics({
  start_time: '2023-01-01',
  end_time: '2023-12-31'
});

// Get performance metrics
const performance = await rago.analytics.getRAGPerformance();
console.log('Average latency:', performance.avg_total_latency);

// Get query records
const queries = await rago.analytics.getRAGQueries({
  limit: 100,
  success: true
});

// Get usage by provider/model
const byProvider = await rago.analytics.getUsageStatsByProvider();
const byModel = await rago.analytics.getUsageStatsByModel(10);

// Get daily usage trends
const dailyStats = await rago.analytics.getDailyUsageStats(30);
```

### Agents Module

AI agent automation:

```typescript
// Run an agent to complete a task
const agentResult = await rago.agents.run({
  task: 'Research the latest developments in AI and write a summary',
  context: { domain: 'artificial intelligence' },
  tools: ['web_search', 'file_write'],
  max_iterations: 10
});

console.log('Result:', agentResult.result);
console.log('Steps taken:', agentResult.total_steps);

// Create a plan without executing
const plan = await rago.agents.plan({
  task: 'Analyze the quarterly sales data',
  context: { quarter: 'Q3 2023' }
});

console.log('Planned steps:', plan.plan);
console.log('Estimated time:', plan.estimated_time);
```

### Tools Module

Built-in tool management:

```typescript
// List available tools
const tools = await rago.tools.list();

// Execute a tool
const result = await rago.tools.execute('calculator', {
  expression: '2 + 2 * 3'
});

// Get tool statistics
const stats = await rago.tools.getStats();
console.log('Total executions:', stats.total_executions);
console.log('Success rate:', stats.success_rate);

// List tool executions
const executions = await rago.tools.listExecutions();
```

## Configuration

```typescript
interface RAGOConfig {
  baseURL: string;           // RAGO server URL
  apiKey?: string;          // Optional API key
  timeout?: number;         // Request timeout (default: 30000ms)
  headers?: Record<string, string>; // Custom headers
}
```

## Error Handling

The SDK throws descriptive errors for different scenarios:

```typescript
try {
  const result = await rago.rag.query({
    query: 'What is this about?'
  });
} catch (error) {
  if (error.status === 404) {
    console.log('Resource not found');
  } else if (error.status >= 500) {
    console.log('Server error:', error.message);
  } else {
    console.log('Client error:', error.message);
  }
}
```

## Streaming Support

Many operations support real-time streaming:

```typescript
// RAG streaming
await rago.rag.queryStream(request, {
  onData: (chunk) => {
    // Handle streaming chunks
    console.log(chunk);
  },
  onComplete: () => {
    console.log('Stream completed');
  },
  onError: (error) => {
    console.error('Stream error:', error);
  }
});

// LLM streaming
await rago.llm.generateStream(request, streamOptions);
await rago.llm.chatStream(request, streamOptions);
```

## TypeScript Support

The SDK is written in TypeScript and provides comprehensive type definitions:

```typescript
import { 
  RAGO, 
  RAGOConfig, 
  QueryRequest, 
  QueryResponse,
  IngestRequest,
  ChatMessage 
} from '@rago/javascript-sdk';

// Full type safety
const config: RAGOConfig = {
  baseURL: 'http://localhost:7127'
};

const rago = new RAGO(config);

const request: QueryRequest = {
  query: 'What is machine learning?',
  top_k: 5,
  temperature: 0.7
};

const response: QueryResponse = await rago.rag.query(request);
```

## Examples

### Basic RAG Workflow

```typescript
import { createRAGO } from '@rago/javascript-sdk';

async function basicRAGWorkflow() {
  const rago = createRAGO({
    baseURL: 'http://localhost:7127'
  });

  // 1. Ingest documents
  await rago.rag.ingest({
    content: 'Machine learning is a subset of artificial intelligence...',
    filename: 'ml-guide.txt'
  });

  // 2. Query the system
  const result = await rago.rag.query({
    query: 'What is machine learning?',
    show_sources: true
  });

  console.log('Answer:', result.answer);
  
  // 3. Get analytics
  const analytics = await rago.analytics.getRAGAnalytics();
  console.log('Total queries:', analytics.total_queries);
}
```

### Agent Workflow

```typescript
async function agentWorkflow() {
  const rago = createRAGO({
    baseURL: 'http://localhost:7127'
  });

  // Run an autonomous agent
  const result = await rago.agents.run({
    task: 'Research the top 5 JavaScript frameworks and create a comparison',
    tools: ['web_search', 'file_write'],
    max_iterations: 15
  });

  console.log('Agent completed task:');
  console.log(result.result);
  
  // Review the steps taken
  result.iterations.forEach((step, i) => {
    console.log(`Step ${i + 1}: ${step.action}`);
    if (step.tool) {
      console.log(`  Used tool: ${step.tool}`);
    }
  });
}
```

## Contributing

1. Clone the repository
2. Install dependencies: `npm install`
3. Build: `npm run build`
4. Test: `npm test`
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: [RAGO Docs](https://github.com/liliang-cn/rago)
- Issues: [GitHub Issues](https://github.com/liliang-cn/rago/issues)
- Discussions: [GitHub Discussions](https://github.com/liliang-cn/rago/discussions)