import { ChangeEvent, FormEvent, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useCreateAgent, useCreateSquad, useAgents, useOpsLogs, useSquads, useStatus } from '../hooks/useApi'
import type { AgentModel, CreateAgentRequest, CreateSquadRequest, OpsLogEntry } from '../lib/api'
import { MultiAgentChatPanel } from '../components/MultiAgentChatPanel'

const capabilityMeta = [
  { key: 'enable_rag', labelKey: 'rag', color: 'bg-emerald-500' },
  { key: 'enable_memory', labelKey: 'memoryNav', color: 'bg-amber-500' },
  { key: 'enable_ptc', label: 'PTC', color: 'bg-sky-500' },
  { key: 'enable_mcp', label: 'MCP', color: 'bg-fuchsia-500' },
] as const

function cn(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(' ')
}

function timeAgo(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string) {
  const diff = Math.max(0, Date.now() - timestamp)
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return t('justNow')
  if (minutes < 60) return t('minutesAgo', { count: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return t('hoursAgo', { count: hours })
  return t('daysAgo', { count: Math.floor(hours / 24) })
}

function formatDate(input: string | undefined, t: (key: string) => string) {
  if (!input) return t('unknown')
  const date = new Date(input)
  if (Number.isNaN(date.getTime())) return input
  return date.toLocaleString()
}

function kindLabel(kind: AgentModel['kind'], t: (key: string) => string) {
  if (kind === 'specialist') return t('kindSpecialist')
  if (kind === 'agent') return t('kindAgent')
  return t('kindCaptain')
}

function countEnabledCapabilities(agent: AgentModel) {
  return capabilityMeta.filter((item) => agent[item.key]).length
}

export function Agent() {
  const { t } = useTranslation()
  const { data: squads = [] } = useSquads()
  const { data: agents = [], isLoading, error } = useAgents()
  const { data: status } = useStatus()
  const { data: activity = [] } = useOpsLogs(20)
  const createAgent = useCreateAgent()
  const createSquad = useCreateSquad()

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [showCreateSquadForm, setShowCreateSquadForm] = useState(false)
  const [expandedAgent, setExpandedAgent] = useState<string | null>(null)
  const [squadForm, setSquadForm] = useState<CreateSquadRequest>({ name: '', description: '' })
  const [createForm, setCreateForm] = useState<CreateAgentRequest>({
    kind: 'specialist',
    squad_id: '',
    name: '',
    description: '',
    instructions: '',
    model: '',
    required_llm_capability: 0,
    enable_rag: true,
    enable_memory: false,
    enable_ptc: false,
    enable_mcp: true,
    mcp_tools: [],
    skills: [],
  })
  const [rawMCPTools, setRawMCPTools] = useState('')
  const [rawSkills, setRawSkills] = useState('')

  const squadPanels = useMemo(
    () =>
      squads
        .map((squad) => ({
          squad,
          leadAgent: squad.lead_agent ?? squad.captain ?? null,
        }))
        .filter((item) => item.leadAgent != null),
    [squads],
  )

  const metrics = useMemo(() => {
    const totalCapabilities = squadPanels.reduce((sum, item) => sum + countEnabledCapabilities(item.leadAgent!), 0)
    const providers = status?.providers?.filter((provider) => provider.status === 'enabled').length ?? 0

    return [
      { label: t('squads'), value: String(squads.length), subtext: t('agentsTotal', { count: agents.length }) },
      { label: t('capabilitiesArmed'), value: String(totalCapabilities), subtext: t('acrossAllSpecialists') },
      { label: t('providersHealthy'), value: String(providers), subtext: t('providersTracked', { count: status?.providers?.length ?? 0 }) },
      { label: t('knowledgeFootprint'), value: String(status?.rag?.documents ?? 0), subtext: t('chunksIndexed', { count: status?.rag?.chunks ?? 0 }) },
    ]
  }, [agents.length, squadPanels, squads.length, status, t])

  const handleCreateFormField =
    (field: 'name' | 'description' | 'instructions' | 'model') =>
    (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setCreateForm((current) => ({ ...current, [field]: event.target.value }))
    }

  const handleRequiredCapabilityChange = (event: ChangeEvent<HTMLInputElement>) => {
    const value = Number(event.target.value)
    setCreateForm((current) => ({
      ...current,
      required_llm_capability: Number.isNaN(value) ? 0 : value,
    }))
  }

  const handleCapabilityToggle =
    (field: (typeof capabilityMeta)[number]['key']) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      setCreateForm((current) => ({ ...current, [field]: event.target.checked }))
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
      setCreateForm({
        kind: 'specialist',
        squad_id: '',
        name: '',
        description: '',
        instructions: '',
        model: '',
        required_llm_capability: 0,
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
      setExpandedAgent(agent.name)
    } catch (mutationError) {
      console.error(mutationError)
    }
  }

  const handleCreateSquad = async (event: FormEvent) => {
    event.preventDefault()
    try {
      await createSquad.mutateAsync({
        name: squadForm.name.trim(),
        description: squadForm.description.trim(),
      })
      setSquadForm({ name: '', description: '' })
      setShowCreateSquadForm(false)
    } catch (mutationError) {
      console.error(mutationError)
    }
  }

  return (
    <div className="space-y-8" data-testid="page-agent">
      <section className="dashboard-hero" data-testid="agent-metrics">
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {metrics.map((metric) => (
            <div key={metric.label} className="glass-panel rounded-[28px] p-5">
              <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{metric.label}</p>
              <p className="mt-3 text-4xl font-semibold text-slate-900">{metric.value}</p>
              <p className="mt-2 text-sm text-slate-600">{metric.subtext}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('squads')}</p>
            <h2 className="mt-2 text-3xl font-semibold text-slate-900">{t('agents')}</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => setShowCreateSquadForm((value) => !value)}
              className="dashboard-secondary-button px-4 py-2 text-sm"
              data-testid="squad-toggle-create"
            >
              {showCreateSquadForm ? t('close') : t('newSquad')}
            </button>
            <button
              type="button"
              onClick={() => setShowCreateForm((value) => !value)}
              className="dashboard-secondary-button px-4 py-2 text-sm"
              data-testid="agent-toggle-create"
            >
              {showCreateForm ? t('close') : t('newAgent')}
            </button>
          </div>
        </div>

        {showCreateSquadForm && (
          <form onSubmit={handleCreateSquad} className="glass-panel space-y-3 rounded-[28px] p-5" data-testid="squad-create-form">
            <input
              value={squadForm.name}
              onChange={(event) => setSquadForm((current) => ({ ...current, name: event.target.value }))}
              placeholder={t('squadNamePlaceholder')}
              className="dashboard-input"
              required
            />
            <input
              value={squadForm.description}
              onChange={(event) => setSquadForm((current) => ({ ...current, description: event.target.value }))}
              placeholder={t('squadDescriptionPlaceholder')}
              className="dashboard-input"
              required
            />
            <button
              type="submit"
              disabled={createSquad.isPending}
              className="dashboard-button w-full justify-center"
              data-testid="squad-create-submit"
            >
              {createSquad.isPending ? t('creating') : t('createSquad')}
            </button>
          </form>
        )}

        {showCreateForm && (
          <form onSubmit={handleCreateAgent} className="glass-panel space-y-3 rounded-[28px] p-5" data-testid="agent-create-form">
            <select
              value={createForm.squad_id}
              onChange={(event) => setCreateForm((current) => ({ ...current, squad_id: event.target.value }))}
              className="dashboard-input"
            >
              <option value="">{t('defaultSquadOption')}</option>
              {squads.map((squad) => (
                <option key={squad.id} value={squad.id}>
                  {squad.name}
                </option>
              ))}
            </select>
            <select
              value={createForm.kind}
              onChange={(event) => setCreateForm((current) => ({ ...current, kind: event.target.value as CreateAgentRequest['kind'] }))}
              className="dashboard-input"
            >
              <option value="agent">{t('kindAgent')}</option>
              <option value="specialist">{t('kindSpecialist')}</option>
              <option value="captain">{t('kindCaptain')}</option>
            </select>
            <input
              value={createForm.name}
              onChange={handleCreateFormField('name')}
              placeholder={t('agentNamePlaceholder')}
              className="dashboard-input"
              required
            />
            <input
              value={createForm.description}
              onChange={handleCreateFormField('description')}
              placeholder={t('oneLineMission')}
              className="dashboard-input"
              required
            />
            <textarea
              value={createForm.instructions}
              onChange={handleCreateFormField('instructions')}
              placeholder={t('systemInstructions')}
              rows={4}
              className="dashboard-input resize-none"
              required
            />
            <input
              value={createForm.model}
              onChange={handleCreateFormField('model')}
              placeholder={t('preferredProviderOrModel')}
              className="dashboard-input"
            />
            <input
              type="number"
              min={0}
              max={5}
              value={createForm.required_llm_capability ?? 0}
              onChange={handleRequiredCapabilityChange}
              placeholder={t('requiredLlmCapability')}
              className="dashboard-input"
            />
            <textarea
              value={rawMCPTools}
              onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setRawMCPTools(event.target.value)}
              placeholder={t('mcpToolsAllowlist')}
              rows={3}
              className="dashboard-input resize-none"
            />
            <textarea
              value={rawSkills}
              onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setRawSkills(event.target.value)}
              placeholder={t('skillIdsAllowlist')}
              rows={3}
              className="dashboard-input resize-none"
            />
            <div className="grid grid-cols-2 gap-2 text-sm text-slate-700">
              {capabilityMeta.map((capability) => (
                <label key={capability.key} className="flex items-center gap-2 rounded-2xl border border-sky-100 bg-sky-50/50 px-3 py-2">
                  <input
                    type="checkbox"
                    checked={Boolean(createForm[capability.key])}
                    onChange={handleCapabilityToggle(capability.key)}
                  />
                  {'labelKey' in capability ? t(capability.labelKey) : capability.label}
                </label>
              ))}
            </div>
            <button
              type="submit"
              disabled={createAgent.isPending}
              className="dashboard-button w-full justify-center"
              data-testid="agent-create-submit"
            >
              {createAgent.isPending ? t('creating') : t('createSpecialist')}
            </button>
          </form>
        )}

        {isLoading && <div className="glass-panel rounded-[28px] p-5 text-sm text-slate-500">{t('loadingAgents')}</div>}
        {error && <div className="glass-panel rounded-[28px] border border-rose-200 bg-rose-50 p-5 text-sm text-rose-700">{error.message}</div>}

        {!isLoading && squadPanels.length === 0 && (
          <div className="glass-panel rounded-[28px] border border-dashed border-sky-100 bg-sky-50/60 p-6 text-sm text-slate-500">
            {t('noAgentsRegistered')}
          </div>
        )}

        <section className="grid gap-4 xl:grid-cols-2" data-testid="squad-list">
          {squads.map((squad) => (
            <article key={squad.id} className="glass-panel rounded-[28px] p-5">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('squads')}</p>
                  <h3 className="mt-2 text-2xl font-semibold text-slate-900">{squad.name}</h3>
                  <p className="mt-2 text-sm text-slate-600">{squad.description}</p>
                </div>
                <div className="text-right text-sm text-slate-500">
                  <div>{t('captainLabel')}: {squad.lead_agent?.name ?? squad.captain?.name ?? t('unknown')}</div>
                  <div>{t('membersCount', { count: squad.members.length })}</div>
                </div>
              </div>
            </article>
          ))}
        </section>

        <div className="grid gap-6 xl:grid-cols-2" data-testid="agent-captain-panels">
          {squadPanels.map(({ squad, leadAgent }) => (
            <div key={squad.id} className="space-y-4">
              <section className="glass-panel rounded-[32px] p-6" data-testid={`captain-card-${squad.id}`}>
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="text-2xl font-semibold text-slate-900">{squad.name}</h3>
                      <span className="rounded-full bg-blue-100 px-3 py-1 text-xs text-blue-800">{kindLabel(leadAgent!.kind, t)}</span>
                    </div>
                    <p className="mt-2 text-sm text-slate-600">{squad.description}</p>
                  </div>
                </div>

                <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                  <div className="dashboard-muted-card rounded-[22px] p-4">
                    <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('captainLabel')}</p>
                    <p className="mt-2 text-sm text-slate-900">{leadAgent!.name}</p>
                  </div>
                  <div className="dashboard-muted-card rounded-[22px] p-4">
                    <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('model')}</p>
                    <p className="mt-2 text-sm text-slate-900">{leadAgent!.model || t('defaultPool')}</p>
                  </div>
                  <div className="dashboard-muted-card rounded-[22px] p-4">
                    <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('created')}</p>
                    <p className="mt-2 text-sm text-slate-900">{formatDate(squad.created_at, t)}</p>
                  </div>
                  <div className="dashboard-muted-card rounded-[22px] p-4">
                    <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('members')}</p>
                    <p className="mt-2 text-sm text-slate-900">{t('membersCount', { count: squad.members.length })}</p>
                  </div>
                </div>
              </section>

              <MultiAgentChatPanel squad={squad} leadAgent={leadAgent!} />
            </div>
          ))}
        </div>

        <section className="glass-panel rounded-[32px] p-6" data-testid="agent-registry">
          <div>
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('members')}</p>
            <h3 className="mt-2 text-2xl font-semibold text-slate-900">{t('registry')}</h3>
          </div>

          <div className="mt-5 space-y-3" data-testid="agent-list">
            {!isLoading && !agents.length && (
              <div className="rounded-[24px] border border-dashed border-sky-100 bg-sky-50/60 p-5 text-sm text-slate-500">
                {t('noAgentsRegistered')}
              </div>
            )}
            {agents.map((agent) => {
              const isExpanded = expandedAgent === agent.name
              return (
                <article key={agent.id} className="rounded-[24px] border border-sky-100 bg-white">
                  <button
                    type="button"
                    onClick={() => setExpandedAgent((current) => (current === agent.name ? null : agent.name))}
                    className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left"
                    data-testid={`agent-row-${agent.name}`}
                  >
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-lg font-medium text-slate-900">{agent.name}</span>
                        <span className="rounded-full bg-sky-100 px-2.5 py-1 text-xs text-sky-800">{kindLabel(agent.kind, t)}</span>
                      </div>
                      <p className="mt-1 text-sm text-slate-600">{agent.description}</p>
                    </div>
                    <span className="text-sm text-slate-400">{isExpanded ? '−' : '+'}</span>
                  </button>

                  {isExpanded && (
                    <div className="border-t border-sky-100 px-5 py-4">
                      <p className="text-sm leading-7 text-slate-600">{agent.instructions}</p>
                      <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                        <div className="dashboard-muted-card rounded-[20px] p-3">
                          <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('model')}</p>
                          <p className="mt-2 text-sm text-slate-900">{agent.model || t('defaultPool')}</p>
                        </div>
                        <div className="dashboard-muted-card rounded-[20px] p-3">
                          <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('created')}</p>
                          <p className="mt-2 text-sm text-slate-900">{formatDate(agent.created_at, t)}</p>
                        </div>
                        <div className="dashboard-muted-card rounded-[20px] p-3">
                          <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('mcpAllowlist')}</p>
                          <p className="mt-2 text-sm text-slate-900">{t('toolCount', { count: agent.mcp_tools?.length || 0 })}</p>
                        </div>
                        <div className="dashboard-muted-card rounded-[20px] p-3">
                          <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('skillAllowlist')}</p>
                          <p className="mt-2 text-sm text-slate-900">{t('skillCountText', { count: agent.skills?.length || 0 })}</p>
                        </div>
                        <div className="dashboard-muted-card rounded-[20px] p-3">
                          <p className="text-xs uppercase tracking-[0.24em] text-slate-500">{t('squads')}</p>
                          <p className="mt-2 text-sm text-slate-900">
                            {agent.squads?.length
                              ? agent.squads
                                  .map((membership) => squads.find((squad) => squad.id === membership.squad_id)?.name || membership.squad_id)
                                  .join(', ')
                              : t('unknown')}
                          </p>
                        </div>
                      </div>
                    </div>
                  )}
                </article>
              )
            })}
          </div>
        </section>

        <section className="glass-panel rounded-[32px] p-6" data-testid="agent-ops-log">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('recentActivity')}</p>
              <h3 className="mt-2 text-2xl font-semibold text-slate-900">{t('opsLog')}</h3>
            </div>
            <p className="text-sm text-slate-500">{t('uiTriggeredActions')}</p>
          </div>

          <div className="mt-5 space-y-3">
            {!activity.length && (
              <div className="rounded-[24px] border border-dashed border-sky-100 bg-sky-50/60 p-5 text-sm text-slate-500">
                {t('noUiActivityYet')}
              </div>
            )}
            {activity.map((item: OpsLogEntry) => (
              <article key={item.id} className="rounded-[24px] border border-sky-100 bg-white p-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={cn(
                          'rounded-full px-2.5 py-1 text-xs',
                          item.status === 'success'
                            ? 'bg-emerald-50 text-emerald-700'
                            : item.status === 'error'
                              ? 'bg-rose-50 text-rose-700'
                              : 'bg-sky-50 text-sky-700',
                        )}
                      >
                        {item.kind}
                      </span>
                      <span className="text-xs uppercase tracking-[0.24em] text-slate-500">{item.agent_name}</span>
                    </div>
                    <h4 className="mt-3 text-lg font-medium text-slate-900">{item.title}</h4>
                    <p className="mt-2 whitespace-pre-wrap text-sm leading-7 text-slate-600">{item.detail}</p>
                  </div>
                  <div className="text-right text-xs text-slate-500">
                    <div>{timeAgo(new Date(item.timestamp).getTime(), t)}</div>
                    {item.duration_ms ? <div className="mt-1">{item.duration_ms} ms</div> : null}
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>
      </section>
    </div>
  )
}
