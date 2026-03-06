import { useStatus } from '../hooks/useApi'
import { useTranslation } from 'react-i18next'

interface Provider {
  name: string
  status: string
  type: string
  model?: string
  healthy?: boolean
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
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <p className="text-red-600 dark:text-red-400">
          Error loading status: {error.message}
        </p>
        <button
          onClick={() => refetch()}
          className="mt-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
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
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
          System Status
        </h2>
        <button
          onClick={() => refetch()}
          className="px-4 py-2 text-sm bg-gray-100 dark:bg-gray-800 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
        >
          Refresh
        </button>
      </div>

      {data && (
        <>
          {/* Status Overview */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">
                Status
              </h3>
              <p className="text-2xl font-semibold text-gray-900 dark:text-white">
                {data.status}
              </p>
            </div>
            <div className="p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">
                Version
              </h3>
              <p className="text-2xl font-semibold text-gray-900 dark:text-white">
                {data.version}
              </p>
            </div>
          </div>

          {/* {t('llmProviders')} */}
          {llmProviders.length > 0 && (
            <div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                {t('llmProviders')}
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {llmProviders.map((provider: Provider) => (
                  <div
                    key={provider.name}
                    className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-gray-900 dark:text-white">
                        {provider.name}
                      </h4>
                      <div className="flex items-center gap-2">
                        <div className={`w-3 h-3 rounded-full ${getStatusColor(provider.status)}`} />
                        <span className="text-sm text-gray-500 dark:text-gray-400 capitalize">
                          {provider.status}
                        </span>
                      </div>
                    </div>
                    {provider.model && (
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        Model: {provider.model}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* {t('embeddingProviders')} */}
          {embedProviders.length > 0 && (
            <div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                {t('embeddingProviders')}
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {embedProviders.map((provider: Provider) => (
                  <div
                    key={provider.name}
                    className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-gray-900 dark:text-white">
                        {provider.name}
                      </h4>
                      <div className="flex items-center gap-2">
                        <div className={`w-3 h-3 rounded-full ${getStatusColor(provider.status)}`} />
                        <span className="text-sm text-gray-500 dark:text-gray-400 capitalize">
                          {provider.status}
                        </span>
                      </div>
                    </div>
                    {provider.model && (
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        Model: {provider.model}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Other Services */}
          {otherProviders.length > 0 && (
            <div>
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                Services
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {otherProviders.map((provider: Provider) => (
                  <div
                    key={provider.name}
                    className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-gray-900 dark:text-white">
                        {provider.name}
                      </h4>
                      <div className="flex items-center gap-2">
                        <div className={`w-3 h-3 rounded-full ${getStatusColor(provider.status)}`} />
                        <span className="text-sm text-gray-500 dark:text-gray-400 capitalize">
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
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                {t('mcpServers')} ({data.mcp.servers} servers, {data.mcp.tools} tools)
              </h3>
              <div className="space-y-3">
                {data.mcp.server_list.map((server: any) => (
                  <div
                    key={server.name}
                    className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-gray-900 dark:text-white">
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
                    <p className="text-sm text-gray-500 dark:text-gray-400">
                      Tools: {server.tool_count}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* RAG Stats */}
          {data.rag?.enabled && (
            <div className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                RAG Database
              </h3>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Documents: {data.rag.documents} | Chunks: {data.rag.chunks}
              </p>
              <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                {data.rag.db_path}
              </p>
            </div>
          )}

          {/* Memory Stats */}
          {data.memory?.enabled && (
            <div className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                Memory
              </h3>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Memories: {data.memory.count}
              </p>
            </div>
          )}

          {/* Skills Stats */}
          {data.skills?.enabled && (
            <div className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
              <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                Skills
              </h3>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Loaded: {data.skills.count}
              </p>
            </div>
          )}
        </>
      )}
    </div>
  )
}
