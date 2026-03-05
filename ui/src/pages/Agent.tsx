import { useState, useRef, useEffect } from 'react'

interface AgentMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: number
  streaming?: boolean
  debug?: {
    duration?: number
    tokens?: number
    model?: string
  }
}

interface AgentRunRequest {
  message: string
  agent_name: string
  system_prompt: string
  debug: boolean
  stream?: boolean
  session_id?: string
}

interface AgentStreamResponse {
  content?: string
  error?: string
  done?: boolean
  debug?: {
    duration?: number
    tokens?: number
    model?: string
  }
}

export function Agent() {
  const [messages, setMessages] = useState<AgentMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [debugMode, setDebugMode] = useState(false)
  const [useStream, setUseStream] = useState(true)
  const [systemPrompt, setSystemPrompt] = useState('You are a helpful AI assistant.')
  const [agentName, setAgentName] = useState('default-agent')
  const [sessionId, setSessionId] = useState(() => crypto.randomUUID())
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const handleRun = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isLoading) return

    const startTime = Date.now()
    const userMessage: AgentMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: input,
      timestamp: startTime,
    }
    setMessages(prev => [...prev, userMessage])
    setInput('')
    setIsLoading(true)

    const assistantId = crypto.randomUUID()

    if (useStream) {
      // Streaming mode
      setMessages(prev => [...prev, {
        id: assistantId,
        role: 'assistant',
        content: '',
        timestamp: Date.now(),
        streaming: true,
      }])

      try {
        const response = await fetch('/api/agent/stream', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            message: input,
            agent_name: agentName,
            system_prompt: systemPrompt,
            debug: debugMode,
            session_id: sessionId,
          }),
        })

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`)
        }

        const reader = response.body?.getReader()
        const decoder = new TextDecoder()

        if (!reader) {
          throw new Error('No reader available')
        }

        let buffer = ''
        let duration = 0

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const data = line.slice(6)
              if (data === '[DONE]') {
                setMessages(prev => prev.map(m =>
                  m.id === assistantId
                    ? { ...m, streaming: false, debug: debugMode ? { duration: Date.now() - startTime } : undefined }
                    : m
                ))
                break
              }
              try {
                const parsed = JSON.parse(data)
                if (parsed.content) {
                  setMessages(prev => prev.map(m =>
                    m.id === assistantId
                      ? { ...m, content: m.content + parsed.content }
                      : m
                  ))
                } else if (parsed.error) {
                  setMessages(prev => prev.map(m =>
                    m.id === assistantId
                      ? { ...m, content: `Error: ${parsed.error}`, streaming: false }
                      : m
                  ))
                  break
                }
              } catch {
                // Plain text
                if (data.trim()) {
                  setMessages(prev => prev.map(m =>
                    m.id === assistantId
                      ? { ...m, content: m.content + data }
                      : m
                  ))
                }
              }
            }
          }
        }

        setIsLoading(false)
      } catch (error) {
        setMessages(prev => prev.map(m =>
          m.id === assistantId
            ? { ...m, content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`, streaming: false }
            : m
        ))
        setIsLoading(false)
      }
    } else {
      // Non-streaming mode
      setMessages(prev => [...prev, {
        id: assistantId,
        role: 'system',
        content: 'Agent is thinking...',
        timestamp: Date.now(),
      }])

      try {
        const response = await fetch('/api/agent/run', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            message: input,
            agent_name: agentName,
            system_prompt: systemPrompt,
            debug: debugMode,
          }),
        })

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`)
        }

        const result = await response.json()
        const duration = Date.now() - startTime

        setMessages(prev => prev.filter(m => m.id !== assistantId))

        const assistantMessage: AgentMessage = {
          id: crypto.randomUUID(),
          role: 'assistant',
          content: result.response || result.output || JSON.stringify(result, null, 2),
          timestamp: Date.now(),
          debug: debugMode ? { duration } : undefined,
        }
        setMessages(prev => [...prev, assistantMessage])
      } catch (error) {
        setMessages(prev => prev.filter(m => m.id !== assistantId))
        const errorMessage: AgentMessage = {
          id: crypto.randomUUID(),
          role: 'system',
          content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
          timestamp: Date.now(),
        }
        setMessages(prev => [...prev, errorMessage])
      } finally {
        setIsLoading(false)
      }
    }
  }

  const handleClear = () => {
    setMessages([])
    setSessionId(crypto.randomUUID())
  }

  return (
    <div className="flex flex-col h-[calc(100vh-200px)]">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Agent</h2>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <input
              type="checkbox"
              checked={useStream}
              onChange={(e) => setUseStream(e.target.checked)}
              className="rounded border-gray-300 dark:border-gray-600"
            />
            Stream
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <input
              type="checkbox"
              checked={debugMode}
              onChange={(e) => setDebugMode(e.target.checked)}
              className="rounded border-gray-300 dark:border-gray-600"
            />
            Debug
          </label>
          <button
            onClick={handleClear}
            className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-800 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
          >
            Clear
          </button>
        </div>
      </div>

      {debugMode && (
        <div className="mb-4 space-y-2">
          <div className="flex items-center gap-2">
            <label className="text-xs text-gray-500 dark:text-gray-400 w-24">Agent Name:</label>
            <input
              type="text"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              className="flex-1 px-2 py-1 text-xs border border-gray-300 rounded dark:bg-gray-800 dark:border-gray-600 dark:text-white"
            />
          </div>
          <div className="flex items-start gap-2">
            <label className="text-xs text-gray-500 dark:text-gray-400 w-24 pt-1">System Prompt:</label>
            <textarea
              value={systemPrompt}
              onChange={(e) => setSystemPrompt(e.target.value)}
              rows={2}
              className="flex-1 px-2 py-1 text-xs border border-gray-300 rounded dark:bg-gray-800 dark:border-gray-600 dark:text-white font-mono"
            />
          </div>
        </div>
      )}

      <div className="flex-1 overflow-y-auto bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4">
        {messages.length === 0 && (
          <div className="text-center text-gray-500 dark:text-gray-400 py-8">
            Configure the agent and send a message to start...
          </div>
        )}
        {messages.map((message) => (
          <div
            key={message.id}
            className={`mb-4 ${
              message.role === 'user' ? 'text-right' :
              message.role === 'system' ? 'text-center' : 'text-left'
            }`}
          >
            <div
              className={`inline-block max-w-[80%] px-4 py-2 rounded-lg ${
                message.role === 'user'
                  ? 'bg-blue-600 text-white'
                  : message.role === 'system'
                  ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 text-sm'
                  : 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white'
              }`}
            >
              <p className="whitespace-pre-wrap">
                {message.content}
                {message.streaming && (
                  <span className="inline-block w-2 h-4 ml-1 bg-blue-600 dark:bg-blue-400 animate-pulse" />
                )}
              </p>
              {debugMode && message.debug && !message.streaming && (
                <div className="mt-2 pt-2 border-t border-gray-200 dark:border-gray-600 text-xs font-mono opacity-70">
                  {message.debug.duration && <span>Duration: {message.debug.duration}ms </span>}
                  {message.debug.tokens && <span>Tokens: {message.debug.tokens} </span>}
                  {message.debug.model && <span>Model: {message.debug.model}</span>}
                </div>
              )}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleRun} className="flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Send a message to the agent..."
          className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-800 dark:border-gray-600 dark:text-white"
          disabled={isLoading}
        />
        <button
          type="submit"
          disabled={isLoading || !input.trim()}
          className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isLoading ? 'Running...' : 'Run'}
        </button>
      </form>
    </div>
  )
}
