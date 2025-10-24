/**
 * RAGO JavaScript SDK
 * 
 * A comprehensive SDK for interacting with RAGO - 
 * A RAG system with agent automation capabilities
 */

// Main exports
export { RAGO, createRAGO } from './rago';
export { RAGOClient } from './client';

// Module exports
export { RAGModule } from './modules/rag';
export { LLMModule } from './modules/llm';
export { ConversationsModule } from './modules/conversations';
export { MCPModule } from './modules/mcp';
export { AnalyticsModule } from './modules/analytics';
export { ToolsModule } from './modules/tools';

// Type exports
export * from './types';

// Default export
export { createRAGO as default } from './rago';