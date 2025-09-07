import { useState } from 'react'

// V3 Core Types
export interface Provider {
  name: string
  type: string
  model: string
  base_url?: string
  enabled: boolean
  weight: number
  status?: 'healthy' | 'unhealthy' | 'unknown'
}

export interface Document {
  id: string
  content: string
  metadata?: Record<string, any>
}

export interface RAGQuery {
  query: string
  top_k?: number
  filters?: Record<string, any>
}

export interface RAGResult {
  documents: Document[]
  answer?: string
}

export interface GenerationRequest {
  prompt: string
  provider?: string
  max_tokens?: number
  temperature?: number
  stream?: boolean
}

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface ChatRequest {
  messages: ChatMessage[]
  provider?: string
  max_tokens?: number
  temperature?: number
  stream?: boolean
}

export interface MCPServer {
  name: string
  command: string
  args?: string[]
  description: string
  enabled: boolean
  status?: 'running' | 'stopped' | 'error'
}

export interface MCPTool {
  name: string
  description: string
  server: string
  parameters?: Record<string, any>
}

export interface WorkflowStep {
  id: string
  type: 'llm_generate' | 'rag_query' | 'mcp_tool' | 'condition'
  description: string
  parameters: Record<string, any>
  depends_on?: string[]
}

export interface WorkflowDefinition {
  id: string
  name: string
  description: string
  steps: WorkflowStep[]
}

export interface AgentTask {
  id?: string
  goal: string
  context?: string
  workflow?: WorkflowDefinition
}

export interface HealthStatus {
  overall: 'healthy' | 'degraded' | 'unhealthy'
  providers: Record<string, string>
  rag: string
  mcp: {
    status: string
    servers: Record<string, string>
  }
  agents: string
}

export interface APIResponse<T> {
  data?: T
  error?: string
  success: boolean
}

class V3APIClient {
  private baseURL = '/api/v3'

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
        return { error: data.error || `HTTP ${response.status}`, success: false }
      }

      return { data, success: true }
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Unknown error', success: false }
    }
  }

  // LLM Pillar APIs
  async generate(request: GenerationRequest) {
    return this.request<{ content: string; provider: string }>('/llm/generate', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async chat(request: ChatRequest) {
    return this.request<{ content: string; provider: string }>('/llm/chat', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async streamGenerate(request: GenerationRequest, onChunk: (chunk: string) => void) {
    const response = await fetch(`${this.baseURL}/llm/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...request, stream: true }),
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const reader = response.body?.getReader()
    const decoder = new TextDecoder()

    if (!reader) throw new Error('No response body')

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      
      const chunk = decoder.decode(value)
      const lines = chunk.split('\n')
      
      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6)
          if (data === '[DONE]') return
          try {
            const parsed = JSON.parse(data)
            onChunk(parsed.content || '')
          } catch (e) {
            // Skip invalid JSON
          }
        }
      }
    }
  }

  async listProviders() {
    return this.request<Provider[]>('/llm/providers', {
      method: 'GET',
    })
  }

  // RAG Pillar APIs
  async ingestDocument(document: Document) {
    return this.request('/rag/ingest', {
      method: 'POST',
      body: JSON.stringify(document),
    })
  }

  async ingestFile(file: File, metadata?: Record<string, any>) {
    const formData = new FormData()
    formData.append('file', file)
    if (metadata) {
      formData.append('metadata', JSON.stringify(metadata))
    }

    try {
      const response = await fetch(`${this.baseURL}/rag/ingest/file`, {
        method: 'POST',
        body: formData,
      })

      const data = await response.json()

      if (!response.ok) {
        return { error: data.error || `HTTP ${response.status}`, success: false }
      }

      return { data, success: true }
    } catch (error) {
      return { error: error instanceof Error ? error.message : 'Unknown error', success: false }
    }
  }

  async queryRAG(query: RAGQuery) {
    return this.request<RAGResult>('/rag/query', {
      method: 'POST',
      body: JSON.stringify(query),
    })
  }

  async listDocuments() {
    return this.request<Document[]>('/rag/documents', {
      method: 'GET',
    })
  }

  async deleteDocument(id: string) {
    return this.request(`/rag/documents/${id}`, {
      method: 'DELETE',
    })
  }

  async resetRAG() {
    return this.request('/rag/reset', {
      method: 'POST',
    })
  }

  // MCP Pillar APIs
  async listMCPServers() {
    return this.request<MCPServer[]>('/mcp/servers', {
      method: 'GET',
    })
  }

  async listMCPTools() {
    return this.request<MCPTool[]>('/mcp/tools', {
      method: 'GET',
    })
  }

  async callMCPTool(server: string, tool: string, params: Record<string, any>) {
    return this.request('/mcp/tools/call', {
      method: 'POST',
      body: JSON.stringify({ server, tool, params }),
    })
  }

  async startMCPServer(name: string) {
    return this.request(`/mcp/servers/${name}/start`, {
      method: 'POST',
    })
  }

  async stopMCPServer(name: string) {
    return this.request(`/mcp/servers/${name}/stop`, {
      method: 'POST',
    })
  }

  // Agents Pillar APIs
  async executeWorkflow(workflow: WorkflowDefinition) {
    return this.request('/agents/workflow/execute', {
      method: 'POST',
      body: JSON.stringify(workflow),
    })
  }

  async generateWorkflow(task: string) {
    return this.request<WorkflowDefinition>('/agents/workflow/generate', {
      method: 'POST',
      body: JSON.stringify({ task }),
    })
  }

  async executeTask(task: AgentTask) {
    return this.request('/agents/task', {
      method: 'POST',
      body: JSON.stringify(task),
    })
  }

  async listScheduledTasks() {
    return this.request('/agents/scheduled', {
      method: 'GET',
    })
  }

  async scheduleTask(task: AgentTask, schedule: string) {
    return this.request('/agents/scheduled', {
      method: 'POST',
      body: JSON.stringify({ task, schedule }),
    })
  }

  // System APIs
  async getHealth() {
    return this.request<HealthStatus>('/health', {
      method: 'GET',
    })
  }

  async getConfig() {
    return this.request('/config', {
      method: 'GET',
    })
  }
}

export const api = new V3APIClient()

// Hooks for common operations
export function useAsyncOperation<T>() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<T | null>(null)

  const execute = async (operation: () => Promise<APIResponse<T>>) => {
    setLoading(true)
    setError(null)
    
    try {
      const result = await operation()
      if (result.error) {
        setError(result.error)
      } else {
        setData(result.data || null)
      }
      return result
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Unknown error'
      setError(errorMsg)
      return { error: errorMsg, success: false }
    } finally {
      setLoading(false)
    }
  }

  return { loading, error, data, execute }
}