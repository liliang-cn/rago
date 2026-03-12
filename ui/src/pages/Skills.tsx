import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSkills, useCreateSkill, useDeleteSkill } from '../hooks/useApi'
import type { Skill, CreateSkillRequest } from '../lib/api'

function formatTimestamp(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function displayValue(value?: string | number | boolean | null) {
  if (value === undefined || value === null || value === '') return '-'
  return String(value)
}

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
    if (confirm(t('confirmDeleteSkill'))) {
      await deleteMutation.mutateAsync(id)
      if (selectedSkill?.id === id) {
        setSelectedSkill(null)
      }
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-[24px] border border-rose-200 bg-rose-50 p-4">
        <p className="text-rose-700">{t('errorLoadingSkills')}: {error.message}</p>
        <button onClick={() => refetch()} className="dashboard-button mt-2 px-4 py-2">
          {t('retry')}
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6" data-testid="page-skills">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-slate-900">{t('skills')}</h2>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="dashboard-button px-4 py-2"
          data-testid="skills-toggle-create"
        >
          {showAddForm ? t('cancel') : t('addSkillButton')}
        </button>
      </div>

      {showAddForm && (
        <div className="glass-panel rounded-[28px] p-6" data-testid="skills-create-panel">
          <h3 className="mb-4 text-lg font-medium text-slate-900">{t('createNewSkill')}</h3>
          <form onSubmit={handleCreateSkill} className="space-y-4" data-testid="skills-create-form">
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-700">
                {t('skillNameRequired')}
              </label>
              <input
                type="text"
                name="name"
                required
                className="dashboard-input"
                placeholder={t('skillNameExample')}
              />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-700">
                {t('skillDescriptionLabel')}
              </label>
              <input
                type="text"
                name="description"
                className="dashboard-input"
                placeholder={t('skillDescriptionPlaceholder')}
              />
            </div>
            <div>
              <label className="mb-2 block text-sm font-medium text-slate-700">
                {t('skillContentLabel')}
              </label>
              <textarea
                name="content"
                required
                rows={10}
                className="dashboard-input font-mono text-sm"
                placeholder={t('skillContentPlaceholder')}
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={createMutation.isPending}
                className="dashboard-button px-6 py-2"
                data-testid="skills-create-submit"
              >
                {createMutation.isPending ? t('creating') : t('createButton')}
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
      )}

      <div className="space-y-4" data-testid="skills-list">
        {skills && skills.length > 0 ? (
          skills.map((skill) => (
            <section
              key={skill.id}
              data-testid={`skill-card-${skill.id}`}
              className="dashboard-muted-card rounded-[24px] p-4"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="font-medium text-slate-900">{skill.name}</h3>
                    <span className={`rounded px-2 py-1 text-xs ${skill.enabled ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-700'}`}>
                      {skill.enabled ? t('skillEnabled') : t('disabled')}
                    </span>
                  </div>
                  <p className="mt-2 text-sm text-slate-600">
                    {skill.description || t('noDescription')}
                  </p>
                  <div className="mt-2 flex flex-wrap gap-3 text-xs text-slate-500">
                    <span>{t('version')}: {displayValue(skill.version)}</span>
                    <span>{t('category')}: {displayValue(skill.category)}</span>
                    <span>{t('author')}: {displayValue(skill.author)}</span>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-3 text-sm">
                  <button
                    type="button"
                    onClick={() => setSelectedSkill(skill)}
                    className="text-blue-600 hover:text-blue-700"
                    data-testid={`skill-open-${skill.id}`}
                  >
                    {t('viewDetails')}
                  </button>
                  <button
                    type="button"
                    onClick={() => handleDeleteSkill(skill.id)}
                    className="text-red-600 hover:text-red-700"
                  >
                    {t('delete')}
                  </button>
                </div>
              </div>
            </section>
          ))
        ) : (
          <div className="py-12 text-center text-slate-500">
            {t('noSkillsFound')}
          </div>
        )}
      </div>

      {selectedSkill && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/18 px-4 backdrop-blur-sm"
          onClick={() => setSelectedSkill(null)}
          data-testid="skills-detail-modal-overlay"
        >
          <div
            className="glass-panel max-h-[85vh] w-full max-w-4xl overflow-auto rounded-[28px] p-6"
            onClick={(event) => event.stopPropagation()}
            data-testid="skills-detail-modal"
          >
            <div className="flex items-start justify-between gap-4">
              <div>
                <h3 className="text-xl font-semibold text-slate-900">{selectedSkill.name}</h3>
                <p className="mt-2 max-w-3xl text-sm text-slate-600">
                  {selectedSkill.description || t('noDescription')}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setSelectedSkill(null)}
                className="dashboard-secondary-button px-4 py-2"
              >
                {t('closeButton')}
              </button>
            </div>

            <div className="mt-6 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <div className="rounded-[18px] border border-sky-100 bg-white p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{t('author')}</p>
                <p className="mt-2 text-sm font-medium text-slate-900">{displayValue(selectedSkill.author)}</p>
              </div>
              <div className="rounded-[18px] border border-sky-100 bg-white p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{t('category')}</p>
                <p className="mt-2 text-sm font-medium text-slate-900">{displayValue(selectedSkill.category)}</p>
              </div>
              <div className="rounded-[18px] border border-sky-100 bg-white p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{t('version')}</p>
                <p className="mt-2 text-sm font-medium text-slate-900">{displayValue(selectedSkill.version)}</p>
              </div>
              <div className="rounded-[18px] border border-sky-100 bg-white p-3">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-500">{t('enabled')}</p>
                <p className="mt-2 text-sm font-medium text-slate-900">{selectedSkill.enabled ? t('yes') : t('no')}</p>
              </div>
            </div>

            <div className="mt-4 rounded-[18px] border border-sky-100 bg-white p-4">
              <div className="grid gap-3 md:grid-cols-2">
                <p className="text-sm text-slate-600">
                  <span className="font-medium text-slate-900">{t('path')}:</span> {displayValue(selectedSkill.path)}
                </p>
                <p className="text-sm text-slate-600">
                  <span className="font-medium text-slate-900">{t('created')}:</span> {formatTimestamp(selectedSkill.created_at || selectedSkill.created)}
                </p>
              </div>
              <div className="mt-4">
                <h4 className="text-sm font-medium text-slate-900">{t('tags')}</h4>
                <div className="mt-2 flex flex-wrap gap-2">
                  {selectedSkill.tags && selectedSkill.tags.length > 0 ? (
                    selectedSkill.tags.map((tag) => (
                      <span key={tag} className="rounded-full bg-sky-100 px-2 py-1 text-xs font-medium text-sky-700">
                        {tag}
                      </span>
                    ))
                  ) : (
                    <span className="text-sm text-slate-500">-</span>
                  )}
                </div>
              </div>
            </div>

            {selectedSkill.variables && Object.keys(selectedSkill.variables).length > 0 && (
              <div className="mt-4">
                <h4 className="mb-2 text-sm font-medium text-slate-900">{t('variables')}</h4>
                <div className="space-y-2">
                  {Object.entries(selectedSkill.variables).map(([name, def]) => (
                    <div key={name} className="rounded-[18px] border border-sky-100 bg-white p-3">
                      <p className="font-medium text-slate-900">{name}</p>
                      <p className="text-sm text-slate-600">{def.description || t('noDescription')}</p>
                      <p className="text-xs text-slate-500">
                        {t('type')}: {def.type} | {t('required')}: {def.required ? t('yes') : t('no')}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {selectedSkill.steps && selectedSkill.steps.length > 0 && (
              <div className="mt-4">
                <h4 className="mb-2 text-sm font-medium text-slate-900">{t('steps')}</h4>
                <div className="space-y-2">
                  {selectedSkill.steps.map((step, index) => (
                    <div key={step.id || `${selectedSkill.id}-step-${index}`} className="rounded-[18px] border border-sky-100 bg-white p-3">
                      <p className="font-medium text-slate-900">
                        {index + 1}. {step.title || t('untitledStep')}
                      </p>
                      <p className="mt-1 text-sm text-slate-600">{step.description || t('noDescription')}</p>
                      {step.content && (
                        <pre className="mt-2 overflow-x-auto rounded-[14px] bg-slate-50 p-3 text-xs text-slate-700">
                          <code>{step.content}</code>
                        </pre>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
