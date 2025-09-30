import { useState } from 'react'

export interface Document {
  id: string
  title?: string
  content: string
  created: string
  metadata?: Record<string, any>
}

export interface DocumentInfo extends Document {
  path?: string
  source?: string
  summary?: string
  keywords?: string[]
}

export interface SearchResult extends Document {
  score: number
}

export interface IngestRequest {
  content: string
  title?: string
  metadata?: Record<string, any>
}

export interface QueryRequest {
  query: string
  context_only?: boolean
  show_thinking?: boolean
  filters?: Record<string, any>
}

export interface APIResponse<T> {
  data?: T
  error?: string
  message?: string
  success?: boolean
}

export interface ListAPIResponse<T> {
  success: boolean
  data: T[]
  total: number
  error?: string
}

export interface PagedAPIResponse<T> {
  success: boolean
  data: T[]
  page: number
  per_page: number
  total: number
  has_more: boolean
  error?: string
}

export interface MCPTool {
  name: string
  description: string
  server_name: string
}

export interface MCPServer {
  name: string
  status: boolean
}

export interface MCPToolResult {
  success: boolean
  data: any
  error?: string
  server_name: string
  tool_name: string
  duration: number
}

export interface MCPToolCall {
  tool_name: string
  args: Record<string, any>
}

export interface TaskItem {
  content: string
  status: 'pending' | 'in_progress' | 'completed'
  activeForm: string
}

export interface SearchRequest {
  query: string
  top_k?: number
  score_threshold?: number
  hybrid_search?: boolean
  vector_weight?: number
  filters?: Record<string, any>
  include_content?: boolean
}

export interface LLMGenerateRequest {
  prompt: string
  temperature?: number
  max_tokens?: number
  stream?: boolean
  system_prompt?: string
}

export interface LLMChatRequest {
  messages: Array<{role: string; content: string}>
  temperature?: number
  max_tokens?: number
  stream?: boolean
  system_prompt?: string
}

export interface MCPChatRequest {
  message: string
  conversation_id?: string
  options?: {
    temperature?: number
    max_tokens?: number
    show_thinking?: boolean
    allowed_tools?: string[]
    max_tool_calls?: number
  }
  context?: Record<string, any>
}

// Usage tracking interfaces
export interface UsageStats {
  total_calls: number
  total_input_tokens: number
  total_output_tokens: number
  total_tokens: number
  total_cost: number
  average_latency: number
  success_rate: number
}

export interface UsageStatsByType {
  [key: string]: UsageStats
}

export interface UsageStatsByProvider {
  [key: string]: UsageStats
}

export interface Conversation {
  id: string
  title: string
  created_at: number | string
  updated_at: number | string
  message_count?: number
  messages?: ConversationMessage[]
  metadata?: Record<string, any>
}

export interface Message {
  id: string
  conversation_id: string
  content: string
  type: 'user' | 'assistant'
  created_at: string
}

export interface ConversationHistory {
  conversation: Conversation
  messages: Message[]
  token_usage: UsageStats
}

export interface UsageFilter {
  conversation_id?: string
  call_type?: string
  provider?: string
  model?: string
  start_time?: string
  end_time?: string
  limit?: number
  offset?: number
}

// RAG visualization interfaces
export interface RAGQueryRecord {
  id: string
  conversation_id: string
  message_id: string
  query: string
  answer: string
  top_k: number
  temperature: number
  max_tokens: number
  show_sources: boolean
  show_thinking: boolean
  tools_enabled: boolean
  total_latency: number
  retrieval_time: number
  generation_time: number
  chunks_found: number
  tool_calls_count: number
  success: boolean
  error_message: string
  created_at: string
}

export interface RAGChunkHit {
  id: string
  rag_query_id: string
  chunk_id: string
  document_id: string
  content: string
  score: number
  rank: number
  used_in_generation: boolean
  source_file: string
  chunk_index: number
  char_start: number
  char_end: number
  created_at: string
}

