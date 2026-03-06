import { useState, useRef } from 'react'
import { useDocuments, useCollections, useDeleteDocument, useDocument } from '../hooks/useApi'
import { api } from '../lib/api'

function getDocumentPath(doc: { path?: string; metadata?: Record<string, unknown> }) {
  if (doc.path) return doc.path
  return typeof doc.metadata?.file_path === 'string' ? doc.metadata.file_path : ''
}

function getDocumentFilename(doc: { path?: string; metadata?: Record<string, unknown> }) {
  const path = getDocumentPath(doc)
  return path.split('/').pop() || path || 'Untitled'
}

function getDocumentCreated(doc: { created?: string; metadata?: Record<string, unknown> }) {
  if (doc.created) return doc.created
  return typeof doc.metadata?.creation_date === 'string' ? doc.metadata.creation_date : ''
}

function getDocumentExtension(doc: { metadata?: Record<string, unknown> }) {
  return typeof doc.metadata?.file_ext === 'string' ? doc.metadata.file_ext : ''
}

function formatDocumentDate(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function DocumentDetailModal({ docId, onClose }: { docId: string; onClose: () => void }) {
  const { data: doc, isLoading, error } = useDocument(docId)

  if (isLoading) {
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-6">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </div>
    )
  }

  if (error || !doc) {
    return (
      <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-lg">
          <p className="text-red-600 dark:text-red-400">Error loading document</p>
          <button onClick={onClose} className="mt-4 px-4 py-2 bg-gray-200 dark:bg-gray-700 rounded">
            Close
          </button>
        </div>
      </div>
    )
  }

  const filename = getDocumentFilename(doc)
  const path = getDocumentPath(doc)
  const created = getDocumentCreated(doc)
  const extension = getDocumentExtension(doc)
  const metadataEntries = Object.entries(doc.metadata ?? {})

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl max-h-[80vh] overflow-auto" onClick={e => e.stopPropagation()}>
        <div className="flex justify-between items-start mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{filename}</h3>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="space-y-3 text-sm">
          <div>
            <span className="text-gray-500 dark:text-gray-400">ID: </span>
            <span className="font-mono text-gray-900 dark:text-white">{doc.id}</span>
          </div>
          <div>
            <span className="text-gray-500 dark:text-gray-400">Path: </span>
            <span className="text-gray-900 dark:text-white break-all">{path || '-'}</span>
          </div>
          <div>
            <span className="text-gray-500 dark:text-gray-400">Created: </span>
            <span className="text-gray-900 dark:text-white">{formatDocumentDate(created)}</span>
          </div>
          <div>
            <span className="text-gray-500 dark:text-gray-400">Type: </span>
            <span className="text-gray-900 dark:text-white">{extension || '-'}</span>
          </div>
          {metadataEntries.length > 0 && (
            <div>
              <span className="text-gray-500 dark:text-gray-400">Metadata: </span>
              <div className="mt-2 rounded-lg bg-gray-50 p-3 dark:bg-gray-900/50">
                <dl className="space-y-2">
                  {metadataEntries.map(([key, value]) => (
                    <div key={key} className="grid grid-cols-[140px_1fr] gap-3">
                      <dt className="font-medium text-gray-500 dark:text-gray-400">{key}</dt>
                      <dd className="break-all text-gray-900 dark:text-white">{String(value)}</dd>
                    </div>
                  ))}
                </dl>
              </div>
            </div>
          )}
          {doc.content && (
            <div>
              <span className="text-gray-500 dark:text-gray-400">Content: </span>
              <pre className="mt-2 whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-xs text-gray-900 dark:bg-gray-900/50 dark:text-white max-h-96 overflow-auto">{doc.content}</pre>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export function Documents() {
  const { data: documents, isLoading: docsLoading, error: docsError } = useDocuments()
  const { data: collections } = useCollections()
  const deleteDoc = useDeleteDocument()
  const [selectedDocId, setSelectedDocId] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      const result = await api.ingest(formData)
      if (result.error) {
        alert(result.error)
      } else {
        window.location.reload()
      }
    } catch (err) {
      alert(`Upload failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
    } finally {
      setUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const handleDelete = async (id: string, path: string) => {
    const filename = path?.split('/').pop() || path || 'this document'
    if (!confirm(`Delete "${filename}"?`)) return
    try {
      await deleteDoc.mutateAsync(id)
    } catch (err) {
      alert(`Delete failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  if (docsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (docsError) {
    return (
      <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <p className="text-red-600 dark:text-red-400">
          Error loading documents: {docsError.message}
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
          Documents & Collections
        </h2>
        <div>
          <input
            ref={fileInputRef}
            type="file"
            onChange={handleUpload}
            className="hidden"
            accept=".txt,.md,.pdf,.json,.csv"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={uploading}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {uploading ? 'Uploading...' : 'Upload File'}
          </button>
        </div>
      </div>

      {collections && collections.length > 0 && (
        <div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-3">
            Collections
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {collections.map((collection) => (
              <div
                key={collection.name}
                className="p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg"
              >
                <h4 className="font-medium text-gray-900 dark:text-white">
                  {collection.name}
                </h4>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {collection.count} documents
                </p>
              </div>
            ))}
          </div>
        </div>
      )}

      {documents && documents.length > 0 && (
        <div>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-3">
            Documents ({documents.length})
          </h3>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Filename
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Path
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                {documents.map((doc) => {
                  const filename = getDocumentFilename(doc)
                  const path = getDocumentPath(doc)
                  const extension = getDocumentExtension(doc)
                  const created = getDocumentCreated(doc)
                  return (
                    <tr key={doc.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                        {filename}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400 max-w-md break-all">
                        {path || '-'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                        {extension || '-'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                        {formatDocumentDate(created)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm space-x-2">
                        <button
                          onClick={() => setSelectedDocId(doc.id)}
                          className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
                        >
                          View
                        </button>
                        <button
                          onClick={() => handleDelete(doc.id, path)}
                          className="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {(!documents || documents.length === 0) && (
        <div className="text-center py-12">
          <p className="text-gray-500 dark:text-gray-400">
            No documents found. Upload a file to get started.
          </p>
        </div>
      )}

      {selectedDocId && (
        <DocumentDetailModal docId={selectedDocId} onClose={() => setSelectedDocId(null)} />
      )}
    </div>
  )
}
