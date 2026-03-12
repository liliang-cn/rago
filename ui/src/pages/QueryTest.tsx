import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryRAG } from '../hooks/useApi'

export function QueryTest() {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [topK, setTopK] = useState(5)
  const mutation = useQueryRAG()

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      mutation.mutate({ query, top_k: topK })
    }
  }

  return (
    <div className="space-y-6" data-testid="page-query">
      <div className="glass-panel rounded-[32px] p-6">
        <h2 className="text-xl font-semibold text-slate-900 mb-4">
          {t('ragQuery')}
        </h2>
        <form onSubmit={handleSubmit} className="space-y-4" data-testid="query-form">
          <div>
            <label
              htmlFor="query"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              {t('query')}
            </label>
            <textarea
              id="query"
              rows={3}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="dashboard-input resize-none"
              placeholder={t('queryPlaceholder')}
              data-testid="query-input"
            />
          </div>
          <div>
            <label
              htmlFor="topK"
              className="block text-sm font-medium text-slate-700 mb-2"
            >
              {t('topKResults', { count: topK })}
            </label>
            <input
              id="topK"
              type="range"
              min={1}
              max={20}
              value={topK}
              onChange={(e) => setTopK(Number(e.target.value))}
              className="w-full"
            />
          </div>
          <button
            type="submit"
            disabled={mutation.isPending || !query.trim()}
            className="dashboard-button px-6 py-2 disabled:cursor-not-allowed"
            data-testid="query-submit"
          >
            {mutation.isPending ? t('querying') : t('query')}
          </button>
        </form>
      </div>

      {mutation.isError && (
        <div className="rounded-[24px] border border-rose-200 bg-rose-50 p-4">
          <p className="text-rose-700">
            {t('error')}: {mutation.error?.message}
          </p>
        </div>
      )}

      {mutation.isSuccess && (
        <div className="space-y-4">
          <div className="glass-panel rounded-[28px] p-6">
            <h3 className="font-medium text-slate-900 mb-3">
              {t('answer')}
            </h3>
            <p className="text-slate-700 whitespace-pre-wrap">
              {mutation.data.answer}
            </p>
          </div>

          {mutation.data.sources && mutation.data.sources.length > 0 && (
            <div>
              <h3 className="font-medium text-slate-900 mb-3">
                {t('sources')}
              </h3>
              <div className="space-y-3">
                {mutation.data.sources.map((source, index) => (
                  <div
                    key={index}
                    className="dashboard-muted-card rounded-[24px] p-4"
                  >
                    <div className="flex justify-between items-center mb-2">
                      <span className="text-sm font-medium text-blue-700">
                        {t('score')}: {source.score.toFixed(4)}
                      </span>
                    </div>
                    <p className="text-sm text-slate-600 line-clamp-3">
                      {source.content}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