export interface RAGToolCall {
  id: string
  rag_query_id: string
  tool_name: string
  arguments: string
  result: string
  success: boolean
  error_message: string
  duration: number
  created_at: string
}

// Enhanced tool call interfaces for comprehensive tracking
export interface ToolCallRecord {
  id: string
  uuid: string
  session_id?: string
  conversation_id?: string
  rag_query_id?: string
  tool_name: string
  tool_type: 'mcp' | 'rag' | 'llm' | 'system'
  server_name?: string
  arguments: Record<string, any>
  result: any
  success: boolean
  error_message?: string
  error_code?: string
  duration_ms: number
  start_time: string
  end_time: string
  created_at: string
  metadata?: Record<string, any>
}

export interface ToolCallFilter {
  start_time?: string
  end_time?: string
  tool_name?: string
  tool_type?: string
  server_name?: string
  success?: boolean
  session_id?: string
  conversation_id?: string
  rag_query_id?: string
  limit?: number
  offset?: number
}

export interface ToolCallStats {
  total_calls: number
  successful_calls: number
  failed_calls: number
  success_rate: number
  avg_duration_ms: number
  total_duration_ms: number
  unique_tools: number
  unique_servers: number
  calls_by_tool: Record<string, number>
  calls_by_server: Record<string, number>
  calls_by_type: Record<string, number>
  error_rate_by_tool: Record<string, number>
  avg_duration_by_tool: Record<string, number>
}

export interface ToolCallAnalytics {
  total_stats: ToolCallStats
  daily_stats: Record<string, ToolCallStats>
  hourly_distribution: Record<string, number>
  error_analysis: {
    top_errors: Array<{
      error_code: string
      error_message: string
      count: number
      tools: string[]
    }>
    error_trend: Record<string, number>
  }
  performance_metrics: {
    slowest_tools: Array<{
      tool_name: string
      avg_duration_ms: number
      call_count: number
    }>
    fastest_tools: Array<{
      tool_name: string
      avg_duration_ms: number
      call_count: number
    }>
    timeout_analysis: {
      timeout_count: number
      timeout_rate: number
      tools_with_timeouts: string[]
    }
  }
  usage_patterns: {
    peak_hours: number[]
    peak_days: string[]
    tool_combinations: Array<{
      tools: string[]
      frequency: number
    }>
  }
}

export interface ToolCallVisualization {
  record: ToolCallRecord
  related_calls: ToolCallRecord[]
  timeline: Array<{
    timestamp: string
    event_type: 'start' | 'end' | 'error'
    message: string
  }>
  performance_context: {
    percentile_rank: number
    compared_to_tool_avg: number
    compared_to_server_avg: number
  }
}

export interface RAGRetrievalMetrics {
  average_score: number
  top_score: number
  score_distribution: Array<{min: number, max: number, count: number}>
  diversity_score: number
  coverage_score: number
}

export interface RAGQualityMetrics {
  answer_length: number
  source_utilization: number
  confidence_score: number
  hallucination_risk: number
  factuality_score: number
}

export interface RAGQueryVisualization {
  query: RAGQueryRecord
  chunk_hits: RAGChunkHit[]
  tool_calls: RAGToolCall[]
  retrieval_metrics: RAGRetrievalMetrics
  quality_metrics: RAGQualityMetrics
}

export interface RAGAnalytics {
  total_queries: number
  success_rate: number
  avg_latency: number
  avg_chunks: number
  avg_score: number
  fast_queries: number
  medium_queries: number
  slow_queries: number
  high_quality_queries: number
  medium_quality_queries: number
  low_quality_queries: number
  start_time: string
  end_time: string
}

export interface RAGSearchFilter {
  conversation_id?: string
  query?: string
  start_time?: string
  end_time?: string
  min_score?: number
  max_score?: number
  tools_used?: string[]
  limit?: number
  offset?: number
}

