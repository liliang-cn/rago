import { useState, useRef, useEffect } from 'react'
import { api } from '../lib/api'
import type { ChatRequest } from '../lib/api'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
}

export function Chat() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [sessionId] = useState(() => crypto.randomUUID())
  const [useStream, setUseStream] = useState(true)
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

    const userMessage: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: input,
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
                  ? { ...msg, content: `Error: ${error.message}`, streaming: false }
                  : msg
              )
            )
            setIsStreaming(false)
          },
          () => {
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantId ? { ...msg, streaming: false } : msg
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
                  content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
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
        const assistantMessage: Message = {
          id: assistantId,
          role: 'assistant',
          content: response.response,
        }
        setMessages((prev) => [...prev, assistantMessage])
      } catch (error) {
        const errorMessage: Message = {
          id: assistantId,
          role: 'assistant',
          content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
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
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Chat</h2>
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
            <input
              type="checkbox"
              checked={useStream}
              onChange={(e) => setUseStream(e.target.checked)}
              className="rounded border-gray-300 dark:border-gray-600"
            />
            Stream responses
          </label>
          <button
            onClick={handleClear}
            className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-800 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
          >
            Clear
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto bg-gray-50 dark:bg-gray-800 rounded-lg p-4 mb-4">
        {messages.length === 0 && (
          <div className="text-center text-gray-500 dark:text-gray-400 py-8">
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
                  : 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white'
              }`}
            >
              <p className="whitespace-pre-wrap">
                {message.content}
                {message.streaming && (
                  <span className="inline-block w-2 h-4 ml-1 bg-blue-600 dark:bg-blue-400 animate-pulse" />
                )}
              </p>
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
          className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-800 dark:border-gray-600 dark:text-white"
          disabled={isStreaming}
        />
        <button
          type="submit"
          disabled={isStreaming || !input.trim()}
          className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isStreaming ? 'Sending...' : 'Send'}
        </button>
      </form>
    </div>
  )
}

function setSessionId(id: string) {
  // This is a placeholder - in a real app you'd manage session state properly
  console.log('Session ID:', id)
}
