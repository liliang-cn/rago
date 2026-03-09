import { FormEvent, useEffect, useMemo, useState } from 'react'
import { useCreateAgent, useDispatchAgentTask, useAgents, useStartAgent, useStatus, useStopAgent } from '../hooks/useApi'
import type { AgentModel, CreateAgentRequest } from '../lib/api'

type ActivityItem = {
  id: string
  agentName: string
  kind: 'dispatch' | 'lifecycle' | 'create'
  status: 'success' | 'error' | 'info'
  title: string
  detail: string
  timestamp: number
  durationMs?: number
}

const capabilityMeta = [
  { key: 'enable_rag', label: 'RAG', color: 'bg-emerald-500' },
  { key: 'enable_memory', label: 'Memory', color: 'bg-amber-500' },
  { key: 'enable_ptc', label: 'PTC', color: 'bg-sky-500' },
  { key: 'enable_mcp', label: 'MCP', color: 'bg-fuchsia-500' },
] as const

function cn(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(' ')
}

function timeAgo(timestamp: number) {
  const diff = Math.max(0, Date.now() - timestamp)
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

function formatDate(input?: string) {
  if (!input) return 'Unknown'
  const date = new Date(input)
  if (Number.isNaN(date.getTime())) return input
  return date.toLocaleString()
}

function statusTone(status: AgentModel['status']) {
  switch (status) {
    case 'running':
      return 'text-emerald-200 bg-emerald-500/15 ring-emerald-500/30'
    case 'error':
      return 'text-rose-200 bg-rose-500/15 ring-rose-500/30'
    default:
      return 'text-stone-200 bg-stone-500/15 ring-stone-500/30'
  }
}

function countEnabledCapabilities(agent: AgentModel) {
  return capabilityMeta.filter((item) => agent[item.key]).length
}

export function Agent() {
  const { data: agents = [], isLoading, error } = useAgents()
  const { data: status } = useStatus()
  const createAgent = useCreateAgent()
  const startAgent = useStartAgent()
  const stopAgent = useStopAgent()
  const dispatchTask = useDispatchAgentTask()

  const [selectedName, setSelectedName] = useState('')
  const [instruction, setInstruction] = useState('')
  const [activity, setActivity] = useState<ActivityItem[]>([])
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [createForm, setCreateForm] = useState<CreateAgentRequest>({
    name: '',
    description: '',
    instructions: '',
    model: '',
    enable_rag: true,
    enable_memory: false,
    enable_ptc: false,
    enable_mcp: true,
    mcp_tools: [],
    skills: [],
  })
  const [rawMCPTools, setRawMCPTools] = useState('')
  const [rawSkills, setRawSkills] = useState('')

  useEffect(() => {
    if (!agents.length) {
      setSelectedName('')
      return
    }
    if (!selectedName || !agents.some((agent) => agent.name === selectedName)) {
      setSelectedName(agents[0].name)
    }
  }, [agents, selectedName])

  const selectedAgent = useMemo(
    () => agents.find((agent) => agent.name === selectedName) ?? null,
    [agents, selectedName],
  )

  const metrics = useMemo(() => {
    const runningAgents = agents.filter((agent) => agent.status === 'running').length
    const totalCapabilities = agents.reduce((sum, agent) => sum + countEnabledCapabilities(agent), 0)
    const providers = status?.providers?.filter((provider) => provider.status === 'enabled').length ?? 0

    return [
      { label: 'Agents online', value: String(runningAgents), subtext: `${agents.length} total agents` },
      { label: 'Capabilities armed', value: String(totalCapabilities), subtext: 'Across all specialists' },
      { label: 'Providers healthy', value: String(providers), subtext: `${status?.providers?.length ?? 0} providers tracked` },
      { label: 'Knowledge footprint', value: String(status?.rag?.documents ?? 0), subtext: `${status?.rag?.chunks ?? 0} chunks indexed` },
    ]
  }, [agents, status])

  const appendActivity = (item: Omit<ActivityItem, 'id' | 'timestamp'>) => {
    setActivity((current) => [
      {
        ...item,
        id: crypto.randomUUID(),
        timestamp: Date.now(),
      },
      ...current,
    ].slice(0, 12))
  }

  const handleCreateAgent = async (event: FormEvent) => {
    event.preventDefault()

    const payload: CreateAgentRequest = {
      ...createForm,
      mcp_tools: rawMCPTools.split('\n').map((item) => item.trim()).filter(Boolean),
      skills: rawSkills.split('\n').map((item) => item.trim()).filter(Boolean),
    }

    try {
      const agent = await createAgent.mutateAsync(payload)
      appendActivity({
        agentName: agent.name,
        kind: 'create',
        status: 'success',
        title: `Created ${agent.name}`,
        detail: agent.description || 'New specialist registered.',
      })
      setCreateForm({
        name: '',
        description: '',
        instructions: '',
        model: '',
        enable_rag: true,
        enable_memory: false,
        enable_ptc: false,
        enable_mcp: true,
        mcp_tools: [],
        skills: [],
      })
      setRawMCPTools('')
      setRawSkills('')
      setShowCreateForm(false)
      setSelectedName(agent.name)
    } catch (mutationError) {
      appendActivity({
        agentName: createForm.name || 'New agent',
        kind: 'create',
        status: 'error',
        title: 'Create agent failed',
        detail: mutationError instanceof Error ? mutationError.message : 'Unknown error',
      })
    }
  }

  const handleLifecycle = async (mode: 'start' | 'stop', agent: AgentModel) => {
    try {
      if (mode === 'start') {
        await startAgent.mutateAsync(agent.name)
      } else {
        await stopAgent.mutateAsync(agent.name)
      }
      appendActivity({
        agentName: agent.name,
        kind: 'lifecycle',
        status: 'info',
        title: `${mode === 'start' ? 'Started' : 'Stopped'} ${agent.name}`,
        detail: mode === 'start' ? 'Agent is ready to receive delegated tasks.' : 'Agent has been paused.',
      })
    } catch (mutationError) {
      appendActivity({
        agentName: agent.name,
        kind: 'lifecycle',
        status: 'error',
        title: `${mode === 'start' ? 'Start' : 'Stop'} failed`,
        detail: mutationError instanceof Error ? mutationError.message : 'Unknown error',
      })
    }
  }

  const handleDispatch = async (event: FormEvent) => {
    event.preventDefault()
    if (!selectedAgent || !instruction.trim()) return

    try {
      const result = await dispatchTask.mutateAsync({
        name: selectedAgent.name,
        instruction,
      })
      appendActivity({
        agentName: selectedAgent.name,
        kind: 'dispatch',
        status: 'success',
        title: `Task completed by ${selectedAgent.name}`,
        detail: result.response,
        durationMs: result.duration_ms,
      })
      setInstruction('')
    } catch (mutationError) {
      appendActivity({
        agentName: selectedAgent.name,
        kind: 'dispatch',
        status: 'error',
        title: `Dispatch failed for ${selectedAgent.name}`,
        detail: mutationError instanceof Error ? mutationError.message : 'Unknown error',
      })
    }
  }

  return (
    <div className="space-y-8">
      <section className="dashboard-hero">
        <div className="max-w-3xl">
          <p className="text-xs font-semibold uppercase tracking-[0.35em] text-[#ffcf99]">Multi-agent command deck</p>
          <h2 className="mt-3 text-4xl font-semibold tracking-tight text-white sm:text-5xl">
            Run your specialists like an ops room, not a chat demo.
          </h2>
          <p className="mt-4 max-w-2xl text-sm leading-7 text-[#f6e8d5] sm:text-base">
            Monitor agent health, launch or stop specialists, delegate tasks directly, and mint new agents from the web UI without dropping to the CLI.
          </p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {metrics.map((metric) => (
            <div key={metric.label} className="glass-panel rounded-[28px] p-5">
              <p className="text-xs uppercase tracking-[0.24em] text-[#8ea0b5]">{metric.label}</p>
              <p className="mt-3 text-4xl font-semibold text-white">{metric.value}</p>
              <p className="mt-2 text-sm text-[#a7b7c9]">{metric.subtext}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="glass-panel rounded-[32px] p-5">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Registry</p>
              <h3 className="mt-2 text-2xl font-semibold text-white">Agents</h3>
            </div>
            <button
              type="button"
              onClick={() => setShowCreateForm((value) => !value)}
              className="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm text-white transition hover:bg-white/10"
            >
              {showCreateForm ? 'Close' : 'New agent'}
            </button>
          </div>

          {showCreateForm && (
            <form onSubmit={handleCreateAgent} className="mt-5 space-y-3 rounded-[24px] border border-white/10 bg-black/10 p-4">
              <input
                value={createForm.name}
                onChange={(event) => setCreateForm((current) => ({ ...current, name: event.target.value }))}
                placeholder="Agent name"
                className="dashboard-input"
                required
              />
              <input
                value={createForm.description}
                onChange={(event) => setCreateForm((current) => ({ ...current, description: event.target.value }))}
                placeholder="One-line mission"
                className="dashboard-input"
                required
              />
              <textarea
                value={createForm.instructions}
                onChange={(event) => setCreateForm((current) => ({ ...current, instructions: event.target.value }))}
                placeholder="System instructions"
                rows={4}
                className="dashboard-input resize-none"
                required
              />
              <input
                value={createForm.model}
                onChange={(event) => setCreateForm((current) => ({ ...current, model: event.target.value }))}
                placeholder="Model override (optional)"
                className="dashboard-input"
              />
              <textarea
                value={rawMCPTools}
                onChange={(event) => setRawMCPTools(event.target.value)}
                placeholder="MCP tools allowlist, one per line"
                rows={3}
                className="dashboard-input resize-none"
              />
              <textarea
                value={rawSkills}
                onChange={(event) => setRawSkills(event.target.value)}
                placeholder="Skill IDs allowlist, one per line"
                rows={3}
                className="dashboard-input resize-none"
              />
              <div className="grid grid-cols-2 gap-2 text-sm text-[#d7e1ec]">
                {capabilityMeta.map((capability) => (
                  <label key={capability.key} className="flex items-center gap-2 rounded-2xl border border-white/8 bg-white/5 px-3 py-2">
                    <input
                      type="checkbox"
                      checked={Boolean(createForm[capability.key])}
                      onChange={(event) => setCreateForm((current) => ({ ...current, [capability.key]: event.target.checked }))}
                    />
                    {capability.label}
                  </label>
                ))}
              </div>
              <button
                type="submit"
                disabled={createAgent.isPending}
                className="dashboard-button w-full justify-center"
              >
                {createAgent.isPending ? 'Creating...' : 'Create specialist'}
              </button>
            </form>
          )}

          <div className="mt-5 space-y-3">
            {isLoading && <div className="rounded-[24px] border border-white/10 bg-white/5 p-4 text-sm text-[#a7b7c9]">Loading agents…</div>}
            {error && <div className="rounded-[24px] border border-rose-500/30 bg-rose-500/10 p-4 text-sm text-rose-100">{error.message}</div>}
            {!isLoading && !agents.length && (
              <div className="rounded-[24px] border border-dashed border-white/10 bg-white/5 p-5 text-sm text-[#a7b7c9]">
                No agents registered yet. Create one or seed defaults from the server.
              </div>
            )}
            {agents.map((agent) => (
              <button
                type="button"
                key={agent.id}
                onClick={() => setSelectedName(agent.name)}
                className={cn(
                  'w-full rounded-[24px] border p-4 text-left transition',
                  selectedAgent?.name === agent.name
                    ? 'border-[#ffb15e] bg-[#2d1f17] shadow-[0_12px_32px_rgba(255,177,94,0.18)]'
                    : 'border-white/8 bg-white/[0.03] hover:bg-white/[0.07]',
                )}
              >
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-lg font-medium text-white">{agent.name}</p>
                    <p className="mt-1 text-sm text-[#a7b7c9]">{agent.description}</p>
                  </div>
                  <span className={cn('rounded-full px-3 py-1 text-xs ring-1', statusTone(agent.status))}>{agent.status}</span>
                </div>
                <div className="mt-4 flex flex-wrap gap-2">
                  {capabilityMeta.map((capability) => (
                    <span
                      key={capability.key}
                      className={cn(
                        'rounded-full px-2.5 py-1 text-xs text-white/90',
                        agent[capability.key] ? capability.color : 'bg-white/10 text-[#8290a3]',
                      )}
                    >
                      {capability.label}
                    </span>
                  ))}
                </div>
              </button>
            ))}
          </div>
        </aside>

        <div className="space-y-6">
          <section className="glass-panel rounded-[32px] p-6">
            {selectedAgent ? (
              <>
                <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
                  <div className="max-w-3xl">
                    <div className="flex flex-wrap items-center gap-3">
                      <h3 className="text-3xl font-semibold text-white">{selectedAgent.name}</h3>
                      <span className={cn('rounded-full px-3 py-1 text-xs ring-1', statusTone(selectedAgent.status))}>
                        {selectedAgent.status}
                      </span>
                    </div>
                    <p className="mt-3 text-base leading-7 text-[#d7e1ec]">{selectedAgent.description}</p>
                    <p className="mt-4 text-sm leading-7 text-[#a7b7c9]">{selectedAgent.instructions}</p>
                  </div>

                  <div className="grid gap-3 sm:grid-cols-2">
                    <button
                      type="button"
                      onClick={() => handleLifecycle('start', selectedAgent)}
                      disabled={startAgent.isPending}
                      className="dashboard-button justify-center"
                    >
                      Start agent
                    </button>
                    <button
                      type="button"
                      onClick={() => handleLifecycle('stop', selectedAgent)}
                      disabled={stopAgent.isPending}
                      className="rounded-full border border-white/12 bg-white/5 px-4 py-3 text-sm font-medium text-white transition hover:bg-white/10"
                    >
                      Stop agent
                    </button>
                  </div>
                </div>

                <div className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                  <div className="rounded-[24px] border border-white/8 bg-black/10 p-4">
                    <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Model</p>
                    <p className="mt-3 text-lg text-white">{selectedAgent.model || 'Default pool'}</p>
                  </div>
                  <div className="rounded-[24px] border border-white/8 bg-black/10 p-4">
                    <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Created</p>
                    <p className="mt-3 text-lg text-white">{formatDate(selectedAgent.created_at)}</p>
                  </div>
                  <div className="rounded-[24px] border border-white/8 bg-black/10 p-4">
                    <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">MCP allowlist</p>
                    <p className="mt-3 text-lg text-white">{selectedAgent.mcp_tools?.length || 0} tools</p>
                  </div>
                  <div className="rounded-[24px] border border-white/8 bg-black/10 p-4">
                    <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Skill allowlist</p>
                    <p className="mt-3 text-lg text-white">{selectedAgent.skills?.length || 0} skills</p>
                  </div>
                </div>

                <div className="mt-8 grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_360px]">
                  <form onSubmit={handleDispatch} className="rounded-[28px] border border-white/8 bg-white/[0.03] p-5">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Direct dispatch</p>
                        <h4 className="mt-2 text-xl font-semibold text-white">Send a focused task</h4>
                      </div>
                      {dispatchTask.isPending && <span className="text-sm text-[#ffcf99]">Running…</span>}
                    </div>
                    <textarea
                      value={instruction}
                      onChange={(event) => setInstruction(event.target.value)}
                      placeholder={`Delegate a concrete task to ${selectedAgent.name}...`}
                      rows={8}
                      className="dashboard-input mt-5 resize-none"
                    />
                    <div className="mt-4 flex flex-wrap gap-3">
                      <button
                        type="submit"
                        disabled={dispatchTask.isPending || !instruction.trim()}
                        className="dashboard-button"
                      >
                        Dispatch task
                      </button>
                      <button
                        type="button"
                        onClick={() => setInstruction(`Summarize your remit, available capabilities, and current operating boundaries in 5 bullets.`)}
                        className="rounded-full border border-white/10 bg-white/5 px-4 py-3 text-sm text-[#d7e1ec] transition hover:bg-white/10"
                      >
                        Insert health-check prompt
                      </button>
                    </div>
                  </form>

                  <div className="rounded-[28px] border border-white/8 bg-white/[0.03] p-5">
                    <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Capability matrix</p>
                    <div className="mt-5 grid gap-3">
                      {capabilityMeta.map((capability) => (
                        <div key={capability.key} className="rounded-[22px] border border-white/8 bg-black/10 p-4">
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium text-white">{capability.label}</span>
                            <span className={cn('h-3 w-3 rounded-full', selectedAgent[capability.key] ? capability.color : 'bg-white/10')} />
                          </div>
                          <p className="mt-2 text-sm text-[#a7b7c9]">
                            {selectedAgent[capability.key]
                              ? `${capability.label} is exposed to this specialist.`
                              : `${capability.label} is currently withheld from this specialist.`}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <div className="rounded-[28px] border border-dashed border-white/10 bg-white/[0.03] p-8 text-[#a7b7c9]">
                Select an agent from the registry to inspect and control it.
              </div>
            )}
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Recent activity</p>
                <h3 className="mt-2 text-2xl font-semibold text-white">Ops log</h3>
              </div>
              <p className="text-sm text-[#a7b7c9]">UI-triggered actions and their latest outcomes.</p>
            </div>

            <div className="mt-5 space-y-3">
              {!activity.length && (
                <div className="rounded-[24px] border border-dashed border-white/10 bg-white/[0.03] p-5 text-sm text-[#a7b7c9]">
                  No UI activity yet. Start, stop, create, or dispatch a task to populate the log.
                </div>
              )}
              {activity.map((item) => (
                <article key={item.id} className="rounded-[24px] border border-white/8 bg-black/10 p-4">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <span
                          className={cn(
                            'rounded-full px-2.5 py-1 text-xs',
                            item.status === 'success'
                              ? 'bg-emerald-500/15 text-emerald-200'
                              : item.status === 'error'
                                ? 'bg-rose-500/15 text-rose-200'
                                : 'bg-sky-500/15 text-sky-200',
                          )}
                        >
                          {item.kind}
                        </span>
                        <span className="text-xs uppercase tracking-[0.24em] text-[#8ea0b5]">{item.agentName}</span>
                      </div>
                      <h4 className="mt-3 text-lg font-medium text-white">{item.title}</h4>
                      <p className="mt-2 whitespace-pre-wrap text-sm leading-7 text-[#d7e1ec]">{item.detail}</p>
                    </div>
                    <div className="text-right text-xs text-[#8ea0b5]">
                      <div>{timeAgo(item.timestamp)}</div>
                      {item.durationMs ? <div className="mt-1">{item.durationMs} ms</div> : null}
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </section>
        </div>
      </section>
    </div>
  )
}