// Conversation Types
export interface ConversationMessage {
  role: string
  content: string
  sources?: any[]
  thinking?: string
  timestamp: number
}

export interface ConversationSummary {
  id: string
  title: string
  message_count: number
  created_at: number
  updated_at: number
}

export interface ConversationListResponse {
  conversations: ConversationSummary[]
  total: number
  page: number
  page_size: number
}

export interface RAGPerformanceReport {
  start_time: string
  end_time: string
  total_queries: number
  success_rate: number
  avg_latency: number
  median_latency: number
  p95_latency: number
  p99_latency: number
  avg_retrieval_time: number
  avg_generation_time: number
  retrieval_ratio: number
  avg_chunks_found: number
  avg_top_score: number
  avg_source_util: number
  avg_confidence: number
  avg_factuality: number
  latency_trend: 'improving' | 'declining' | 'stable'
  quality_trend: 'improving' | 'declining' | 'stable'
  slow_queries: Array<{
    query_id: string
    query: string
    latency: number
    issue: string
    severity: string
    timestamp: string
  }>
  low_quality_queries: Array<{
    query_id: string
    query: string
    issue: string
    severity: string
    timestamp: string
  }>
  recommendations: Array<{
    type: string
    title: string
    description: string
    impact: string
    priority: string
    metrics: Record<string, any>
  }>
}

class APIClient {
  private baseURL = '/api'

