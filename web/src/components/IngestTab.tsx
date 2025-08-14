import { useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Upload, FileText, Loader2 } from 'lucide-react'
import { apiClient } from '@/lib/api'

export function IngestTab() {
  const [textContent, setTextContent] = useState('')
  const [textTitle, setTextTitle] = useState('')
  const [textMetadata, setTextMetadata] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleTextSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!textContent.trim()) return

    setIsLoading(true)
    setMessage(null)

    try {
      let metadata: Record<string, any> | undefined
      if (textMetadata.trim()) {
        try {
          metadata = JSON.parse(textMetadata)
        } catch (e) {
          setMessage({ type: 'error', text: 'Invalid JSON in metadata field' })
          setIsLoading(false)
          return
        }
      }

      const response = await apiClient.ingestText(textContent, textTitle || undefined, metadata)
      if (response.error) {
        setMessage({ type: 'error', text: response.error })
      } else {
        setMessage({ type: 'success', text: 'Text ingested successfully!' })
        setTextContent('')
        setTextTitle('')
        setTextMetadata('')
      }
    } catch (error) {
      setMessage({ type: 'error', text: error instanceof Error ? error.message : 'Unknown error' })
    } finally {
      setIsLoading(false)
    }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsLoading(true)
    setMessage(null)

    try {
      let metadata: Record<string, any> | undefined
      if (textMetadata.trim()) {
        try {
          metadata = JSON.parse(textMetadata)
        } catch (e) {
          setMessage({ type: 'error', text: 'Invalid JSON in metadata field' })
          setIsLoading(false)
          return
        }
      }

      const response = await apiClient.ingestFile(file, metadata)
      if (response.error) {
        setMessage({ type: 'error', text: response.error })
      } else {
        setMessage({ type: 'success', text: `File "${file.name}" ingested successfully!` })
        setTextMetadata('')
      }
    } catch (error) {
      setMessage({ type: 'error', text: error instanceof Error ? error.message : 'Unknown error' })
    } finally {
      setIsLoading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-left">
            <FileText className="h-5 w-5" />
            Ingest Content
          </CardTitle>
          <CardDescription className="text-left">
            Add documents to your knowledge base by uploading files or entering text directly.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="text" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="text">Text Input</TabsTrigger>
              <TabsTrigger value="file">File Upload</TabsTrigger>
            </TabsList>
            
            <TabsContent value="text" className="space-y-4">
              <form onSubmit={handleTextSubmit} className="space-y-4">
                <div>
                  <label htmlFor="title" className="text-sm font-medium">
                    Title (optional)
                  </label>
                  <Input
                    id="title"
                    value={textTitle}
                    onChange={(e) => setTextTitle(e.target.value)}
                    placeholder="Document title..."
                    disabled={isLoading}
                  />
                </div>
                <div>
                  <label htmlFor="content" className="text-sm font-medium">
                    Content *
                  </label>
                  <Textarea
                    id="content"
                    value={textContent}
                    onChange={(e) => setTextContent(e.target.value)}
                    placeholder="Enter your text content here..."
                    className="min-h-[200px]"
                    disabled={isLoading}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="metadata" className="text-sm font-medium">
                    Metadata (optional, JSON format)
                  </label>
                  <Textarea
                    id="metadata"
                    value={textMetadata}
                    onChange={(e) => setTextMetadata(e.target.value)}
                    placeholder='{"author": "John Doe", "category": "technical", "tags": ["ai", "rag"]}'
                    className="min-h-[80px]"
                    disabled={isLoading}
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Enter valid JSON for document metadata (e.g., author, category, tags)
                  </p>
                </div>
                <Button type="submit" disabled={isLoading || !textContent.trim()}>
                  {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Ingest Text
                </Button>
              </form>
            </TabsContent>
            
            <TabsContent value="file" className="space-y-4">
              <div>
                <label htmlFor="file-metadata" className="text-sm font-medium">
                  Metadata (optional, JSON format)
                </label>
                <Textarea
                  id="file-metadata"
                  value={textMetadata}
                  onChange={(e) => setTextMetadata(e.target.value)}
                  placeholder='{"author": "John Doe", "category": "technical", "tags": ["ai", "rag"]}'
                  className="min-h-[80px]"
                  disabled={isLoading}
                />
                <p className="text-xs text-gray-500 mt-1">
                  Enter valid JSON for document metadata (e.g., author, category, tags)
                </p>
              </div>
              <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
                <Upload className="mx-auto h-12 w-12 text-gray-400 mb-4" />
                <div className="space-y-2">
                  <p className="text-sm font-medium">Upload a file</p>
                  <p className="text-xs text-gray-500">
                    Supports: PDF, TXT, DOC, DOCX, MD and more
                  </p>
                </div>
                <input
                  ref={fileInputRef}
                  type="file"
                  onChange={handleFileUpload}
                  className="hidden"
                  accept=".pdf,.txt,.doc,.docx,.md,.html,.rtf"
                  disabled={isLoading}
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isLoading}
                  className="mt-4"
                >
                  {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Choose File
                </Button>
              </div>
            </TabsContent>
          </Tabs>

          {message && (
            <div className={`mt-4 p-3 rounded-md text-sm ${
              message.type === 'success' 
                ? 'bg-green-50 text-green-700 border border-green-200' 
                : 'bg-red-50 text-red-700 border border-red-200'
            }`}>
              {message.text}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}