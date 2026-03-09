import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSkills, useCreateSkill, useDeleteSkill } from '../hooks/useApi'
import type { Skill, CreateSkillRequest } from '../lib/api'

export function Skills() {
  const { t } = useTranslation()
  const [showAddForm, setShowAddForm] = useState(false)
  const [selectedSkill, setSelectedSkill] = useState<Skill | null>(null)
  const { data: skills, isLoading, error, refetch } = useSkills()
  const createMutation = useCreateSkill()
  const deleteMutation = useDeleteSkill()

  const handleCreateSkill = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    const data: CreateSkillRequest = {
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      content: formData.get('content') as string,
    }
    await createMutation.mutateAsync(data)
    setShowAddForm(false)
  }

  const handleDeleteSkill = async (id: string) => {
    if (confirm('Are you sure you want to delete this skill?')) {
      await deleteMutation.mutateAsync(id)
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
        <p className="text-rose-700">{t('errorLoadingSkills')}: {error.message}</p>
        <button onClick={() => refetch()} className="dashboard-button mt-2 px-4 py-2">
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-slate-900">{t('skills')}</h2>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="dashboard-button px-4 py-2"
        >
          {showAddForm ? 'Cancel' : 'Add Skill'}
        </button>
      </div>

      {showAddForm && (
        <div className="glass-panel rounded-[28px] p-6">
          <h3 className="text-lg font-medium text-slate-900 mb-4">{t('createNewSkill')}</h3>
          <form onSubmit={handleCreateSkill} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Name *
              </label>
              <input
                type="text"
                name="name"
                required
                className="dashboard-input"
                placeholder="my-skill"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Description
              </label>
              <input
                type="text"
                name="description"
                className="dashboard-input"
                placeholder="A brief description of the skill"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-2">
                Content (SKILL.md format) *
              </label>
              <textarea
                name="content"
                required
                rows={10}
                className="dashboard-input font-mono text-sm"
                placeholder={`# My Skill

## Variables
- input: The input to process (required)

## Steps

### Step 1: Process
Process the input and generate output.`}
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={createMutation.isPending}
                className="dashboard-button px-6 py-2"
              >
                {createMutation.isPending ? 'Creating...' : 'Create'}
              </button>
              <button
                type="button"
                onClick={() => setShowAddForm(false)}
                className="dashboard-secondary-button px-6 py-2"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {selectedSkill && (
        <div className="glass-panel rounded-[28px] p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-slate-900">{selectedSkill.name}</h3>
            <button
              onClick={() => setSelectedSkill(null)}
              className="text-slate-500 hover:text-slate-700"
            >
              Close
            </button>
          </div>
          <div className="space-y-4">
            <p className="text-slate-600">{selectedSkill.description}</p>
            <div className="text-sm text-slate-500">
              <p>Version: {selectedSkill.version}</p>
              <p>Path: {selectedSkill.path}</p>
              <p>Enabled: {selectedSkill.enabled ? 'Yes' : 'No'}</p>
            </div>
            {selectedSkill.variables && Object.keys(selectedSkill.variables).length > 0 && (
              <div>
                <h4 className="font-medium text-slate-900 mb-2">Variables</h4>
                <div className="space-y-2">
                  {Object.entries(selectedSkill.variables).map(([name, def]) => (
                    <div key={name} className="dashboard-muted-card rounded-[20px] p-3">
                      <p className="font-medium text-slate-900">{name}</p>
                      <p className="text-sm text-slate-600">{def.description}</p>
                      <p className="text-xs text-slate-500">Type: {def.type} | Required: {def.required ? 'Yes' : 'No'}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {skills && skills.length > 0 ? (
          skills.map((skill) => (
            <div
              key={skill.id}
              className="dashboard-muted-card rounded-[24px] p-4 hover:border-sky-300 transition-colors"
            >
              <div className="flex items-start justify-between mb-2">
                <h3
                  className="font-medium text-slate-900 cursor-pointer hover:text-blue-600"
                  onClick={() => setSelectedSkill(skill)}
                >
                  {skill.name}
                </h3>
                <span className={`px-2 py-1 text-xs rounded ${skill.enabled ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'}`}>
                  {skill.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>
              <p className="text-sm text-slate-600 mb-3 line-clamp-2">
                {skill.description || 'No description'}
              </p>
              <div className="flex items-center justify-between text-xs text-slate-500">
                <span>v{skill.version}</span>
                <button
                  onClick={() => handleDeleteSkill(skill.id)}
                  className="text-red-600 hover:text-red-700"
                >
                  Delete
                </button>
              </div>
            </div>
          ))
        ) : (
          <div className="col-span-full text-center py-12 text-slate-500">
            No skills found. Click "Add Skill" to create one.
          </div>
        )}
      </div>
    </div>
  )
}