  async request<T>(endpoint: string, options: RequestInit = {}): Promise<APIResponse<T>> {
    try {
      const response = await fetch(`${this.baseURL}${endpoint}`, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
        ...options,
      })

      const data = await response.json()

      if (!response.ok) {
        return { error: data.error || `HTTP ${response.status}` }
      }

      // If the response has success/data structure, extract the inner data
      if (data && typeof data === 'object' && 'success' in data && 'data' in data) {
        return { 
          data: data.data,
          success: data.success 
        }
      }
      
      return { data }
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Unknown error' }
    }
  }

  async ingestText(content: string, title?: string, metadata?: Record<string, any>) {
    return this.request('/ingest', {
      method: 'POST',
      body: JSON.stringify({ content, title, metadata }),
    })
  }

  async ingestFile(file: File, metadata?: Record<string, any>) {
    const formData = new FormData()
    formData.append('file', file)
    if (metadata) {
      formData.append('metadata', JSON.stringify(metadata))
    }
    
    return fetch(`${this.baseURL}/ingest`, {
      method: 'POST',
      body: formData,
    }).then(async (response) => {
      const data = await response.json()
      if (!response.ok) {
        return { error: data.error || `HTTP ${response.status}` }
      }
      return { data }
    }).catch((error) => {
      return { error: error instanceof Error ? error.message : 'Unknown error' }
    })
  }

  async query(question: string, context_only = false, filters?: Record<string, any>, show_thinking = true): Promise<APIResponse<{ answer: string; sources: SearchResult[] }>> {
    return this.request('/query', {
      method: 'POST',
      body: JSON.stringify({ query: question, context_only, filters, show_thinking }),
    })
  }

  async search(query: string): Promise<APIResponse<SearchResult[]>> {
    return this.request('/search', {
      method: 'POST',
      body: JSON.stringify({ query: query }),
    })
  }

  async semanticSearch(request: SearchRequest): Promise<APIResponse<{results: SearchResult[], count: number, query: string}>> {
    return this.request('/rag/search/semantic', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async hybridSearch(request: SearchRequest): Promise<APIResponse<{results: SearchResult[], count: number, query: string}>> {
    return this.request('/rag/search/hybrid', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async filteredSearch(request: SearchRequest): Promise<APIResponse<{results: SearchResult[], count: number, query: string, filters: Record<string, any>}>> {
    return this.request('/rag/search/filtered', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async getDocuments(): Promise<ListAPIResponse<Document>> {
    const response = await fetch(`${this.baseURL}/documents`)
    const data = await response.json()
    
    if (!response.ok) {
      return { success: false, data: [], total: 0, error: data.error || `HTTP ${response.status}` }
    }
    
    // 确保 data 字段总是数组
    if (!Array.isArray(data.data)) {
      return { success: true, data: [], total: 0 }
    }
    
    return data as ListAPIResponse<Document>
  }

  async getDocumentsWithInfo(): Promise<APIResponse<DocumentInfo[]>> {
    return this.request('/rag/documents/info')
  }

  async getDocumentInfo(id: string): Promise<APIResponse<DocumentInfo>> {
    return this.request(`/rag/documents/${id}`)
  }

  async deleteDocument(id: string) {
    return this.request(`/documents/${id}`, {
      method: 'DELETE',
    })
  }

  async reset() {
    return this.request('/reset', {
      method: 'POST',
    })
  }

  async health() {
    return this.request('/health')
  }

  // LLM API methods
  async llmGenerate(request: LLMGenerateRequest): Promise<APIResponse<{content: string}>> {
    return this.request('/llm/generate', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async llmChat(request: LLMChatRequest): Promise<APIResponse<{response: string, messages: Array<{role: string; content: string}>}>> {
    return this.request('/llm/chat', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async llmStructured(prompt: string, schema: Record<string, any>, temperature = 0.3, max_tokens = 500): Promise<APIResponse<{data: any, valid: boolean, raw: string}>> {
    return this.request('/llm/structured', {
      method: 'POST',
      body: JSON.stringify({ prompt, schema, temperature, max_tokens }),
    })
  }

  async llmGenerateStream(request: LLMGenerateRequest, onChunk: (chunk: string) => void): Promise<void> {
    const response = await fetch(`${this.baseURL}/llm/generate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ ...request, stream: true }),
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }

    const reader = response.body?.getReader()
    const decoder = new TextDecoder()

    if (reader) {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const chunk = decoder.decode(value, { stream: true })
        const lines = chunk.split('\n')
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6)
            if (data !== '[DONE]' && data !== '') {
              onChunk(data)
            }
          }
        }
      }
    }
  }

  // MCP API methods
  async getMCPTools(): Promise<APIResponse<{ tools: MCPTool[]; count: number }>> {
    return this.request('/mcp/tools')
  }

  async getMCPTool(name: string): Promise<APIResponse<MCPTool & { schema: any }>> {
    return this.request(`/mcp/tools/${name}`)
  }

  async callMCPTool(toolName: string, args: Record<string, any>, timeout = 30): Promise<APIResponse<MCPToolResult>> {
    return this.request('/mcp/tools/call', {
      method: 'POST',
      body: JSON.stringify({ tool_name: toolName, args, timeout }),
    })
  }

  async batchCallMCPTools(calls: MCPToolCall[], timeout = 60): Promise<APIResponse<{ results: MCPToolResult[]; count: number }>> {
    return this.request('/mcp/tools/batch', {
      method: 'POST',
      body: JSON.stringify({ calls, timeout }),
    })
  }

  async getMCPServers(): Promise<APIResponse<{ servers: Record<string, boolean>; enabled: boolean }>> {
    return this.request('/mcp/servers')
  }

  async startMCPServer(serverName: string): Promise<APIResponse<any>> {
    return this.request('/mcp/servers/start', {
      method: 'POST',
      body: JSON.stringify({ server_name: serverName }),
    })
  }

  async stopMCPServer(serverName: string): Promise<APIResponse<any>> {
    return this.request('/mcp/servers/stop', {
      method: 'POST',
      body: JSON.stringify({ server_name: serverName }),
    })
  }

  async getMCPToolsForLLM(): Promise<APIResponse<{ tools: any[]; count: number }>> {
    return this.request('/mcp/llm/tools')
  }

  async getMCPToolsByServer(serverName: string): Promise<APIResponse<{ server: string; tools: MCPTool[]; count: number }>> {
    return this.request(`/mcp/servers/${serverName}/tools`)
  }

  async mcpChat(request: MCPChatRequest): Promise<APIResponse<{
    content: string
    final_response?: string
    conversation_id: string
    tool_calls?: Array<{
      tool_name: string
      args: Record<string, any>
      result: any
      success: boolean
      error?: string
      duration?: string
    }>
    thinking?: string
    has_thinking: boolean
  }>> {
    return this.request('/mcp/chat', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async mcpQuery(query: string, options?: {
    top_k?: number
    temperature?: number
    max_tokens?: number
    enable_tools?: boolean
    allowed_tools?: string[]
    filters?: Record<string, any>
  }): Promise<APIResponse<{
    answer: string
    sources: any[]
    tool_calls: any[]
  }>> {
    return this.request('/mcp/query', {
      method: 'POST',
      body: JSON.stringify({ query, ...options }),
    })
  }

  // Usage tracking API methods
  async getUsageStats(filter?: UsageFilter): Promise<APIResponse<UsageStats>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/usage/stats${queryString ? `?${queryString}` : ''}`)
  }

  async getUsageStatsByType(filter?: UsageFilter): Promise<APIResponse<UsageStatsByType>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/usage/stats/type${queryString ? `?${queryString}` : ''}`)
  }

  async getUsageStatsByProvider(filter?: UsageFilter): Promise<APIResponse<UsageStatsByProvider>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/usage/stats/provider${queryString ? `?${queryString}` : ''}`)
  }

  async getDailyUsage(days: number = 30): Promise<APIResponse<Record<string, UsageStats>>> {
    return this.request(`/v1/usage/stats/daily?days=${days}`)
  }

  async getTopModels(limit: number = 10): Promise<APIResponse<Record<string, number>>> {
    return this.request(`/v1/usage/stats/models?limit=${limit}`)
  }

  async getCostByProvider(startTime?: string, endTime?: string): Promise<APIResponse<Record<string, number>>> {
    const params = new URLSearchParams()
    if (startTime) params.append('start_time', startTime)
    if (endTime) params.append('end_time', endTime)
    const queryString = params.toString()
    return this.request(`/v1/usage/stats/cost${queryString ? `?${queryString}` : ''}`)
  }

  // Conversation management API methods
  async getConversations(limit: number = 50, offset: number = 0): Promise<APIResponse<Conversation[]>> {
    try {
      const response = await fetch(`${this.baseURL}/v1/conversations?limit=${limit}&offset=${offset}`)
      const data = await response.json()
      
      if (!response.ok) {
        return { error: data.error || `HTTP ${response.status}` }
      }
      
      // Handle paged response format: {success: true, data: [...], page: 1, ...}
      if (data.success && Array.isArray(data.data)) {
        return { data: data.data }
      }
      
      // Fallback to empty array if data structure is unexpected
      return { data: [] }
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Unknown error' }
    }
  }

  async createConversation(title: string = 'New Conversation'): Promise<APIResponse<Conversation>> {
    return this.request('/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({ title }),
    })
  }

  async getConversation(conversationID: string): Promise<APIResponse<ConversationHistory>> {
    return this.request(`/v1/conversations/${conversationID}`)
  }

  async deleteConversation(conversationID: string): Promise<APIResponse<any>> {
    return this.request(`/v1/conversations/${conversationID}`, {
      method: 'DELETE',
    })
  }

  async exportConversation(conversationID: string): Promise<APIResponse<any>> {
    return this.request(`/v1/conversations/${conversationID}/export`)
  }

  // RAG visualization API methods
  async getRAGQueries(filter?: RAGSearchFilter): Promise<APIResponse<RAGQueryRecord[]>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, String(v)))
          } else {
            params.append(key, String(value))
          }
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/rag/queries${queryString ? `?${queryString}` : ''}`)
  }

  async getRAGQuery(queryID: string): Promise<APIResponse<RAGQueryRecord>> {
    return this.request(`/v1/rag/queries/${queryID}`)
  }

  async getRAGVisualization(queryID: string): Promise<APIResponse<RAGQueryVisualization>> {
    return this.request(`/v1/rag/queries/${queryID}/visualization`)
  }

  async getRAGAnalytics(filter?: RAGSearchFilter): Promise<APIResponse<RAGAnalytics>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, String(v)))
          } else {
            params.append(key, String(value))
          }
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/rag/analytics${queryString ? `?${queryString}` : ''}`)
  }

  async getRAGPerformanceReport(filter?: RAGSearchFilter): Promise<APIResponse<RAGPerformanceReport>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, String(v)))
          } else {
            params.append(key, String(value))
          }
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/rag/performance${queryString ? `?${queryString}` : ''}`)
  }

  // Tool calls tracking and visualization API methods
  async getToolCalls(filter?: ToolCallFilter): Promise<APIResponse<ToolCallRecord[]>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/tool-calls${queryString ? `?${queryString}` : ''}`)
  }

  async getToolCall(uuid: string): Promise<APIResponse<ToolCallRecord>> {
    return this.request(`/v1/tool-calls/${uuid}`)
  }

  async getToolCallVisualization(uuid: string): Promise<APIResponse<ToolCallVisualization>> {
    return this.request(`/v1/tool-calls/${uuid}/visualization`)
  }

  async getToolCallStats(filter?: ToolCallFilter): Promise<APIResponse<ToolCallStats>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/tool-calls/stats${queryString ? `?${queryString}` : ''}`)
  }

  async getToolCallAnalytics(filter?: ToolCallFilter): Promise<APIResponse<ToolCallAnalytics>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/tool-calls/analytics${queryString ? `?${queryString}` : ''}`)
  }

  async getToolCallsBySession(sessionId: string, limit: number = 50, offset: number = 0): Promise<APIResponse<ToolCallRecord[]>> {
    return this.request(`/v1/tool-calls/session/${sessionId}?limit=${limit}&offset=${offset}`)
  }

  async getToolCallsByConversation(conversationId: string, limit: number = 50, offset: number = 0): Promise<APIResponse<ToolCallRecord[]>> {
    return this.request(`/v1/tool-calls/conversation/${conversationId}?limit=${limit}&offset=${offset}`)
  }

  async getToolCallsByRAGQuery(ragQueryId: string): Promise<APIResponse<ToolCallRecord[]>> {
    return this.request(`/v1/tool-calls/rag-query/${ragQueryId}`)
  }

  async searchToolCalls(query: string, filter?: ToolCallFilter): Promise<APIResponse<{results: ToolCallRecord[], count: number}>> {
    const params = new URLSearchParams({ q: query })
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    return this.request(`/v1/tool-calls/search?${params.toString()}`)
  }

  async deleteToolCall(uuid: string): Promise<APIResponse<any>> {
    return this.request(`/v1/tool-calls/${uuid}`, {
      method: 'DELETE',
    })
  }

  async exportToolCalls(filter?: ToolCallFilter): Promise<APIResponse<any>> {
    const params = new URLSearchParams()
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          params.append(key, String(value))
        }
      })
    }
    const queryString = params.toString()
    return this.request(`/v1/tool-calls/export${queryString ? `?${queryString}` : ''}`)
  }
}

