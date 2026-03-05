const API_BASE = '/api'

export interface QueryRequest {
  query: string
  collection?: string
  top_k?: number
  stream?: boolean
}

export interface QueryResult {
  answer: string
  sources: Source[]
}

export interface Source {
  content: string
  score: number
  metadata?: Record<string, unknown>
}

export interface Document {
  id: string
  filename: string
  collection: string
  created_at: string
  chunks?: number
}

export interface Collection {
  name: string
  count: number
}

export interface ChatRequest {
  message: string
  session_id?: string
  stream?: boolean
}

export interface ChatResponse {
  response: string
  session_id: string
}

export interface StatusResponse {
  status: string
  version: string
  providers: ProviderStatus[]
}

export interface ProviderStatus {
  name: string
  status: 'online' | 'offline' | 'unknown'
  latency?: number
  type?: string
}

// Skills types
export interface Skill {
  id: string
  name: string
  description: string
  version: string
  author?: string
  category?: string
  tags?: string[]
  enabled: boolean
  user_invocable?: boolean
  path: string
  created_at: string
  updated_at: string
  variables?: Record<string, VariableDef>
  steps?: SkillStep[]
}

export interface VariableDef {
  name: string
  description: string
  type: string
  required: boolean
  default?: unknown
  pattern?: string
}

export interface SkillStep {
  id: string
  title: string
  description: string
  content: string
  interactive: boolean
  confirm: boolean
}

export interface CreateSkillRequest {
  name: string
  description: string
  content: string
  variables?: Record<string, unknown>
}

// MCP types
export interface MCPServer {
  name: string
  description: string
  command: string
  running: boolean
  tool_count: number
  tools?: MCPToolSummary[]
}

export interface MCPToolSummary {
  name: string
  description: string
  server_name: string
}

export interface MCPTool {
  server_name: string
  name: string
  description: string
  input_schema: Record<string, unknown>
  last_used?: string
  usage_count?: number
}

export interface AddMCPServerRequest {
  name: string
  command?: string
  args?: string[]
  type?: string
  url?: string
}

export interface CallToolRequest {
  tool_name: string
  arguments: Record<string, unknown>
}

export interface ToolResult {
  success: boolean
  data: unknown
  error?: string
}

// Memory types
export interface Memory {
  id: string
  type: string
  content: string
  importance: number
  session_id?: string
  created_at: string
  updated_at?: string
}

export interface AddMemoryRequest {
  content: string
  type: string
  importance: number
}

export interface UpdateMemoryRequest {
  id: string
  content: string
}

// Stream callback types
export type StreamCallback = (chunk: string) => void
export type StreamErrorCallback = (error: Error) => void
export type StreamCompleteCallback = () => void

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `API Error: ${response.status}`)
  }

  return response.json()
}

// Streaming fetch for SSE/NDJSON responses
async function streamFetch(
  path: string,
  options: RequestInit,
  onChunk: StreamCallback,
  onError?: StreamErrorCallback,
  onComplete?: StreamCompleteCallback
): Promise<void> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream',
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    const err = new Error(error.error || `API Error: ${response.status}`)
    onError?.(err)
    return
  }

  if (!response.body) {
    onComplete?.()
    return
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value)
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6)
          if (data === '[DONE]') {
            onComplete?.()
            return
          }
          try {
            const parsed = JSON.parse(data)
            if (parsed.content) {
              onChunk(parsed.content)
            } else if (parsed.delta?.content) {
              onChunk(parsed.delta.content)
            } else if (parsed.choices?.[0]?.delta?.content) {
              onChunk(parsed.choices[0].delta.content)
            }
          } catch {
            // Try as plain text if JSON parse fails
            if (data.trim()) {
              onChunk(data)
            }
          }
        } else if (line.trim() && !line.startsWith(':')) {
          // NDJSON format (newline-delimited JSON)
          try {
            const parsed = JSON.parse(line)
            if (parsed.content) {
              onChunk(parsed.content)
            } else if (parsed.delta?.content) {
              onChunk(parsed.delta.content)
            }
          } catch {
            // Plain text
            onChunk(line)
          }
        }
      }
    }

    onComplete?.()
  } catch (error) {
    onError?.(error instanceof Error ? error : new Error(String(error)))
  }
}

export const api = {
  query: (data: QueryRequest) =>
    fetchAPI<QueryResult>('/query', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Streaming query with callbacks
  queryStream: (
    data: QueryRequest,
    onChunk: StreamCallback,
    onError?: StreamErrorCallback,
    onComplete?: StreamCompleteCallback
  ) =>
    streamFetch('/query', {
      method: 'POST',
      body: JSON.stringify({ ...data, stream: true }),
    }, onChunk, onError, onComplete),

  getDocuments: () => fetchAPI<Document[]>('/documents'),

  getDocument: (id: string) => fetchAPI<Document>(`/documents/${id}`),

  deleteDocument: (id: string) =>
    fetchAPI<{ success: boolean; id: string }>(`/documents/${id}`, {
      method: 'DELETE',
    }),

  getCollections: () => fetchAPI<Collection[]>('/collections'),

  getStatus: () => fetchAPI<StatusResponse>('/status'),

  chat: (data: ChatRequest) =>
    fetchAPI<ChatResponse>('/chat', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Streaming chat with callbacks
  chatStream: (
    data: ChatRequest,
    onChunk: StreamCallback,
    onError?: StreamErrorCallback,
    onComplete?: StreamCompleteCallback
  ) =>
    streamFetch('/chat', {
      method: 'POST',
      body: JSON.stringify({ ...data, stream: true }),
    }, onChunk, onError, onComplete),

  ingest: (formData: FormData) =>
    fetch(`${API_BASE}/ingest`, {
      method: 'POST',
      body: formData,
    }).then((r) => r.json()),

  // Skills API
  getSkills: () => fetchAPI<Skill[]>('/skills'),

  getSkill: (id: string) => fetchAPI<Skill>(`/skills/${id}`),

  createSkill: (data: CreateSkillRequest) =>
    fetchAPI<{ success: boolean; skill: Skill }>('/skills/add', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  deleteSkill: (id: string) =>
    fetchAPI<{ success: boolean }>(`/skills/${id}`, {
      method: 'DELETE',
    }),

  // MCP API
  getMCPServers: () => fetchAPI<MCPServer[]>('/mcp/servers'),

  getMCPTools: () => fetchAPI<MCPTool[]>('/mcp/tools'),

  addMCPServer: (data: AddMCPServerRequest) =>
    fetchAPI<{ success: boolean; name: string }>('/mcp/add', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  callTool: (data: CallToolRequest) =>
    fetchAPI<ToolResult>('/mcp/call', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Memory API
  getMemories: () => fetchAPI<Memory[]>('/memories'),

  getMemory: (id: string) => fetchAPI<Memory>(`/memories/${id}`),

  addMemory: (data: AddMemoryRequest) =>
    fetchAPI<{ success: boolean; id: string }>('/memories/add', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  deleteMemory: (id: string) =>
    fetchAPI<{ success: boolean }>(`/memories/${id}`, {
      method: 'DELETE',
    }),

  searchMemories: (query: string) =>
    fetchAPI<Memory[]>(`/memories/search?q=${encodeURIComponent(query)}`),
}
