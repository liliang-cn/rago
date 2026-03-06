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
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">{t('mcp')}</h2>
        <div className="flex gap-2">
          <button
            onClick={() => {
              refetchServers()
              refetchTools()
            }}
            className="px-4 py-2 text-sm bg-gray-100 dark:bg-gray-800 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700"
          >
            Refresh
          </button>
          <button
            onClick={() => setShowAddForm(true)}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Add Server
          </button>
        </div>
      </div>

      {/* Add Server Form */}
      {showAddForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-md mx-4">
            <h3 className="text-lg font-semibold mb-4">Add MCP Server</h3>
            <form onSubmit={handleAddServer} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Name *
                </label>
                <input
                  name="name"
                  required
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                  placeholder="my-server"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Type
                </label>
                <select
                  name="type"
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                >
                  <option value="stdio">Stdio</option>
                  <option value="http">HTTP</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Command (for stdio)
                </label>
                <input
                  name="command"
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                  placeholder="npx -y @my-server/mcp-server"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Args (space-separated)
                </label>
                <input
                  name="args"
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                  placeholder="--port 3000"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  URL (for HTTP)
                </label>
                <input
                  name="url"
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                  placeholder="http://localhost:3000/mcp"
                />
              </div>
              <div className="flex justify-end gap-2 pt-4">
                <button
                  type="button"
                  onClick={() => setShowAddForm(false)}
                  className="px-4 py-2 text-gray-600 dark:text-gray-400"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg"
                >
                  Add Server
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Servers List */}
        <div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Servers</h3>
          <div className="space-y-3">
            {servers && servers.length > 0 ? (
              servers.map((server) => (
                <div
                  key={server.name}
                  className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
                >
                  <div className="flex items-center justify-between mb-2">
                    <h4 className="font-medium text-gray-900 dark:text-white">
                      {server.name}
                    </h4>
                    <span
                      className={`px-2 py-1 text-xs rounded ${
                        server.running
                          ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                          : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
                      }`}
                    >
                      {server.running ? 'Running' : 'Stopped'}
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                    {server.description || server.command || 'No command'}
                  </p>
                  <p className="text-sm text-gray-500 dark:text-gray-500">
                    Tools: {server.tool_count}
                  </p>
                </div>
              ))
            ) : (
              <div className="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg text-center text-gray-500 dark:text-gray-400">
                No MCP servers configured
              </div>
            )}
          </div>
        </div>

        {/* Tools List */}
        <div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Available Tools</h3>
          <div className="space-y-3 max-h-96 overflow-y-auto">
            {tools && tools.length > 0 ? (
              tools.map((tool) => (
                <div
                  key={`${tool.server_name}-${tool.name}`}
                  className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg cursor-pointer hover:border-blue-500 transition-colors"
                  onClick={() => setSelectedTool(tool)}
                >
                  <div className="flex items-start justify-between mb-1">
                    <h4 className="font-medium text-gray-900 dark:text-white text-sm">
                      {tool.name}
                    </h4>
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      {tool.server_name}
                    </span>
                  </div>
                  <p className="text-xs text-gray-600 dark:text-gray-400 line-clamp-2">
                    {tool.description}
                  </p>
                </div>
              ))
            ) : (
              <div className="p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg text-center text-gray-500 dark:text-gray-400">
                No tools available
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Tool Test Panel */}
      {selectedTool && (
        <div className="mt-6 p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
              Test Tool: {selectedTool.name}
            </h3>
            <button
              onClick={() => {
                setSelectedTool(null)
                setToolResult(null)
              }}
              className="text-gray-500 hover:text-gray-700"
            >
              Close
            </button>
          </div>
          <form onSubmit={handleCallTool} className="space-y-4">
            <input type="hidden" name="tool_name" value={selectedTool.name} />
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Arguments (JSON)
              </label>
              <textarea
                name="arguments"
                rows={4}
                className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 font-mono text-sm"
                placeholder={JSON.stringify(selectedTool.input_schema?.properties || {}, null, 2)}
              />
            </div>
            <button
              type="submit"
              disabled={callToolMutation.isPending}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {callToolMutation.isPending ? 'Calling...' : 'Call Tool'}
            </button>
          </form>
          {toolResult && (
            <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
              <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Result</h4>
              <pre className="text-xs text-gray-600 dark:text-gray-400 overflow-x-auto whitespace-pre-wrap">
                {toolResult}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
