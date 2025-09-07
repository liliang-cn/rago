import { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Database, 
  Search, 
  Upload, 
  FileText, 
  Trash2, 
  RefreshCw,
  File,
  AlertCircle
} from 'lucide-react'
import { api, Document, RAGResult, useAsyncOperation } from '@/lib/api'

export function RAGTab() {
  const [documents, setDocuments] = useState<Document[]>([])
  const [query, setQuery] = useState('')
  const [topK, setTopK] = useState(5)
  const [queryResult, setQueryResult] = useState<RAGResult | null>(null)
  const [textContent, setTextContent] = useState('')
  const [documentId, setDocumentId] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const documentsOp = useAsyncOperation<Document[]>()
  const ingestOp = useAsyncOperation<any>()
  const queryOp = useAsyncOperation<RAGResult>()
  const deleteOp = useAsyncOperation<any>()
  const resetOp = useAsyncOperation<any>()

  useEffect(() => {
    loadDocuments()
  }, [])

  const loadDocuments = async () => {
    const result = await documentsOp.execute(() => api.listDocuments())
    if (result.data) {
      setDocuments(result.data)
    }
  }

  const handleTextIngest = async () => {
    if (!textContent.trim() || !documentId.trim()) return

    const result = await ingestOp.execute(() =>
      api.ingestDocument({
        id: documentId,
        content: textContent,
        metadata: { source: 'manual', timestamp: new Date().toISOString() },
      })
    )

    if (result.success) {
      setTextContent('')
      setDocumentId('')
      loadDocuments()
    }
  }

  const handleFileIngest = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const result = await ingestOp.execute(() =>
      api.ingestFile(file, {
        filename: file.name,
        size: file.size,
        type: file.type,
        timestamp: new Date().toISOString(),
      })
    )

    if (result.success) {
      loadDocuments()
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const handleQuery = async () => {
    if (!query.trim()) return

    const result = await queryOp.execute(() =>
      api.queryRAG({
        query,
        top_k: topK,
      })
    )

    if (result.data) {
      setQueryResult(result.data)
    }
  }

  const handleDelete = async (id: string) => {
    const result = await deleteOp.execute(() => api.deleteDocument(id))
    if (result.success) {
      loadDocuments()
    }
  }

  const handleReset = async () => {
    if (!confirm('Are you sure you want to reset the RAG system? This will delete all documents.')) {
      return
    }

    const result = await resetOp.execute(() => api.resetRAG())
    if (result.success) {
      setDocuments([])
      setQueryResult(null)
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            RAG Pillar
          </CardTitle>
          <CardDescription>
            Local document storage with vector and keyword search capabilities
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">{documents.length}</p>
                <p className="text-xs text-gray-500">Documents</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Database className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">SQLite + Bleve</p>
                <p className="text-xs text-gray-500">Storage Backend</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Search className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">Hybrid</p>
                <p className="text-xs text-gray-500">Search Mode</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="ingest" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="ingest">Ingest</TabsTrigger>
          <TabsTrigger value="query">Query</TabsTrigger>
          <TabsTrigger value="documents">Documents</TabsTrigger>
        </TabsList>

        <TabsContent value="ingest" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Document Ingestion</CardTitle>
              <CardDescription>
                Add documents to your RAG knowledge base
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Text Ingestion */}
              <div className="space-y-4">
                <h3 className="text-sm font-medium">Text Content</h3>
                <div>
                  <Label>Document ID</Label>
                  <Input
                    value={documentId}
                    onChange={(e) => setDocumentId(e.target.value)}
                    placeholder="e.g., doc-001"
                    className="mt-2"
                  />
                </div>
                <div>
                  <Label>Content</Label>
                  <Textarea
                    value={textContent}
                    onChange={(e) => setTextContent(e.target.value)}
                    placeholder="Enter document content..."
                    className="min-h-[150px] mt-2"
                  />
                </div>
                <Button
                  onClick={handleTextIngest}
                  disabled={ingestOp.loading || !textContent.trim() || !documentId.trim()}
                  className="w-full"
                >
                  <Upload className="h-4 w-4 mr-2" />
                  {ingestOp.loading ? 'Ingesting...' : 'Ingest Text'}
                </Button>
              </div>

              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-white px-2 text-gray-500">Or</span>
                </div>
              </div>

              {/* File Ingestion */}
              <div className="space-y-4">
                <h3 className="text-sm font-medium">File Upload</h3>
                <div className="border-2 border-dashed border-gray-200 rounded-lg p-6 text-center">
                  <File className="h-8 w-8 mx-auto text-gray-400 mb-2" />
                  <p className="text-sm text-gray-600 mb-2">
                    Upload PDF, TXT, or Markdown files
                  </p>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".pdf,.txt,.md"
                    onChange={handleFileIngest}
                    className="hidden"
                  />
                  <Button
                    variant="outline"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={ingestOp.loading}
                  >
                    Choose File
                  </Button>
                </div>
              </div>

              {ingestOp.error && (
                <Alert variant="destructive">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>{ingestOp.error}</AlertDescription>
                </Alert>
              )}

              {ingestOp.data && (
                <Alert>
                  <AlertDescription>Document ingested successfully!</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="query" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Semantic Search</CardTitle>
              <CardDescription>
                Query your knowledge base using natural language
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Query</Label>
                <Textarea
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Ask a question about your documents..."
                  className="min-h-[80px] mt-2"
                />
              </div>

              <div>
                <Label>Top K Results: {topK}</Label>
                <Input
                  type="range"
                  min="1"
                  max="20"
                  value={topK}
                  onChange={(e) => setTopK(parseInt(e.target.value))}
                  className="mt-2"
                />
              </div>

              <Button
                onClick={handleQuery}
                disabled={queryOp.loading || !query.trim()}
                className="w-full"
              >
                <Search className="h-4 w-4 mr-2" />
                {queryOp.loading ? 'Searching...' : 'Search'}
              </Button>

              {queryOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{queryOp.error}</AlertDescription>
                </Alert>
              )}

              {queryResult && (
                <div className="space-y-4">
                  {queryResult.answer && (
                    <Card className="bg-blue-50 border-blue-200">
                      <CardHeader>
                        <CardTitle className="text-sm">Generated Answer</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <p className="text-sm">{queryResult.answer}</p>
                      </CardContent>
                    </Card>
                  )}

                  <div>
                    <h3 className="text-sm font-medium mb-2">
                      Retrieved Documents ({queryResult.documents.length})
                    </h3>
                    <div className="space-y-2">
                      {queryResult.documents.map((doc, idx) => (
                        <Card key={idx} className="bg-gray-50">
                          <CardContent className="pt-4">
                            <div className="flex justify-between items-start mb-2">
                              <Badge variant="outline">{doc.id}</Badge>
                              {doc.metadata?.score && (
                                <span className="text-xs text-gray-500">
                                  Score: {doc.metadata.score.toFixed(3)}
                                </span>
                              )}
                            </div>
                            <p className="text-sm text-gray-700 line-clamp-3">
                              {doc.content}
                            </p>
                          </CardContent>
                        </Card>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="documents" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Document Management</CardTitle>
              <CardDescription>
                View and manage ingested documents
              </CardDescription>
              <div className="flex gap-2 mt-4">
                <Button onClick={loadDocuments} variant="outline" size="sm">
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Refresh
                </Button>
                <Button onClick={handleReset} variant="destructive" size="sm">
                  <Trash2 className="h-4 w-4 mr-2" />
                  Reset RAG
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {documentsOp.loading ? (
                <p className="text-center text-gray-500">Loading documents...</p>
              ) : documents.length === 0 ? (
                <p className="text-center text-gray-500">No documents ingested yet</p>
              ) : (
                <div className="space-y-2">
                  {documents.map((doc) => (
                    <Card key={doc.id} className="bg-gray-50">
                      <CardContent className="pt-4">
                        <div className="flex justify-between items-start">
                          <div className="flex-1">
                            <Badge className="mb-2">{doc.id}</Badge>
                            <p className="text-sm text-gray-700 line-clamp-2">
                              {doc.content}
                            </p>
                            {doc.metadata && (
                              <div className="mt-2 flex gap-2 flex-wrap">
                                {Object.entries(doc.metadata).map(([key, value]) => (
                                  <span key={key} className="text-xs text-gray-500">
                                    {key}: {String(value)}
                                  </span>
                                ))}
                              </div>
                            )}
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(doc.id)}
                            disabled={deleteOp.loading}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}

              {documentsOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{documentsOp.error}</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}