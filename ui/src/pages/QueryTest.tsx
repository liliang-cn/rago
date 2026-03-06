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
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
          RAG Query
        </h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="query"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              Query
            </label>
            <textarea
              id="query"
              rows={3}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-800 dark:border-gray-600 dark:text-white"
              placeholder="Enter your query..."
            />
          </div>
          <div>
            <label
              htmlFor="topK"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              Top K Results: {topK}
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
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {mutation.isPending ? 'Querying...' : 'Query'}
          </button>
        </form>
      </div>

      {mutation.isError && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-red-600 dark:text-red-400">
            Error: {mutation.error?.message}
          </p>
        </div>
      )}

      {mutation.isSuccess && (
        <div className="space-y-4">
          <div className="p-6 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <h3 className="font-medium text-gray-900 dark:text-white mb-3">
              Answer
            </h3>
            <p className="text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
              {mutation.data.answer}
            </p>
          </div>

          {mutation.data.sources && mutation.data.sources.length > 0 && (
            <div>
              <h3 className="font-medium text-gray-900 dark:text-white mb-3">
                Sources
              </h3>
              <div className="space-y-3">
                {mutation.data.sources.map((source, index) => (
                  <div
                    key={index}
                    className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
                  >
                    <div className="flex justify-between items-center mb-2">
                      <span className="text-sm font-medium text-blue-600 dark:text-blue-400">
                        Score: {source.score.toFixed(4)}
                      </span>
                    </div>
                    <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-3">
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
