import { useState, useRef, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useRAGChat } from '@/lib/api'
import { Send, MessageSquare, User, Bot, ExternalLink, Settings, Zap } from 'lucide-react'

export function ChatTab() {
  const { messages, isLoading, sendMessage, sendMessageStream, clearMessages } = useRAGChat()
  const [input, setInput] = useState('')
  const [filters, setFilters] = useState('')
  const [useStreaming, setUseStreaming] = useState(true)
  const [showThinking, setShowThinking] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isLoading) return

    const message = input.trim()
    setInput('')

    // Parse filters if provided
    let parsedFilters: Record<string, any> | undefined
    if (filters.trim()) {
      try {
        parsedFilters = JSON.parse(filters)
      } catch (error) {
        // If filters are invalid, show error but continue without filters
        console.warn('Invalid filter JSON:', error)
      }
    }

    if (useStreaming) {
      await sendMessageStream(message, parsedFilters, undefined, showThinking)
    } else {
      await sendMessage(message, parsedFilters, showThinking)
    }
  }

  return (
    <div className="flex flex-col h-[600px]">
      <Card className="flex-1 flex flex-col">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-left">
            <MessageSquare className="h-5 w-5" />
            Chat with your Documents
          </CardTitle>
          <CardDescription className="text-left">
            Ask questions about your ingested documents and get AI-powered answers with sources.
          </CardDescription>
        </CardHeader>
        
        <CardContent className="flex-1 flex flex-col">
          <div className="flex-1 overflow-y-auto space-y-4 mb-4 p-4 border rounded-lg bg-gray-50">
            {messages.length === 0 ? (
              <div className="text-center text-gray-500 py-8">
                <MessageSquare className="mx-auto h-12 w-12 mb-4 opacity-50" />
                <p>Start a conversation by asking a question about your documents.</p>
              </div>
            ) : (
              messages.map((message, index) => (
                <div key={index} className="space-y-2">
                  <div className={`flex items-start gap-3 ${
                    message.role === 'user' ? 'justify-end' : 'justify-start'
                  }`}>
                    <div className={`flex gap-2 max-w-[80%] ${
                      message.role === 'user' ? 'flex-row-reverse' : 'flex-row'
                    }`}>
                      <div className={`w-8 h-8 min-w-[2rem] min-h-[2rem] flex-shrink-0 rounded-full flex items-center justify-center text-white text-sm shadow-md border-2 ${
                        message.role === 'user' 
                          ? 'bg-blue-500 border-blue-600' 
                          : 'bg-gradient-to-br from-purple-500 to-blue-600 border-purple-400'
                      }`}>
                        {message.role === 'user' ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
                      </div>
                      <div className={`rounded-lg p-3 ${
                        message.role === 'user' 
                          ? 'bg-blue-500 text-white text-right' 
                          : 'bg-white border shadow-sm text-left'
                      }`}>
                        <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                        
                        {message.sources && message.sources.length > 0 && (
                          <div className="mt-3 pt-3 border-t border-gray-200">
                            <p className="text-xs font-semibold text-gray-600 mb-2">Sources:</p>
                            <div className="space-y-1">
                              {message.sources.map((source, idx) => (
                                <div key={idx} className="text-xs bg-gray-50 p-2 rounded border">
                                  <div className="flex items-center gap-1 font-medium">
                                    <ExternalLink className="h-3 w-3" />
                                    {source.title || `Document ${source.id}`}
                                    <span className="text-gray-500">({(source.score * 100).toFixed(1)}%)</span>
                                  </div>
                                  <p className="text-gray-600 mt-1 line-clamp-2">
                                    {source.content.substring(0, 150)}...
                                  </p>
                                  {source.metadata && (
                                    <div className="mt-1 text-xs text-gray-500">
                                      {Object.entries(source.metadata).map(([key, value]) => (
                                        <span key={key} className="mr-2">
                                          {key}: {JSON.stringify(value)}
                                        </span>
                                      ))}
                                    </div>
                                  )}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              ))
            )}
            <div ref={messagesEndRef} />
          </div>

          {/* Advanced Options */}
          <div className="mb-4">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-xs"
            >
              <Settings className="h-3 w-3 mr-1" />
              Advanced Options
            </Button>
            
            {showAdvanced && (
              <div className="mt-2 space-y-3 p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="streaming"
                    checked={useStreaming}
                    onChange={(e) => setUseStreaming(e.target.checked)}
                    className="rounded"
                  />
                  <label htmlFor="streaming" className="text-sm flex items-center gap-1">
                    <Zap className="h-3 w-3" />
                    Enable Streaming
                  </label>
                </div>
                
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="thinking"
                    checked={showThinking}
                    onChange={(e) => setShowThinking(e.target.checked)}
                    className="rounded"
                  />
                  <label htmlFor="thinking" className="text-sm flex items-center gap-1">
                    <Bot className="h-3 w-3" />
                    Show AI Thinking Process
                  </label>
                </div>
                
                <div>
                  <label htmlFor="filters" className="text-xs font-medium text-gray-700 block mb-1">
                    Filters (JSON format)
                  </label>
                  <Textarea
                    id="filters"
                    value={filters}
                    onChange={(e) => setFilters(e.target.value)}
                    placeholder='{"author": "John Doe", "category": "technical"}'
                    className="text-xs h-16"
                    disabled={isLoading}
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Filter documents by metadata (e.g., author, category, tags)
                  </p>
                </div>
              </div>
            )}
          </div>

          <form onSubmit={handleSubmit} className="flex gap-2">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Ask a question about your documents..."
              disabled={isLoading}
              className="flex-1"
            />
            <Button type="submit" disabled={isLoading || !input.trim()}>
              <Send className="h-4 w-4" />
            </Button>
            {messages.length > 0 && (
              <Button type="button" variant="outline" onClick={clearMessages}>
                Clear
              </Button>
            )}
          </form>
        </CardContent>
      </Card>
    </div>
  )
}