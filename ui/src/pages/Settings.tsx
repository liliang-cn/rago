import { useState, useEffect } from 'react'
import { useConfig, useUpdateConfig } from '../hooks/useApi'

export function Settings() {
  const { data: config, isLoading, error } = useConfig()
  const updateConfigMutation = useUpdateConfig()
  const [saved, setSaved] = useState(false)

  const [homeDir, setHomeDir] = useState('')
  const [allowedDirs, setAllowedDirs] = useState('')

  useEffect(() => {
    if (config) {
      setHomeDir(config.home || '')
      setAllowedDirs(config.mcpAllowedDirs?.join('\n') || '')
    }
  }, [config])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await updateConfigMutation.mutateAsync({
        home: homeDir,
        mcpAllowedDirs: allowedDirs.split('\n').map(d => d.trim()).filter(d => d),
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (err) {
      alert(`Failed to save: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  if (isLoading) {
    return <div className="flex items-center justify-center h-64">Loading...</div>
  }

  if (error) {
    return <div className="text-red-500">Error loading config: {error.message}</div>
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold mb-6 dark:text-white">Settings</h2>

      <form onSubmit={handleSave} className="space-y-6">
        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4 dark:text-white">Working Directory</h3>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Home Directory
            </label>
            <input
              type="text"
              value={homeDir}
              onChange={(e) => setHomeDir(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
              placeholder="~/.agentgo"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Base directory for all AgentGo data (default: ~/.agentgo)
            </p>
          </div>
        </div>

        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4 dark:text-white">MCP Filesystem</h3>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Allowed Directories (one per line)
            </label>
            <textarea
              value={allowedDirs}
              onChange={(e) => setAllowedDirs(e.target.value)}
              rows={6}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
              placeholder="/Users/username/projects&#10;/Users/username/documents"
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Directories that MCP filesystem server can access
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={updateConfigMutation.isPending}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {updateConfigMutation.isPending ? 'Saving...' : 'Save Settings'}
          </button>
          {saved && (
            <span className="text-green-600 dark:text-green-400">Settings saved successfully!</span>
          )}
        </div>
      </form>
    </div>
  )
}