export const apiClient = new APIClient()

export function useRAGChat() {
  const [isLoading, setIsLoading] = useState(false)
  const [messages, setMessages] = useState<Array<{ role: 'user' | 'assistant'; content: string; sources?: SearchResult[] }>>([])

  const sendMessage = async (content: string, filters?: Record<string, any>, showThinking = true) => {
    setIsLoading(true)
    setMessages(prev => [...prev, { role: 'user', content }])

    try {
      const response = await apiClient.query(content, false, filters, showThinking)
      if (response.data) {
        setMessages(prev => [...prev, { 
          role: 'assistant', 
          content: response.data!.answer,
          sources: response.data!.sources 
        }])
      } else {
        setMessages(prev => [...prev, { 
          role: 'assistant', 
          content: `Error: ${response.error}` 
        }])
      }
    } catch (error) {
      setMessages(prev => [...prev, { 
        role: 'assistant', 
        content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}` 
      }])
    } finally {
      setIsLoading(false)
    }
  }

  const sendMessageStream = async (content: string, filters?: Record<string, any>, onChunk?: (chunk: string) => void, showThinking = true) => {
    setIsLoading(true)
    setMessages(prev => [...prev, { role: 'user', content }])

    try {
      // Add empty assistant message that will be updated
      const assistantMessageIndex = messages.length + 1
      setMessages(prev => [...prev, { role: 'assistant', content: '' }])

      const response = await fetch(`${apiClient['baseURL']}/query-stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ query: content, filters, show_thinking: showThinking }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || `HTTP ${response.status}`)
      }

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()
      let fullContent = ''

      if (reader) {
        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value, { stream: true })
          fullContent += chunk
          onChunk?.(chunk)
          
          // Update the assistant message
          setMessages(prev => {
            const newMessages = [...prev]
            newMessages[assistantMessageIndex] = {
              ...newMessages[assistantMessageIndex],
              content: fullContent
            }
            return newMessages
          })
        }
      }
    } catch (error) {
      setMessages(prev => {
        const newMessages = [...prev]
        newMessages[newMessages.length - 1] = {
          role: 'assistant',
          content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
        }
        return newMessages
      })
    } finally {
      setIsLoading(false)
    }
  }

  const clearMessages = () => setMessages([])

  return {
    messages,
    isLoading,
    sendMessage,
    sendMessageStream,
    clearMessages,
  }
}

