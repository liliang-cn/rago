import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  useMCPServers,
  useMCPTools,
  useAddMCPServer,
  useCallMCPTool,
} from '../hooks/useApi'
import type { MCPServer, MCPTool, AddMCPServerRequest, CallToolRequest } from '../lib/api'

export function MCP() {
  const { t } = useTranslation()
  const [showAddForm, setShowAddForm] = useState(false)
  const [selectedTool, setSelectedTool] = useState<MCPTool | null>(null)
  const [toolResult, setToolResult] = useState<string | null>(null)
  const { data: servers, isLoading: serversLoading, refetch: refetchServers } = useMCPServers()
  const { data: tools, isLoading: toolsLoading, refetch: refetchTools } = useMCPTools()
  const addServerMutation = useAddMCPServer()
  const callToolMutation = useCallMCPTool()

  const handleAddServer = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const data: AddMCPServerRequest = {
      name: formData.get('name') as string,
      command: formData.get('command') as string || undefined,
      args: formData.get('args') ? (formData.get('args') as string).split(' ') : undefined,
      type: (formData.get('type') as string) || 'stdio',
      url: formData.get('url') as string || undefined,
    }
    await addServerMutation.mutateAsync(data)
    setShowAddForm(false)
  }

  const handleCallTool = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const data: CallToolRequest = {
      tool_name: formData.get('tool_name') as string,
      arguments: JSON.parse(formData.get('arguments') as string || '{}'),
    }
    try {
      const result = await callToolMutation.mutateAsync(data)
      setToolResult(JSON.stringify(result, null, 2))
    } catch (err) {
      setToolResult(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  if (serversLoading || toolsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6" data-testid="page-mcp">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-slate-900">{t('mcp')}</h2>
        <div className="flex gap-2">
          <button
            onClick={() => {
              refetchServers()
              refetchTools()
            }}
            className="dashboard-secondary-button px-4 py-2 text-sm"
            data-testid="mcp-refresh"
          >
            {t('refresh')}
          </button>
          <button
            onClick={() => setShowAddForm(true)}
            className="dashboard-button px-4 py-2 text-sm"
            data-testid="mcp-add-server"
          >
            {t('addServerButton')}
          </button>
        </div>
      </div>

      {/* Add Server Form */}
      {showAddForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-sky-950/10 backdrop-blur-sm">
          <div className="glass-panel w-full max-w-md rounded-[28px] p-6 mx-4" data-testid="mcp-add-server-modal">
            <h3 className="text-lg font-semibold text-slate-900 mb-4">{t('addMcpServerTitle')}</h3>
            <form onSubmit={handleAddServer} className="space-y-4" data-testid="mcp-add-server-form">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  {t('skillNameRequired')}
                </label>
                <input
                  name="name"
                  required
                  className="dashboard-input"
                  placeholder={t('serverNameExample')}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  {t('serverTypeLabel')}
                </label>
                <select
                  name="type"
                  className="dashboard-input"
                >
                  <option value="stdio">Stdio</option>
                  <option value="http">HTTP</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  {t('commandForStdio')}
                </label>
                <input
                  name="command"
                  className="dashboard-input"
                  placeholder={t('serverCommandExample')}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  {t('argsSpaceSeparated')}
                </label>
                <input
                  name="args"
                  className="dashboard-input"
                  placeholder={t('serverArgsExample')}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  {t('urlForHttp')}
                </label>
                <input
                  name="url"
                  className="dashboard-input"
                  placeholder={t('serverUrlExample')}
                />
              </div>
              <div className="flex justify-end gap-2 pt-4">
                <button
                  type="button"
                  onClick={() => setShowAddForm(false)}
                  className="dashboard-secondary-button px-4 py-2"
                >
                  {t('cancel')}
                </button>
                <button
                  type="submit"
                  className="dashboard-button px-4 py-2"
                >
                  {t('addServerButton')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Servers List */}
        <div>
          <h3 className="text-lg font-medium text-slate-900 mb-4">{t('servers')}</h3>
          <div className="space-y-3" data-testid="mcp-server-list">
            {servers && servers.length > 0 ? (
              servers.map((server) => (
                <div
                  key={server.name}
                  className="dashboard-muted-card rounded-[24px] p-4"
                >
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="font-medium text-slate-900">
                      {server.name}
                    </h4>
                    <span
                      className={`px-2 py-1 text-xs rounded ${
                        server.running
                          ? 'bg-emerald-100 text-emerald-700'
                          : 'bg-slate-100 text-slate-700'
                      }`}
                    >
                      {server.running ? t('running') : t('stopped')}
                    </span>
                  </div>
                  <p className="text-sm text-slate-600 mb-2">
                    {server.description || server.command || t('noCommand')}
                  </p>
                  <p className="text-sm text-slate-500">
                    {t('toolsSummary', { count: server.tool_count })}
                  </p>
                </div>
              ))
            ) : (
              <div className="dashboard-muted-card rounded-[24px] p-4 text-center text-slate-500">
                {t('noServers')}
              </div>
            )}
          </div>
        </div>

        {/* Tools List */}
        <div>
          <h3 className="text-lg font-medium text-slate-900 mb-4">{t('availableTools')}</h3>
          <div className="space-y-3 max-h-96 overflow-y-auto" data-testid="mcp-tool-list">
            {tools && tools.length > 0 ? (
              tools.map((tool) => (
                <div
                  key={`${tool.server_name}-${tool.name}`}
                  className="dashboard-muted-card cursor-pointer rounded-[24px] p-4 transition-colors hover:border-sky-300 hover:bg-sky-50/50"
                  onClick={() => setSelectedTool(tool)}
                  data-testid={`mcp-tool-${tool.server_name}-${tool.name}`}
                >
                  <div className="flex items-start justify-between mb-1">
                    <h4 className="font-medium text-slate-900 text-sm">
                      {tool.name}
                    </h4>
                    <span className="text-xs text-slate-500">
                      {tool.server_name}
                    </span>
                  </div>
                  <p className="text-xs text-slate-600 line-clamp-2">
                    {tool.description}
                  </p>
                </div>
              ))
            ) : (
              <div className="dashboard-muted-card rounded-[24px] p-4 text-center text-slate-500">
                {t('noToolsAvailable')}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Tool Test Panel */}
      {selectedTool && (
        <div className="glass-panel mt-6 rounded-[28px] p-4" data-testid="mcp-tool-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-slate-900">
              {t('testToolTitle', { name: selectedTool.name })}
            </h3>
            <button
              onClick={() => {
                setSelectedTool(null)
                setToolResult(null)
              }}
              className="text-slate-500 hover:text-slate-700"
            >
              {t('closeButton')}
            </button>
          </div>
          <form onSubmit={handleCallTool} className="space-y-4" data-testid="mcp-tool-form">
            <input type="hidden" name="tool_name" value={selectedTool.name} />
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                {t('toolArgs')}
              </label>
              <textarea
                name="arguments"
                rows={4}
                className="dashboard-input font-mono text-sm"
                placeholder={JSON.stringify(selectedTool.input_schema?.properties || {}, null, 2)}
              />
            </div>
            <button
              type="submit"
              disabled={callToolMutation.isPending}
              className="dashboard-button px-4 py-2"
              data-testid="mcp-call-tool"
            >
              {callToolMutation.isPending ? t('calling') : t('callTool')}
            </button>
          </form>
          {toolResult && (
            <div className="dashboard-muted-card mt-4 rounded-[20px] p-4">
              <h4 className="text-sm font-medium text-slate-700 mb-2">{t('result')}</h4>
              <pre className="text-xs text-slate-600 overflow-x-auto whitespace-pre-wrap">
                {toolResult}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
