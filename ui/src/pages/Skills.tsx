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
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <p className="text-red-600 dark:text-red-400">Error loading skills: {error.message}</p>
        <button onClick={() => refetch()} className="mt-2 px-4 py-2 bg-red-600 text-white rounded-lg">
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Skills</h2>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          {showAddForm ? 'Cancel' : 'Add Skill'}
        </button>
      </div>

      {showAddForm && (
        <div className="p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Create New Skill</h3>
          <form onSubmit={handleCreateSkill} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Name *
              </label>
              <input
                type="text"
                name="name"
                required
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                placeholder="my-skill"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Description
              </label>
              <input
                type="text"
                name="description"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                placeholder="A brief description of the skill"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Content (SKILL.md format) *
              </label>
              <textarea
                name="content"
                required
                rows={10}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono text-sm dark:bg-gray-700 dark:border-gray-600 dark:text-white"
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
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {createMutation.isPending ? 'Creating...' : 'Create'}
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
      )}

      {selectedSkill && (
        <div className="p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white">{selectedSkill.name}</h3>
            <button
              onClick={() => setSelectedSkill(null)}
              className="text-gray-500 hover:text-gray-700 dark:text-gray-400"
            >
              Close
            </button>
          </div>
          <div className="space-y-4">
            <p className="text-gray-600 dark:text-gray-400">{selectedSkill.description}</p>
            <div className="text-sm text-gray-500 dark:text-gray-400">
              <p>Version: {selectedSkill.version}</p>
              <p>Path: {selectedSkill.path}</p>
              <p>Enabled: {selectedSkill.enabled ? 'Yes' : 'No'}</p>
            </div>
            {selectedSkill.variables && Object.keys(selectedSkill.variables).length > 0 && (
              <div>
                <h4 className="font-medium text-gray-900 dark:text-white mb-2">Variables</h4>
                <div className="space-y-2">
                  {Object.entries(selectedSkill.variables).map(([name, def]) => (
                    <div key={name} className="p-3 bg-gray-50 dark:bg-gray-700 rounded">
                      <p className="font-medium text-gray-900 dark:text-white">{name}</p>
                      <p className="text-sm text-gray-600 dark:text-gray-400">{def.description}</p>
                      <p className="text-xs text-gray-500">Type: {def.type} | Required: {def.required ? 'Yes' : 'No'}</p>
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
              className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-blue-500 transition-colors"
            >
              <div className="flex items-start justify-between mb-2">
                <h3
                  className="font-medium text-gray-900 dark:text-white cursor-pointer hover:text-blue-600"
                  onClick={() => setSelectedSkill(skill)}
                >
                  {skill.name}
                </h3>
                <span className={`px-2 py-1 text-xs rounded ${skill.enabled ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'}`}>
                  {skill.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-3 line-clamp-2">
                {skill.description || 'No description'}
              </p>
              <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
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
          <div className="col-span-full text-center py-12 text-gray-500 dark:text-gray-400">
            No skills found. Click "Add Skill" to create one.
          </div>
        )}
      </div>
    </div>
  )
}