// Conversation API
const API_BASE = '/api'

export const conversationApi = {
  // Create new conversation
  createNew: async (): Promise<{ id: string }> => {
    const response = await fetch(`${API_BASE}/conversations/new`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })
    
    if (!response.ok) {
      throw new Error('Failed to create new conversation')
    }
    
    const data = await response.json()
    return data.data || data
  },

  // Save conversation
  save: async (conversation: {
    id?: string
    title?: string
    messages: ConversationMessage[]
    metadata?: Record<string, any>
  }) => {
    const response = await fetch(`${API_BASE}/conversations/save`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(conversation),
    })
    
    if (!response.ok) {
      throw new Error('Failed to save conversation')
    }
    
    const data = await response.json()
    return data.data || data
  },

  // Get conversation by ID
  get: async (id: string): Promise<Conversation> => {
    const response = await fetch(`${API_BASE}/conversations/${id}`)
    
    if (!response.ok) {
      throw new Error('Failed to get conversation')
    }
    
    const data = await response.json()
    return data.data || data
  },

  // List conversations
  list: async (page = 1, pageSize = 20): Promise<ConversationListResponse> => {
    const response = await fetch(
      `${API_BASE}/conversations?page=${page}&page_size=${pageSize}`
    )
    
    if (!response.ok) {
      throw new Error('Failed to list conversations')
    }
    
    const data = await response.json()
    return data.data || data
  },

  // Delete conversation
  delete: async (id: string) => {
    const response = await fetch(`${API_BASE}/conversations/${id}`, {
      method: 'DELETE',
    })
    
    if (!response.ok) {
      throw new Error('Failed to delete conversation')
    }
    
    return true
  },

  // Search conversations
  search: async (query: string, page = 1, pageSize = 20): Promise<ConversationListResponse> => {
    const response = await fetch(
      `${API_BASE}/conversations/search?q=${encodeURIComponent(query)}&page=${page}&page_size=${pageSize}`
    )
    
    if (!response.ok) {
      throw new Error('Failed to search conversations')
    }
    
    const data = await response.json()
    return data.data || data
  },
}