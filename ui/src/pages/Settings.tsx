import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useConfig, useUpdateConfig } from '../hooks/useApi'

function splitLines(value: string) {
  return value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
}

export function Settings() {
  const { t, i18n } = useTranslation()
  const { data: config, isLoading, error } = useConfig()
  const updateConfigMutation = useUpdateConfig()
  const [saved, setSaved] = useState(false)

  const [homeDir, setHomeDir] = useState('')
  const [debug, setDebug] = useState(false)
  const [serverHost, setServerHost] = useState('')
  const [serverPort, setServerPort] = useState('7127')
  const [mcpEnabled, setMCPEnabled] = useState(true)
  const [allowedDirs, setAllowedDirs] = useState('')
  const [skillsPaths, setSkillsPaths] = useState('')
  const [ragDbPath, setRagDbPath] = useState('')
  const [memoryStoreType, setMemoryStoreType] = useState('')
  const [memoryPath, setMemoryPath] = useState('')

  useEffect(() => {
    if (!config) return
    setHomeDir(config.home || '')
    setDebug(Boolean(config.debug))
    setServerHost(config.serverHost || '')
    setServerPort(String(config.serverPort || 7127))
    setMCPEnabled(Boolean(config.mcpEnabled))
    setAllowedDirs(config.mcpAllowedDirs?.join('\n') || '')
    setSkillsPaths(config.skillsPaths?.join('\n') || '')
    setRagDbPath(config.ragDbPath || '')
    setMemoryStoreType(config.memoryStoreType || '')
    setMemoryPath(config.memoryPath || '')
  }, [config])

  const handleSave = async (event: React.FormEvent) => {
    event.preventDefault()
    try {
      await updateConfigMutation.mutateAsync({
        home: homeDir,
        debug,
        serverHost,
        serverPort: Number(serverPort),
        mcpEnabled,
        mcpAllowedDirs: splitLines(allowedDirs),
        skillsPaths: splitLines(skillsPaths),
        ragDbPath,
        memoryStoreType,
        memoryPath,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (mutationError) {
      alert(`${t('error')}: ${mutationError instanceof Error ? mutationError.message : 'Unknown error'}`)
    }
  }

  if (isLoading) {
    return <div className="glass-panel rounded-[28px] p-6 text-slate-600">{t('loading')}</div>
  }

  if (error) {
    return <div className="rounded-[28px] border border-rose-200 bg-rose-50 p-6 text-rose-700">{t('error')}: {error.message}</div>
  }

  return (
    <div className="space-y-6" data-testid="page-settings">
      <div className="glass-panel rounded-[32px] p-6">
        <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('projectConfiguration')}</p>
        <h2 className="mt-2 text-3xl font-semibold text-slate-900">{t('settings')}</h2>
        <p className="mt-3 max-w-3xl text-sm leading-7 text-slate-600">
          {t('settingsIntro')}
        </p>
      </div>

      <form onSubmit={handleSave} className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_380px]" data-testid="settings-form">
        <div className="space-y-6">
          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('core')}</p>
            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('home')}</span>
                <input value={homeDir} onChange={(e) => setHomeDir(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('debugLabel')}</span>
                <div className="dashboard-muted-card rounded-[18px] px-4 py-3 text-slate-700">
                  <label className="flex items-center gap-3">
                    <input type="checkbox" checked={debug} onChange={(e) => setDebug(e.target.checked)} />
                    {t('enableVerboseRuntimeLogging')}
                  </label>
                </div>
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('serverHost')}</span>
                <input value={serverHost} onChange={(e) => setServerHost(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('serverPort')}</span>
                <input value={serverPort} onChange={(e) => setServerPort(e.target.value)} className="dashboard-input" />
              </label>
            </div>
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">MCP</p>
            <div className="mt-5 space-y-4">
              <div className="dashboard-muted-card rounded-[20px] px-4 py-3 text-slate-700">
                <label className="flex items-center gap-3">
                  <input type="checkbox" checked={mcpEnabled} onChange={(e) => setMCPEnabled(e.target.checked)} />
                  {t('mcp')}
                </label>
              </div>
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('filesystemAllowlist')}</span>
                <textarea value={allowedDirs} onChange={(e) => setAllowedDirs(e.target.value)} rows={5} className="dashboard-input resize-none" />
              </label>
            </div>
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('knowledgeAndMemory')}</p>
            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm font-medium text-slate-700">{t('skillsPaths')}</span>
                <textarea value={skillsPaths} onChange={(e) => setSkillsPaths(e.target.value)} rows={4} className="dashboard-input resize-none" />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm font-medium text-slate-700">{t('ragDatabasePath')}</span>
                <input value={ragDbPath} onChange={(e) => setRagDbPath(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('memoryStoreType')}</span>
                <input value={memoryStoreType} onChange={(e) => setMemoryStoreType(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-slate-700">{t('memoryPath')}</span>
                <input value={memoryPath} onChange={(e) => setMemoryPath(e.target.value)} className="dashboard-input" />
              </label>
            </div>
          </section>

          <div className="flex items-center gap-4">
            <button type="submit" disabled={updateConfigMutation.isPending} className="dashboard-button">
              {updateConfigMutation.isPending ? t('loading') : t('saveSettings')}
            </button>
            {saved && <span className="text-emerald-600">{t('settingsSaved')}</span>}
          </div>
        </div>

        <aside className="space-y-6">
          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('sourceOfTruth')}</p>
            <dl className="mt-5 space-y-4 text-sm">
              <div>
                <dt className="text-slate-500">{t('configFile')}</dt>
                <dd className="mt-1 break-all text-slate-900">{config?.configPath}</dd>
              </div>
              <div>
                <dt className="text-slate-500">{t('dataDir')}</dt>
                <dd className="mt-1 break-all text-slate-900">{config?.dataDir}</dd>
              </div>
              <div>
                <dt className="text-slate-500">{t('workspaceDir')}</dt>
                <dd className="mt-1 break-all text-slate-900">{config?.workspaceDir}</dd>
              </div>
              <div>
                <dt className="text-slate-500">{t('mcpServersFile')}</dt>
                <dd className="mt-1 break-all text-slate-900">{config?.mcpServersPath}</dd>
              </div>
            </dl>
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-slate-500">{t('language')}</p>
            <div className="mt-5 flex gap-3">
              <button
                type="button"
                onClick={() => i18n.changeLanguage('zh')}
                className={i18n.language === 'zh' ? 'dashboard-button' : 'dashboard-secondary-button px-4 py-3 text-sm'}
              >
                中文
              </button>
              <button
                type="button"
                onClick={() => i18n.changeLanguage('en')}
                className={i18n.language === 'en' ? 'dashboard-button' : 'dashboard-secondary-button px-4 py-3 text-sm'}
              >
                English
              </button>
            </div>
          </section>
        </aside>
      </form>
    </div>
  )
}
