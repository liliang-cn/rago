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
    return <div className="glass-panel rounded-[28px] p-6 text-[#d7e1ec]">{t('loading')}</div>
  }

  if (error) {
    return <div className="glass-panel rounded-[28px] border border-rose-500/30 p-6 text-rose-200">{t('error')}: {error.message}</div>
  }

  return (
    <div className="space-y-6">
      <div className="glass-panel rounded-[32px] p-6">
        <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Project configuration</p>
        <h2 className="mt-2 text-3xl font-semibold text-white">{t('settings')}</h2>
        <p className="mt-3 max-w-3xl text-sm leading-7 text-[#a7b7c9]">
          This page now reads and writes the actual project configuration file used by AgentGo, not a separate UI-only settings file.
        </p>
      </div>

      <form onSubmit={handleSave} className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_380px]">
        <div className="space-y-6">
          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Core</p>
            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Home</span>
                <input value={homeDir} onChange={(e) => setHomeDir(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Debug</span>
                <div className="rounded-[18px] border border-white/10 bg-white/5 px-4 py-3 text-white">
                  <label className="flex items-center gap-3">
                    <input type="checkbox" checked={debug} onChange={(e) => setDebug(e.target.checked)} />
                    Enable verbose runtime logging
                  </label>
                </div>
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Server host</span>
                <input value={serverHost} onChange={(e) => setServerHost(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Server port</span>
                <input value={serverPort} onChange={(e) => setServerPort(e.target.value)} className="dashboard-input" />
              </label>
            </div>
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">MCP</p>
            <div className="mt-5 space-y-4">
              <div className="rounded-[20px] border border-white/10 bg-white/5 px-4 py-3 text-white">
                <label className="flex items-center gap-3">
                  <input type="checkbox" checked={mcpEnabled} onChange={(e) => setMCPEnabled(e.target.checked)} />
                  Enable MCP subsystem
                </label>
              </div>
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Filesystem allowlist</span>
                <textarea value={allowedDirs} onChange={(e) => setAllowedDirs(e.target.value)} rows={5} className="dashboard-input resize-none" />
              </label>
            </div>
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Knowledge and memory</p>
            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Skills paths</span>
                <textarea value={skillsPaths} onChange={(e) => setSkillsPaths(e.target.value)} rows={4} className="dashboard-input resize-none" />
              </label>
              <label className="space-y-2 md:col-span-2">
                <span className="text-sm font-medium text-[#d7e1ec]">RAG database path</span>
                <input value={ragDbPath} onChange={(e) => setRagDbPath(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Memory store type</span>
                <input value={memoryStoreType} onChange={(e) => setMemoryStoreType(e.target.value)} className="dashboard-input" />
              </label>
              <label className="space-y-2">
                <span className="text-sm font-medium text-[#d7e1ec]">Memory path</span>
                <input value={memoryPath} onChange={(e) => setMemoryPath(e.target.value)} className="dashboard-input" />
              </label>
            </div>
          </section>

          <div className="flex items-center gap-4">
            <button type="submit" disabled={updateConfigMutation.isPending} className="dashboard-button">
              {updateConfigMutation.isPending ? t('loading') : t('saveSettings')}
            </button>
            {saved && <span className="text-emerald-300">{t('settingsSaved')}</span>}
          </div>
        </div>

        <aside className="space-y-6">
          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Source of truth</p>
            <dl className="mt-5 space-y-4 text-sm">
              <div>
                <dt className="text-[#8ea0b5]">Config file</dt>
                <dd className="mt-1 break-all text-white">{config?.configPath}</dd>
              </div>
              <div>
                <dt className="text-[#8ea0b5]">Data dir</dt>
                <dd className="mt-1 break-all text-white">{config?.dataDir}</dd>
              </div>
              <div>
                <dt className="text-[#8ea0b5]">Workspace dir</dt>
                <dd className="mt-1 break-all text-white">{config?.workspaceDir}</dd>
              </div>
              <div>
                <dt className="text-[#8ea0b5]">MCP servers file</dt>
                <dd className="mt-1 break-all text-white">{config?.mcpServersPath}</dd>
              </div>
            </dl>
          </section>

          <section className="glass-panel rounded-[32px] p-6">
            <p className="text-xs uppercase tracking-[0.28em] text-[#8ea0b5]">Language</p>
            <div className="mt-5 flex gap-3">
              <button
                type="button"
                onClick={() => i18n.changeLanguage('zh')}
                className={i18n.language === 'zh' ? 'dashboard-button' : 'rounded-full border border-white/10 bg-white/5 px-4 py-3 text-sm text-white'}
              >
                中文
              </button>
              <button
                type="button"
                onClick={() => i18n.changeLanguage('en')}
                className={i18n.language === 'en' ? 'dashboard-button' : 'rounded-full border border-white/10 bg-white/5 px-4 py-3 text-sm text-white'}
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
