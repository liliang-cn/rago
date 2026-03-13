import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useChat } from '@ai-sdk/react'
import { DefaultChatTransport } from 'ai'
import { useSquads } from '../hooks/useApi'

type ChatMode = 'rag' | 'agent'

type UIMessagePart = {
  type: string
  text?: string
  state?: string
  toolName?: string
  input?: unknown
  output?: unknown
  errorText?: string
  data?: Record<string, unknown>
}

type UIMessage = {
  id: string
  role: 'user' | 'assistant' | 'system'
  metadata?: Record<string, unknown>
  parts?: UIMessagePart[]
}

function stringify(value: unknown) {
  if (value == null) return ''
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function getTextParts(message: UIMessage) {
  return (message.parts ?? []).filter((part) => part.type === 'text' && typeof part.text === 'string')
}

function getReasoningParts(message: UIMessage) {
  return (message.parts ?? []).filter((part) => part.type === 'reasoning' && typeof part.text === 'string')
}

function getToolParts(message: UIMessage) {
  return (message.parts ?? []).filter((part) => part.type === 'dynamic-tool' || part.type.startsWith('tool-'))
}

function getAgentEventParts(message: UIMessage) {
  return (message.parts ?? []).filter((part) => part.type === 'data-agent-event')
}

function renderToolState(part: UIMessagePart) {
  switch (part.state) {
    case 'input-streaming':
      return 'input'
    case 'input-available':
      return 'queued'
    case 'output-available':
      return 'done'
    case 'output-error':
      return 'error'
    case 'output-denied':
      return 'denied'
    default:
      return part.state ?? 'unknown'
  }
}

export function Chat() {
  const { t } = useTranslation()
  const [input, setInput] = useState('')
  const [chatMode, setChatMode] = useState<ChatMode>('rag')
  const [debugEnabled, setDebugEnabled] = useState(false)
  const [selectedAgent, setSelectedAgent] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const { data: squads = [] } = useSquads()
  const leadAgents = useMemo(
    () =>
      squads
        .map((squad) => squad.lead_agent ?? squad.captain)
        .filter((agent): agent is NonNullable<typeof agent> => Boolean(agent)),
    [squads],
  )

  const { messages, sendMessage, status, error, setMessages } = useChat({
    transport: new DefaultChatTransport({
      api: '/api/chat',
      prepareSendMessagesRequest: ({ body, messages }) => {
        const requestBody = (body ?? {}) as Record<string, unknown>
        const lastUserMessage = [...messages]
          .reverse()
          .find((message) => message.role === 'user')
        const parts = (Array.isArray(lastUserMessage?.parts) ? lastUserMessage.parts : []) as Array<any>
        const messageText = parts
          .filter((part) => Boolean(part) && typeof part === 'object' && part.type === 'text' && typeof part.text === 'string')
          .map((part) => String(part.text))
          .join('\n')

        return {
          api: '/api/chat',
          body: {
            ...requestBody,
            ...(messageText ? { message: messageText } : {}),
            mode: chatMode,
            ...(chatMode === 'agent' && selectedAgent ? { agent_name: selectedAgent } : {}),
          },
        }
      },
    }),
  })

  useEffect(() => {
    if (!selectedAgent && leadAgents.length > 0) {
      setSelectedAgent(leadAgents[0].name)
    }
  }, [leadAgents, selectedAgent])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const isStreaming = status === 'submitted' || status === 'streaming'
  const totalParts = useMemo(
    () => messages.reduce((sum, message) => sum + (((message as UIMessage).parts?.length) ?? 0), 0),
    [messages],
  )
  const agentEvents = useMemo(
    () => messages.flatMap((message) => getAgentEventParts(message as UIMessage)),
    [messages],
  )
  const lastAssistant = [...messages].reverse().find((message) => message.role === 'assistant') as UIMessage | undefined

  const handleSend = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!input.trim() || isStreaming) return

    const text = input
    setInput('')
    await sendMessage({
      role: 'user',
      parts: [{ type: 'text', text }],
    })
  }

  return (
    <div className="flex h-[calc(100vh-200px)] flex-col gap-4" data-testid="page-chat">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold text-slate-900">{t('chat')}</h2>
          <p className="mt-1 text-sm text-slate-500">{t('chatWorkbench')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2" data-testid="chat-toolbar">
          <label className="inline-flex items-center gap-3 rounded-xl border border-sky-100 bg-white px-3 py-2 text-sm text-slate-700" data-testid="chat-debug-toggle">
            <span>{t('debug')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={debugEnabled}
              onClick={() => setDebugEnabled((value) => !value)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${debugEnabled ? 'bg-blue-600' : 'bg-slate-200'}`}
              data-testid="chat-debug-switch"
            >
              <span
                className={`inline-block h-5 w-5 transform rounded-full bg-white transition-transform ${debugEnabled ? 'translate-x-5' : 'translate-x-1'}`}
              />
            </button>
          </label>
          <button
            onClick={() => {
              setMessages([])
              setInput('')
            }}
            className="dashboard-secondary-button px-3 py-2 text-sm"
            data-testid="chat-clear"
          >
            {t('clear')}
          </button>
        </div>
      </div>

      <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="glass-panel rounded-[24px] p-4" data-testid="chat-console">
          <div className="mb-4 grid gap-3 md:grid-cols-[180px_220px_minmax(0,1fr)]">
            <label className="text-sm text-slate-600">
              <span className="mb-1 block font-medium text-slate-700">{t('chatMode')}</span>
              <select
                value={chatMode}
                onChange={(e) => setChatMode(e.target.value as ChatMode)}
                className="dashboard-input"
                data-testid="chat-mode"
              >
                <option value="rag">{t('chatModeRag')}</option>
                <option value="agent">{t('chatModeAgent')}</option>
              </select>
            </label>

            <label className="text-sm text-slate-600">
              <span className="mb-1 block font-medium text-slate-700">{t('chatTargetAgent')}</span>
              <select
                value={selectedAgent}
                onChange={(e) => setSelectedAgent(e.target.value)}
                className="dashboard-input"
                disabled={chatMode !== 'agent' || leadAgents.length === 0}
                data-testid="chat-agent-select"
              >
                {leadAgents.length === 0 && <option value="">{t('loadingAgents')}</option>}
                {leadAgents.map((agent) => (
                  <option key={agent.id} value={agent.name}>
                    {agent.name}
                  </option>
                ))}
              </select>
            </label>

            <div className="dashboard-muted-card rounded-2xl p-3 text-xs text-slate-600" data-testid="chat-session-meta">
              <div className="font-mono text-[11px] text-slate-500">
                {t('chatStatusLine', {
                  id: lastAssistant?.id ?? '-',
                  status,
                  messages: messages.length,
                  parts: totalParts,
                })}
              </div>
              <div className="mt-2 flex flex-wrap gap-2">
                <span className="rounded-full bg-sky-100 px-2 py-1 text-[11px] text-sky-800">
                  {t(chatMode === 'agent' ? 'chatModeAgent' : 'chatModeRag')}
                </span>
                {chatMode === 'agent' && selectedAgent && (
                  <span className="rounded-full bg-blue-100 px-2 py-1 text-[11px] text-blue-800">{selectedAgent}</span>
                )}
                {debugEnabled && (
                  <span className="rounded-full bg-slate-100 px-2 py-1 text-[11px] text-slate-700">
                    {t('debug')}
                  </span>
                )}
              </div>
            </div>
          </div>

          <div className="max-h-[calc(100vh-410px)] space-y-4 overflow-y-auto pr-1" data-testid="chat-messages">
            {messages.length === 0 && <div className="py-10 text-center text-slate-500">{t('startConversation')}</div>}

            {messages.map((rawMessage) => {
              const message = rawMessage as UIMessage
              const textParts = getTextParts(message)
              const reasoningParts = getReasoningParts(message)
              const toolParts = getToolParts(message)
              const eventParts = getAgentEventParts(message)
              const isUser = message.role === 'user'
              const isAssistant = message.role === 'assistant'

              return (
                <div key={message.id} className={isUser ? 'text-right' : 'text-left'} data-testid={`chat-message-${message.role}`}>
                  <div
                    className={`inline-block max-w-[88%] rounded-2xl px-4 py-3 ${
                      isUser ? 'bg-blue-600 text-white' : 'border border-sky-100 bg-sky-50/80 text-slate-900'
                    }`}
                  >
                    {textParts.length > 0 && (
                      <div className="space-y-3">
                        {textParts.map((part, index) => (
                          <p key={`${message.id}-text-${index}`} className="whitespace-pre-wrap leading-7">
                            {part.text}
                            {isAssistant && isStreaming && part.state === 'streaming' && message.id === messages[messages.length - 1]?.id && (
                              <span className="ml-1 inline-block h-4 w-2 animate-pulse bg-blue-600 align-middle" />
                            )}
                          </p>
                        ))}
                      </div>
                    )}

                    {debugEnabled && reasoningParts.length > 0 && (
                      <div className="mt-3 rounded-xl border border-amber-200 bg-amber-50 p-3 text-left text-xs text-amber-900">
                        <div className="mb-2 font-semibold">{t('chatReasoning')}</div>
                        {reasoningParts.map((part, index) => (
                          <pre key={`${message.id}-reasoning-${index}`} className="whitespace-pre-wrap font-mono">
                            {part.text}
                          </pre>
                        ))}
                      </div>
                    )}

                    {toolParts.length > 0 && (
                      <div className="mt-3 space-y-2 text-left">
                        {toolParts.map((part, index) => (
                          <div key={`${message.id}-tool-${index}`} className="rounded-xl border border-sky-200 bg-white/90 p-3 text-xs text-slate-700">
                            <div className="flex flex-wrap items-center justify-between gap-2">
                              <div className="font-semibold text-slate-900">{part.toolName ?? part.type}</div>
                              <span className="rounded-full bg-slate-100 px-2 py-1 font-mono text-[11px] text-slate-600">
                                {renderToolState(part)}
                              </span>
                            </div>
                            {debugEnabled && (
                              <>
                                {part.input !== undefined && (
                                  <div className="mt-2">
                                    <div className="mb-1 font-medium text-slate-600">{t('chatToolInput')}</div>
                                    <pre className="overflow-x-auto rounded-lg bg-slate-950/95 p-3 text-[11px] text-sky-100">{stringify(part.input)}</pre>
                                  </div>
                                )}
                                {part.output !== undefined && (
                                  <div className="mt-2">
                                    <div className="mb-1 font-medium text-slate-600">{t('chatToolOutput')}</div>
                                    <pre className="overflow-x-auto rounded-lg bg-slate-950/95 p-3 text-[11px] text-emerald-100">{stringify(part.output)}</pre>
                                  </div>
                                )}
                                {part.errorText && <div className="mt-2 rounded-lg border border-rose-200 bg-rose-50 p-2 text-rose-700">{part.errorText}</div>}
                              </>
                            )}
                          </div>
                        ))}
                      </div>
                    )}

                    {debugEnabled && eventParts.length > 0 && (
                      <div className="mt-3 space-y-2">
                        {eventParts.map((part, index) => {
                          const eventType = String(part.data?.event_type ?? 'event')
                          const content = String(part.data?.content ?? '')
                          return (
                            <div key={`${message.id}-event-${index}`} className="rounded-xl border border-blue-100 bg-blue-50 p-2 text-left text-xs text-blue-900">
                              <div className="font-semibold">{eventType}</div>
                              {content && <div className="mt-1 whitespace-pre-wrap">{content}</div>}
                            </div>
                          )
                        })}
                      </div>
                    )}

                    {debugEnabled && (
                      <div className="mt-3 border-t border-slate-200 pt-2 text-left text-[11px] font-mono text-slate-500">
                        {message.role} | {message.id}
                        {message.metadata && (
                          <pre className="mt-2 overflow-x-auto rounded-lg bg-slate-100 p-2 text-[11px] text-slate-700">{stringify(message.metadata)}</pre>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}

            {error && <div className="rounded-[20px] border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{t('error')}: {error.message}</div>}
            <div ref={messagesEndRef} />
          </div>
        </div>

        {debugEnabled && (
        <aside className="space-y-3" data-testid="chat-debug-panel">
          <div className="dashboard-muted-card rounded-[24px] p-4">
            <div className="mb-2 text-sm font-semibold text-slate-900">{t('debug')}</div>
            <p className="text-sm text-slate-600">{t('chatDebugHint')}</p>
          </div>

          <div className="dashboard-muted-card rounded-[24px] p-4" data-testid="chat-debug-summary">
            <div className="mb-3 text-sm font-semibold text-slate-900">{t('chatProtocolSummary')}</div>
            <div className="space-y-2 text-xs text-slate-600">
              <div>{t('chatMessagesCount', { count: messages.length })}</div>
              <div>{t('chatPartsCount', { count: totalParts })}</div>
              <div>{t('chatEventsCount', { count: agentEvents.length })}</div>
              <div>{t('chatCurrentStatus', { status })}</div>
            </div>
          </div>

          <div className="dashboard-muted-card rounded-[24px] p-4" data-testid="chat-last-message">
            <div className="mb-3 text-sm font-semibold text-slate-900">{t('chatLastAssistantMessage')}</div>
            {lastAssistant ? (
              <pre className="max-h-64 overflow-auto rounded-xl bg-slate-950/95 p-3 text-[11px] text-sky-100">
                {stringify({
                  id: lastAssistant.id,
                  metadata: lastAssistant.metadata ?? null,
                  parts: lastAssistant.parts ?? [],
                })}
              </pre>
            ) : (
              <div className="text-sm text-slate-500">{t('chatNoStructuredData')}</div>
            )}
          </div>
        </aside>
        )}
      </div>

      <form onSubmit={handleSend} className="flex gap-2" data-testid="chat-form">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder={chatMode === 'agent' ? t('chatPromptAgent') : t('typeMessage')}
          className="dashboard-input flex-1"
          disabled={isStreaming}
          data-testid="chat-input"
        />
        <button
          type="submit"
          disabled={isStreaming || !input.trim() || (chatMode === 'agent' && !selectedAgent)}
          className="dashboard-button px-6 py-2 disabled:cursor-not-allowed"
          data-testid="chat-send"
        >
          {isStreaming ? t('sending') : t('sendMessage')}
        </button>
      </form>
    </div>
  )
}
