import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMemories, useAddMemory, useDeleteMemory } from '../hooks/useApi'
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
    if (confirm(t('confirmDeleteMemory'))) {
      await deleteMutation.mutateAsync(id)
      if (selectedMemory?.id === id) {
        setSelectedMemory(null)
      }
    }
  }

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'fact': return 'bg-blue-100 text-blue-700'
      case 'skill': return 'bg-indigo-100 text-indigo-700'
      case 'pattern': return 'bg-emerald-100 text-emerald-700'
      case 'context': return 'bg-amber-100 text-amber-700'
      case 'preference': return 'bg-pink-100 text-pink-700'
      default: return 'bg-slate-100 text-slate-700'
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
      <div className="rounded-[24px] border border-rose-200 bg-rose-50 p-4">
        <p className="text-rose-700">{t('errorLoadingMemories')}: {error.message}</p>
        <button onClick={() => refetch()} className="mt-2 px-4 py-2 bg-red-600 text-white rounded-lg">
          {t('retry')}
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6" data-testid="page-memory">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-slate-900">{t('memoryNav')}</h2>
        <div className="flex gap-2">
          <button
            onClick={() => refetch()}
            className="dashboard-secondary-button px-4 py-2 text-sm"
            data-testid="memory-refresh"
          >
            {t('refresh')}
          </button>
          <button
            onClick={() => setShowAddForm(true)}
            className="dashboard-button px-4 py-2 text-sm"
            data-testid="memory-add"
          >
            {t('addMemory')}
          </button>
        </div>
      </div>

      {/* Search Bar */}
      <div className="glass-panel rounded-[28px] p-4">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t('searchMemories')}
          className="dashboard-input"
          data-testid="memory-search"
        />
      </div>

      {/* Add Memory Form */}
      {showAddForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-sky-950/10 backdrop-blur-sm">
          <div className="glass-panel mx-4 w-full max-w-lg rounded-[28px] p-6" data-testid="memory-add-modal">
            <h3 className="mb-4 text-lg font-semibold text-slate-900">{t('addNewMemory')}</h3>
            <form onSubmit={handleAddMemory} className="space-y-4" data-testid="memory-add-form">
              <div>
                <label className="mb-2 block text-sm font-medium text-slate-700">
                  {t('typeLabel')}
                </label>
                <select
                  name="type"
                  className="dashboard-input"
                >
                  <option value="fact">{t('fact')}</option>
                  <option value="skill">{t('skill')}</option>
                  <option value="pattern">{t('pattern')}</option>
                  <option value="context">{t('context')}</option>
                  <option value="preference">{t('preference')}</option>
                </select>
              </div>
              <div>
                <label className="mb-2 block text-sm font-medium text-slate-700">
                  {t('contentLabel')} *
                </label>
                <textarea
                  name="content"
                  required
                  rows={4}
                  className="dashboard-input resize-none"
                  placeholder={t('enterMemoryContent')}
                />
              </div>
              <div>
                <label className="mb-2 block text-sm font-medium text-slate-700">
                  {t('importanceLabel')} (0-1)
                </label>
                <input
                  type="number"
                  name="importance"
                  step="0.1"
                  min="0"
                  max="1"
                  defaultValue={0.5}
                  className="dashboard-input"
                />
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={addMutation.isPending}
                  className="dashboard-button px-6 py-2 disabled:opacity-50"
                  data-testid="memory-add-submit"
                >
                  {addMutation.isPending ? t('adding') : t('addButton')}
                </button>
                <button
                  type="button"
                  onClick={() => setShowAddForm(false)}
                  className="dashboard-secondary-button px-6 py-2"
                >
                  {t('cancel')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Memory Detail Panel */}
      {selectedMemory && (
        <div className="glass-panel rounded-[28px] p-6" data-testid="memory-detail-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-slate-900">{t('memoryDetails')}</h3>
            <button
              onClick={() => setSelectedMemory(null)}
              className="text-slate-500 hover:text-slate-700"
            >
              {t('closeButton')}
            </button>
          </div>
          <div className="space-y-4">
            <div>
              <span className="text-sm text-slate-500">{t('id')}:</span>
              <p className="font-mono text-sm text-slate-700">{selectedMemory.id}</p>
            </div>
            <div>
              <span className="text-sm text-slate-500">{t('typeLabel')}:</span>
              <p className={`inline-block px-2 py-1 text-xs rounded ${getTypeColor(selectedMemory.type)}`}>
                {selectedMemory.type}
              </p>
            </div>
            <div>
              <span className="text-sm text-slate-500">{t('contentLabel')}:</span>
              <p className="text-slate-700">{selectedMemory.content}</p>
            </div>
            <div className="flex gap-4 text-sm text-slate-500">
              <span>{t('importanceLabel')}: {selectedMemory.importance.toFixed(2)}</span>
              <span>{t('created')}: {new Date(selectedMemory.created_at).toLocaleString()}</span>
            </div>
            <button
              onClick={() => handleDeleteMemory(selectedMemory.id)}
              className="px-4 py-2 text-sm bg-red-600 text-white rounded-lg hover:bg-red-700"
              data-testid="memory-delete"
            >
              {t('deleteMemory')}
            </button>
          </div>
        </div>
      )}

      {/* Memories List */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4" data-testid="memory-list">
        {filteredMemories && filteredMemories.length > 0 ? (
          filteredMemories.map((memory) => (
            <div
              key={memory.id}
              className="rounded-[24px] border border-sky-100 bg-white p-4 transition-colors hover:border-sky-300 hover:bg-sky-50/40 cursor-pointer"
              onClick={() => setSelectedMemory(memory)}
              data-testid={`memory-card-${memory.id}`}
            >
              <div className="flex items-start justify-between mb-2">
                <span className={`px-2 py-1 text-xs rounded ${getTypeColor(memory.type)}`}>
                  {memory.type}
                </span>
                <span className="text-xs text-slate-500">
                  {memory.importance.toFixed(2)}
                </span>
              </div>
              <p className="line-clamp-3 text-sm text-slate-700">
                {memory.content}
              </p>
              <p className="mt-2 text-xs text-slate-400">
                {new Date(memory.created_at).toLocaleDateString()}
              </p>
            </div>
          ))
        ) : (
          <div className="col-span-full rounded-[24px] border border-dashed border-sky-100 bg-sky-50/60 py-12 text-center text-slate-500">
            {searchQuery ? t('noMemoriesMatch') : t('noMemoriesFoundCta')}
          </div>
        )}
      </div>
    </div>
  )
}
