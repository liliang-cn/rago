import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { apiClient, Document } from '@/lib/api'
import { FileText, Trash2, AlertTriangle, Loader2, Search } from 'lucide-react'
import { Input } from '@/components/ui/input'

export function DocumentsTab() {
  const [documents, setDocuments] = useState<Document[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [deleteLoading, setDeleteLoading] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [filteredDocuments, setFilteredDocuments] = useState<Document[]>([])

  const loadDocuments = async () => {
    setIsLoading(true)
    try {
      const response = await apiClient.getDocuments()
      if (response.data) {
        setDocuments(response.data)
        setFilteredDocuments(response.data)
      }
    } catch (error) {
      console.error('Failed to load documents:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this document?')) return

    setDeleteLoading(id)
    try {
      await apiClient.deleteDocument(id)
      setDocuments(prev => prev.filter(doc => doc.id !== id))
      setFilteredDocuments(prev => prev.filter(doc => doc.id !== id))
    } catch (error) {
      console.error('Failed to delete document:', error)
    } finally {
      setDeleteLoading(null)
    }
  }

  const handleReset = async () => {
    if (!confirm('Are you sure you want to delete ALL documents? This action cannot be undone.')) return

    setIsLoading(true)
    try {
      await apiClient.reset()
      setDocuments([])
      setFilteredDocuments([])
    } catch (error) {
      console.error('Failed to reset documents:', error)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadDocuments()
  }, [])

  useEffect(() => {
    if (!searchQuery.trim()) {
      setFilteredDocuments(documents)
    } else {
      const filtered = documents.filter(doc => 
        (doc.title || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        doc.content.toLowerCase().includes(searchQuery.toLowerCase())
      )
      setFilteredDocuments(filtered)
    }
  }, [searchQuery, documents])

  const formatDate = (dateString: string) => {
    try {
      const date = new Date(dateString)
      if (isNaN(date.getTime())) {
        return 'Invalid date'
      }
      return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })
    } catch (error) {
      return 'Invalid date'
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-left">
            <FileText className="h-5 w-5" />
            Document Management
          </CardTitle>
          <CardDescription className="text-left">
            View, search, and manage your ingested documents.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4 mb-6">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
              <Input
                placeholder="Search documents..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <Button onClick={loadDocuments} disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Refresh
            </Button>
            <Button 
              variant="destructive" 
              onClick={handleReset}
              disabled={isLoading || documents.length === 0}
            >
              <AlertTriangle className="mr-2 h-4 w-4" />
              Reset All
            </Button>
          </div>

          {isLoading ? (
            <div className="text-center py-8">
              <Loader2 className="mx-auto h-8 w-8 animate-spin text-gray-400" />
              <p className="mt-2 text-gray-500">Loading documents...</p>
            </div>
          ) : filteredDocuments.length === 0 ? (
            <div className="text-center py-8">
              <FileText className="mx-auto h-12 w-12 text-gray-400 mb-4" />
              <p className="text-gray-500">
                {documents.length === 0 ? 'No documents found. Start by ingesting some content.' : 'No documents match your search.'}
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-sm text-gray-600">
                Showing {filteredDocuments.length} of {documents.length} documents
              </p>
              {filteredDocuments.map((doc) => (
                <Card key={doc.id} className="p-4">
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0 text-left">
                      <h3 className="font-medium text-gray-900 truncate text-left">
                        {doc.title || `Document ${doc.id}`}
                      </h3>
                      <p className="text-sm text-gray-500 mt-1 text-left">
                        Created: {formatDate(doc.created)}
                      </p>
                      <p className="text-sm text-gray-700 mt-2 line-clamp-3 text-left">
                        {doc.content.substring(0, 200)}
                        {doc.content.length > 200 && '...'}
                      </p>
                      {doc.metadata && Object.keys(doc.metadata).length > 0 && (
                        <div className="mt-2 text-xs text-gray-500 text-left">
                          <span className="font-medium">Metadata: </span>
                          {Object.entries(doc.metadata).map(([key, value]) => (
                            <span key={key} className="mr-2 bg-gray-100 px-1 py-0.5 rounded">
                              {key}: {JSON.stringify(value)}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleDelete(doc.id)}
                      disabled={deleteLoading === doc.id}
                    >
                      {deleteLoading === doc.id ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}