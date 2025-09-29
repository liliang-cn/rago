// Core RAGO SDK Types and Interfaces

export interface RAGOConfig {
  baseURL: string;
  apiKey?: string;
  timeout?: number;
  headers?: Record<string, string>;
}

export interface APIResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// === RAG Types ===

export interface IngestRequest {
  content: string;
  filename?: string;
  metadata?: Record<string, any>;
  chunk_size?: number;
  chunk_overlap?: number;
}

export interface IngestResponse {
  document_id: string;
  chunks_created: number;
  total_tokens: number;
  processing_time: number;
}

export interface QueryRequest {
  query: string;
  top_k?: number;
  temperature?: number;
  max_tokens?: number;
  show_sources?: boolean;
  show_thinking?: boolean;
  tools_enabled?: boolean;
}

export interface QueryResponse {
  answer: string;
  sources: Chunk[];
  metadata?: {
    total_tokens?: number;
    processing_time?: number;
    model?: string;
  };
}

export interface Chunk {
  id: string;
  content: string;
  score: number;
  document_id: string;
  metadata?: Record<string, any>;
}

export interface Document {
  id: string;
  filename: string;
  content: string;
  chunk_count: number;
  created_at: string;
  metadata?: Record<string, any>;
}

// === LLM Types ===

export interface GenerateRequest {
  prompt: string;
  temperature?: number;
  max_tokens?: number;
  stream?: boolean;
}

export interface GenerateResponse {
  content: string;
  usage?: TokenUsage;
  model?: string;
}

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface ChatRequest {
  messages: ChatMessage[];
  temperature?: number;
  max_tokens?: number;
  stream?: boolean;
}

export interface ChatResponse {
  messages: ChatMessage[];
  response: string;
  usage?: TokenUsage;
  model?: string;
}

export interface StructuredGenerateRequest {
  prompt: string;
  schema: Record<string, any>;
  temperature?: number;
  max_tokens?: number;
}

export interface StructuredGenerateResponse {
  data: any;
  raw: string;
  valid: boolean;
  validation_errors?: string[];
}

export interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

// === Conversation Types ===

export interface Conversation {
  id: string;
  title: string;
  messages: ChatMessage[];
  created_at: string;
  updated_at: string;
  metadata?: Record<string, any>;
}

export interface ConversationSummary {
  id: string;
  title: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

// === MCP Types ===

export interface MCPTool {
  name: string;
  description: string;
  server_name: string;
  input_schema?: Record<string, any>;
}

export interface MCPServer {
  name: string;
  status: boolean;
  tools_count?: number;
}

export interface MCPToolCall {
  tool_name: string;
  arguments: Record<string, any>;
}

export interface MCPToolResult {
  success: boolean;
  result?: any;
  error?: string;
  execution_time?: number;
}

// === Analytics Types ===

export interface UsageStats {
  total_calls: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_tokens: number;
  total_cost: number;
  average_latency: number;
  success_rate: number;
}

export interface RAGAnalytics {
  total_queries: number;
  success_rate: number;
  avg_latency: number;
  avg_chunks: number;
  avg_score: number;
  fast_queries: number;
  medium_queries: number;
  slow_queries: number;
  high_quality_queries: number;
  medium_quality_queries: number;
  low_quality_queries: number;
  start_time: string;
  end_time: string;
}

export interface RAGQueryRecord {
  id: string;
  conversation_id: string;
  message_id: string;
  query: string;
  answer: string;
  top_k: number;
  temperature: number;
  max_tokens: number;
  show_sources: boolean;
  show_thinking: boolean;
  tools_enabled: boolean;
  total_latency: number;
  retrieval_time: number;
  generation_time: number;
  chunks_found: number;
  tool_calls_count: number;
  success: boolean;
  error_message?: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  estimated_cost: number;
  model: string;
  created_at: string;
}

export interface RAGPerformanceMetrics {
  avg_total_latency: number;
  avg_retrieval_time: number;
  avg_generation_time: number;
  p50_latency: number;
  p95_latency: number;
  p99_latency: number;
  success_rate: number;
  error_rate: number;
}

// === Agent Types ===

export interface AgentRunRequest {
  task: string;
  context?: Record<string, any>;
  tools?: string[];
  max_iterations?: number;
}

export interface AgentRunResponse {
  result: string;
  iterations: AgentIteration[];
  total_steps: number;
  success: boolean;
  error?: string;
}

export interface AgentIteration {
  step: number;
  action: string;
  tool?: string;
  result?: string;
  thinking?: string;
}

export interface AgentPlanRequest {
  task: string;
  context?: Record<string, any>;
}

export interface AgentPlanResponse {
  plan: AgentStep[];
  estimated_time: number;
  required_tools: string[];
}

export interface AgentStep {
  step_number: number;
  description: string;
  tool?: string;
  expected_output: string;
}

// === Tool Types ===

export interface Tool {
  name: string;
  description: string;
  parameters: Record<string, any>;
  enabled: boolean;
}

export interface ToolExecution {
  id: string;
  tool_name: string;
  arguments: Record<string, any>;
  result?: any;
  success: boolean;
  error?: string;
  execution_time: number;
  created_at: string;
}

// === Search Types ===

export interface SearchRequest {
  query: string;
  top_k?: number;
  threshold?: number;
  filters?: Record<string, any>;
}

export interface SearchResponse {
  results: Chunk[];
  total_found: number;
  processing_time: number;
}

export interface HybridSearchRequest extends SearchRequest {
  semantic_weight?: number;
  keyword_weight?: number;
}

// === Streaming Types ===

export interface StreamCallback<T> {
  (data: T): void;
}

export interface StreamOptions {
  onData?: StreamCallback<string>;
  onComplete?: () => void;
  onError?: (error: Error) => void;
}

// === Filter Options ===

export interface FilterOptions {
  start_time?: string;
  end_time?: string;
  limit?: number;
  offset?: number;
  provider?: string;
  model?: string;
  success?: boolean;
}