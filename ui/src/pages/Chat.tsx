import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../lib/api'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
  debug?: {
    sessionId: string
    timestamp: number
    latency?: number
  }
}

export function Chat() {
  const { t } = useTranslation()
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [sessionId, setSessionId] = useState(() => crypto.randomUUID())
  const [useStream, setUseStream] = useState(true)
  const [debugMode, setDebugMode] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isStreaming) return

    const startTime = Date.now()
    const userMessage: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: input,
      debug: debugMode ? { sessionId, timestamp: startTime } : undefined,
    }
    setMessages((prev) => [...prev, userMessage])
    setInput('')

    // Create placeholder for streaming response
    const assistantId = crypto.randomUUID()
    if (useStream) {
      setMessages((prev) => [
        ...prev,
        { id: assistantId, role: 'assistant', content: '', streaming: true },
      ])
      setIsStreaming(true)

      try {
        await api.chatStream(
          { message: input, session_id: sessionId },
          (chunk) => {
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantId
                  ? { ...msg, content: msg.content + chunk }
                  : msg
              )
            )
          },
          (error) => {
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantId
                  ? { ...msg, content: `{t('error')}: ${error.message}`, streaming: false }
                  : msg
              )
            )
            setIsStreaming(false)
          },
          () => {
            const latency = Date.now() - startTime
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantId
                  ? {
                      ...msg,
                      streaming: false,
                      debug: debugMode ? { sessionId, timestamp: Date.now(), latency } : undefined,
                    }
                  : msg
              )
            )
            setIsStreaming(false)
          }
        )
      } catch (error) {
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === assistantId
              ? {
                  ...msg,
                  content: `{t('error')}: ${error instanceof Error ? error.message : 'Unknown error'}`,
                  streaming: false,
                }
              : msg
          )
        )
        setIsStreaming(false)
      }
    } else {
      // Non-streaming fallback
      try {
        const response = await api.chat({ message: input, session_id: sessionId })
        const latency = Date.now() - startTime
        const assistantMessage: Message = {
          id: assistantId,
          role: 'assistant',
          content: response.response,
          debug: debugMode ? { sessionId, timestamp: Date.now(), latency } : undefined,
        }
        setMessages((prev) => [...prev, assistantMessage])
      } catch (error) {
        const errorMessage: Message = {
          id: assistantId,
          role: 'assistant',
          content: `{t('error')}: ${error instanceof Error ? error.message : 'Unknown error'}`,
        }
        setMessages((prev) => [...prev, errorMessage])
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
        <h2 className="text-xl font-semibold text-slate-900">{t('chat')}</h2>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input
              type="checkbox"
              checked={useStream}
              onChange={(e) => setUseStream(e.target.checked)}
              className="rounded border-slate-300"
            />
            Stream
          </label>
          <label className="flex items-center gap-2 text-sm text-slate-600">
            <input
              type="checkbox"
              checked={debugMode}
              onChange={(e) => setDebugMode(e.target.checked)}
              className="rounded border-slate-300"
            />
            Debug
          </label>
          <button
            onClick={handleClear}
            className="dashboard-secondary-button px-3 py-1 text-sm"
          >
            Clear
          </button>
        </div>
      </div>

      {debugMode && (
        <div className="dashboard-muted-card mb-4 rounded-2xl p-3 text-xs font-mono text-slate-600">
          Session: {sessionId}
        </div>
      )}

      <div className="glass-panel flex-1 overflow-y-auto rounded-[28px] p-4 mb-4">
        {messages.length === 0 && (
          <div className="text-center text-slate-500 py-8">
            Start a conversation...
          </div>
        )}
        {messages.map((message) => (
          <div
            key={message.id}
            className={`mb-4 ${
              message.role === 'user' ? 'text-right' : 'text-left'
            }`}
          >
            <div
              className={`inline-block max-w-[80%] px-4 py-2 rounded-lg ${
                message.role === 'user'
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-50 text-slate-900 border border-slate-200'
              }`}
            >
              <p className="whitespace-pre-wrap">
                {message.content}
                {message.streaming && (
                  <span className="inline-block w-2 h-4 ml-1 bg-blue-600 animate-pulse" />
                )}
              </p>
              {debugMode && message.debug && (
                <div className="mt-2 pt-2 border-t border-slate-200 text-xs font-mono opacity-70">
                  {message.debug.latency && <span>Latency: {message.debug.latency}ms</span>}
                </div>
              )}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSend} className="flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type a message..."
          className="dashboard-input flex-1"
          disabled={isStreaming}
        />
        <button
          type="submit"
          disabled={isStreaming || !input.trim()}
          className="dashboard-button px-6 py-2 disabled:cursor-not-allowed"
        >
          {isStreaming ? t('sending') : t('sendMessage')}
        </button>
      </form>
    </div>
  )
}
