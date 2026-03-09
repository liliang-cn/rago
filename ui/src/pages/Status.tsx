import { useStatus } from '../hooks/useApi'
import { useTranslation } from 'react-i18next'

interface Provider {
  name: string
  status: string
  type: string
  model?: string
  healthy?: boolean
  active_requests?: number
  max_concurrency?: number
  capability?: number
}

function ProviderCard({ provider, statusColor }: { provider: Provider; statusColor: string }) {
  return (
    <div className="dashboard-muted-card rounded-[24px] p-4">
      <div className="flex items-center justify-between mb-2">
        <h4 className="font-medium text-slate-900">
          {provider.name}
        </h4>
        <div className="flex items-center gap-2">
          <div className={`w-3 h-3 rounded-full ${statusColor}`} />
          <span className="text-sm text-slate-500 capitalize">
            {provider.status}
          </span>
        </div>
      </div>
      {provider.model && (
        <p className="text-sm text-slate-500">
          Model: {provider.model}
        </p>
      )}
      {(provider.capability !== undefined || provider.max_concurrency !== undefined || provider.active_requests !== undefined) && (
        <dl className="mt-3 grid grid-cols-3 gap-3 text-xs">
          <div>
            <dt className="text-slate-400">Capability</dt>
            <dd className="mt-1 text-slate-700">{provider.capability ?? '-'}</dd>
          </div>
          <div>
            <dt className="text-slate-400">Max concurrent</dt>
            <dd className="mt-1 text-slate-700">{provider.max_concurrency ?? '-'}</dd>
          </div>
          <div>
            <dt className="text-slate-400">Active</dt>
            <dd className="mt-1 text-slate-700">{provider.active_requests ?? 0}</dd>
          </div>
        </dl>
      )}
    </div>
  )
}

export function Status() {
  const { t } = useTranslation()
  const { data, isLoading, error, refetch } = useStatus()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-[24px] border border-rose-200 bg-rose-50 p-4">
        <p className="text-rose-700">
          Error loading status: {error.message}
        </p>
        <button
          onClick={() => refetch()}
          className="dashboard-button mt-2 px-4 py-2"
        >
          Retry
        </button>
      </div>
    )
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'enabled':
        return 'bg-green-500'
      case 'disabled':
        return 'bg-red-500'
      default:
        return 'bg-gray-500'
    }
  }

  // Separate providers by type
  const llmProviders = data?.providers?.filter((p: Provider) => p.type === 'llm') || []
  const embedProviders = data?.providers?.filter((p: Provider) => p.type === 'embedding') || []
  const otherProviders = data?.providers?.filter((p: Provider) => !['llm', 'embedding'].includes(p.type)) || []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-slate-900">
          System Status
        </h2>
        <button
          onClick={() => refetch()}
          className="dashboard-secondary-button px-4 py-2 text-sm"
        >
          Refresh
        </button>
      </div>

      {data && (
        <>
          {/* Status Overview */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="glass-panel rounded-[28px] p-6">
              <h3 className="text-sm font-medium text-slate-500 mb-1">
                Status
              </h3>
              <p className="text-2xl font-semibold text-slate-900">
                {data.status}
              </p>
            </div>
            <div className="glass-panel rounded-[28px] p-6">
              <h3 className="text-sm font-medium text-slate-500 mb-1">
                Version
              </h3>
              <p className="text-2xl font-semibold text-slate-900">
                {data.version}
              </p>
            </div>
          </div>

          {/* {t('llmProviders')} */}
          {llmProviders.length > 0 && (
            <div>
              <h3 className="text-lg font-medium text-slate-900 mb-4">
                {t('llmProviders')}
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {llmProviders.map((provider: Provider) => (
                  <ProviderCard
                    key={provider.name}
                    provider={provider}
                    statusColor={getStatusColor(provider.status)}
                  />
                ))}
              </div>
            </div>
          )}

          {/* {t('embeddingProviders')} */}
          {embedProviders.length > 0 && (
            <div>
              <h3 className="text-lg font-medium text-slate-900 mb-4">
                {t('embeddingProviders')}
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {embedProviders.map((provider: Provider) => (
                  <ProviderCard
                    key={provider.name}
                    provider={provider}
                    statusColor={getStatusColor(provider.status)}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Other Services */}
          {otherProviders.length > 0 && (
            <div>
              <h3 className="text-lg font-medium text-slate-900 mb-4">
                Services
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {otherProviders.map((provider: Provider) => (
                  <div
                    key={provider.name}
                    className="dashboard-muted-card rounded-[24px] p-4"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-slate-900">
                        {provider.name}
                      </h4>
                      <div className="flex items-center gap-2">
                        <div className={`w-3 h-3 rounded-full ${getStatusColor(provider.status)}`} />
                        <span className="text-sm text-slate-500 capitalize">
                          {provider.status}
                        </span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* {t('mcpServers')} Detail */}
          {data.mcp?.enabled && data.mcp?.server_list && (
            <div>
              <h3 className="text-lg font-medium text-slate-900 mb-4">
                {t('mcpServers')} ({data.mcp.servers} servers, {data.mcp.tools} tools)
              </h3>
              <div className="space-y-3">
                {data.mcp.server_list.map((server: any) => (
                  <div
                    key={server.name}
                    className="dashboard-muted-card rounded-[24px] p-4"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-slate-900">
                        {server.name}
                      </h4>
                      <span className={`px-2 py-1 text-xs rounded ${
                        server.running
                          ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                          : 'bg-gray-100 text-gray-800'
                      }`}>
                        {server.running ? 'Running' : 'Stopped'}
                      </span>
                    </div>
                    <p className="text-sm text-slate-500">
                      Tools: {server.tool_count}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* RAG Stats */}
          {data.rag?.enabled && (
            <div className="glass-panel rounded-[24px] p-4">
              <h3 className="text-lg font-medium text-slate-900 mb-2">
                RAG Database
              </h3>
              <p className="text-sm text-slate-500">
                Documents: {data.rag.documents} | Chunks: {data.rag.chunks}
              </p>
              <p className="text-xs text-slate-400 mt-1">
                {data.rag.db_path}
              </p>
            </div>
          )}

          {/* Memory Stats */}
          {data.memory?.enabled && (
            <div className="glass-panel rounded-[24px] p-4">
              <h3 className="text-lg font-medium text-slate-900 mb-2">
                Memory
              </h3>
              <p className="text-sm text-slate-500">
                Memories: {data.memory.count}
              </p>
            </div>
          )}

          {/* Skills Stats */}
          {data.skills?.enabled && (
            <div className="glass-panel rounded-[24px] p-4">
              <h3 className="text-lg font-medium text-slate-900 mb-2">
                Skills
              </h3>
              <p className="text-sm text-slate-500">
                Loaded: {data.skills.count}
              </p>
            </div>
          )}
        </>
      )}
    </div>
  )
}
