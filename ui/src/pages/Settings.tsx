import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useConfig, useUpdateConfig } from '../hooks/useApi'

export function Settings() {
  const { t, i18n } = useTranslation()
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
      alert(`${t('error')}: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng)
  }

  if (isLoading) {
    return <div className="flex items-center justify-center h-64">{t('loading')}</div>
  }

  if (error) {
    return <div className="text-red-500">{t('error')}: {error.message}</div>
  }

  return (
    <div className="max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold mb-6 dark:text-white">{t('settings')}</h2>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Language */}
        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4 dark:text-white">{t('language') || 'Language'}</h3>
          <div className="flex gap-4">
            <button
              type="button"
              onClick={() => changeLanguage('zh')}
              className={`px-4 py-2 rounded-lg ${
                i18n.language === 'zh'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 dark:text-white'
              }`}
            >
              中文
            </button>
            <button
              type="button"
              onClick={() => changeLanguage('en')}
              className={`px-4 py-2 rounded-lg ${
                i18n.language === 'en'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 dark:text-white'
              }`}
            >
              English
            </button>
          </div>
        </div>

        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4 dark:text-white">{t('workingDirectory')}</h3>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('homeDirectory')}
            </label>
            <input
              type="text"
              value={homeDir}
              onChange={(e) => setHomeDir(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
              placeholder={t('homeDirectoryPlaceholder')}
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t('homeDirectoryDesc')}
            </p>
          </div>
        </div>

        <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-4 dark:text-white">{t('mcpFilesystem')}</h3>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              {t('allowedDirectories')}
            </label>
            <textarea
              value={allowedDirs}
              onChange={(e) => setAllowedDirs(e.target.value)}
              rows={6}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
              placeholder={t('allowedDirectoriesPlaceholder')}
            />
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t('allowedDirectoriesDesc')}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={updateConfigMutation.isPending}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {updateConfigMutation.isPending ? t('loading') : t('saveSettings')}
          </button>
          {saved && (
            <span className="text-green-600 dark:text-green-400">{t('settingsSaved')}</span>
          )}
        </div>
      </form>
    </div>
  )
}
