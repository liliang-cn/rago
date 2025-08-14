import { useState } from 'react'

export interface Document {
  id: string
  title?: string
  content: string
  created: string
  metadata?: Record<string, any>
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

  async query(question: string, context_only = false, filters?: Record<string, any>, show_thinking = false): Promise<APIResponse<{ answer: string; sources: SearchResult[] }>> {
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

  async getDocuments(): Promise<APIResponse<Document[]>> {
    return this.request('/documents')
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
}

export const apiClient = new APIClient()

export function useRAGChat() {
  const [isLoading, setIsLoading] = useState(false)
  const [messages, setMessages] = useState<Array<{ role: 'user' | 'assistant'; content: string; sources?: SearchResult[] }>>([])

  const sendMessage = async (content: string, filters?: Record<string, any>, showThinking = false) => {
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

  const sendMessageStream = async (content: string, filters?: Record<string, any>, onChunk?: (chunk: string) => void, showThinking = false) => {
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