import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMemories, useAddMemory, useDeleteMemory, useSearchMemories } from '../hooks/useApi'
import type { Memory, AddMemoryRequest } from '../lib/api'

export function Memory() {
  const { t } = useTranslation()
  const [showAddForm, setShowAddForm] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedMemory, setSelectedMemory] = useState<Memory | null>(null)
  
  const { data: memories, isLoading, error, refetch } = useMemories()
  const addMutation = useAddMemory()
  const deleteMutation = useDeleteMemory()
  
  // Filter memories based on search
  const filteredMemories = searchQuery 
    ? memories?.filter(m => 
        m.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
        m.type.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : memories

  const handleAddMemory = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const data: AddMemoryRequest = {
      content: formData.get('content') as string,
      type: formData.get('type') as string || 'fact',
      importance: parseFloat(formData.get('importance') as string) || 0.5,
    }
    await addMutation.mutateAsync(data)
    setShowAddForm(false)
  }

  const handleDeleteMemory = async (id: string) => {
    if (confirm('Are you sure you want to delete this memory?')) {
      await deleteMutation.mutateAsync(id)
      if (selectedMemory?.id === id) {
        setSelectedMemory(null)
      }
    }
  }

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'fact': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
      case 'skill': return 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400'
      case 'pattern': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
      case 'context': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
      case 'preference': return 'bg-pink-100 text-pink-800 dark:bg-pink-900/30 dark:text-pink-400'
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
    }
  }

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
        <p className="text-red-600 dark:text-red-400">Error loading memories: {error.message}</p>
        <button onClick={() => refetch()} className="mt-2 px-4 py-2 bg-red-600 text-white rounded-lg">
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Memory</h2>
        <div className="flex gap-2">
          <button
            onClick={() => refetch()}
            className="px-4 py-2 text-sm bg-gray-100 dark:bg-gray-800 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700"
          >
            Refresh
          </button>
          <button
            onClick={() => setShowAddForm(true)}
            className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Add Memory
          </button>
        </div>
      </div>

      {/* Search Bar */}
      <div className="mb-4">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search memories..."
          className="w-full px-4 py-2 border border-gray-300 rounded-lg dark:bg-gray-800 dark:border-gray-600 dark:text-white"
        />
      </div>

      {/* Add Memory Form */}
      {showAddForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-lg mx-4">
            <h3 className="text-lg font-semibold mb-4">Add New Memory</h3>
            <form onSubmit={handleAddMemory} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Type
                </label>
                <select
                  name="type"
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                >
                  <option value="fact">Fact</option>
                  <option value="skill">Skill</option>
                  <option value="pattern">Pattern</option>
                  <option value="context">Context</option>
                  <option value="preference">Preference</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Content *
                </label>
                <textarea
                  name="content"
                  required
                  rows={4}
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                  placeholder="Enter memory content..."
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Importance (0-1)
                </label>
                <input
                  type="number"
                  name="importance"
                  step="0.1"
                  min="0"
                  max="1"
                  defaultValue={0.5}
                  className="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
                />
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={addMutation.isPending}
                  className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {addMutation.isPending ? 'Adding...' : 'Add'}
                </button>
                <button
                  type="button"
                  onClick={() => setShowAddForm(false)}
                  className="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 dark:bg-gray-600 dark:text-gray-200"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Memory Detail Panel */}
      {selectedMemory && (
        <div className="p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">Memory Details</h3>
            <button
              onClick={() => setSelectedMemory(null)}
              className="text-gray-500 hover:text-gray-700 dark:text-gray-400"
            >
              Close
            </button>
          </div>
          <div className="space-y-4">
            <div>
              <span className="text-sm text-gray-500 dark:text-gray-400">ID:</span>
              <p className="font-mono text-sm text-gray-700 dark:text-gray-300">{selectedMemory.id}</p>
            </div>
            <div>
              <span className="text-sm text-gray-500 dark:text-gray-400">Type:</span>
              <p className={`inline-block px-2 py-1 text-xs rounded ${getTypeColor(selectedMemory.type)}`}>
                {selectedMemory.type}
              </p>
            </div>
            <div>
              <span className="text-sm text-gray-500 dark:text-gray-400">Content:</span>
              <p className="text-gray-700 dark:text-gray-300">{selectedMemory.content}</p>
            </div>
            <div className="flex gap-4 text-sm text-gray-500 dark:text-gray-400">
              <span>Importance: {selectedMemory.importance.toFixed(2)}</span>
              <span>Created: {new Date(selectedMemory.created_at).toLocaleString()}</span>
            </div>
            <button
              onClick={() => handleDeleteMemory(selectedMemory.id)}
              className="px-4 py-2 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700"
            >
              Delete Memory
            </button>
          </div>
        </div>
      )}

      {/* Memories List */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredMemories && filteredMemories.length > 0 ? (
          filteredMemories.map((memory) => (
            <div
              key={memory.id}
              className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-blue-500 transition-colors cursor-pointer"
              onClick={() => setSelectedMemory(memory)}
            >
              <div className="flex items-start justify-between mb-2">
                <span className={`px-2 py-1 text-xs rounded ${getTypeColor(memory.type)}`}>
                  {memory.type}
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {memory.importance.toFixed(2)}
                </span>
              </div>
              <p className="text-sm text-gray-700 dark:text-gray-300 line-clamp-3">
                {memory.content}
              </p>
              <p className="text-xs text-gray-400 dark:text-gray-500 mt-2">
                {new Date(memory.created_at).toLocaleDateString()}
              </p>
            </div>
          ))
        ) : (
          <div className="col-span-full text-center py-12 text-gray-500 dark:text-gray-400">
            {searchQuery ? 'No memories match your search.' : 'No memories found. Click "Add Memory" to create one.'}
          </div>
        )}
      </div>
    </div>
  )
}
