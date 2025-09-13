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
  options?: {
    temperature?: number
    max_tokens?: number
    show_thinking?: boolean
    allowed_tools?: string[]
    max_tool_calls?: number
  }
  context?: Record<string, any>
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

  async getDocuments(): Promise<APIResponse<Document[]>> {
    return this.request('/documents')
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