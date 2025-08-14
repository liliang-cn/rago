import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { IngestTab } from '@/components/IngestTab'
import { ChatTab } from '@/components/ChatTab'
import { DocumentsTab } from '@/components/DocumentsTab'
import { Database, MessageSquare, FileText, Github } from 'lucide-react'
import './App.css'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Database className="h-8 w-8 text-blue-600" />
              <div className="text-left">
                <h1 className="text-2xl font-bold text-gray-900 text-left">RAGO</h1>
                <p className="text-sm text-gray-600 text-left">Local RAG based on Ollama</p>
              </div>
            </div>
            <a
              href="https://github.com/liliang-cn/rago"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 text-gray-600 hover:text-gray-900 transition-colors"
            >
              <Github className="h-5 w-5" />
              <span className="text-sm">GitHub</span>
            </a>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <Tabs defaultValue="chat" className="w-full">
          <TabsList className="grid w-full grid-cols-3 mb-8">
            <TabsTrigger value="chat" className="flex items-center gap-2">
              <MessageSquare className="h-4 w-4" />
              Chat
            </TabsTrigger>
            <TabsTrigger value="ingest" className="flex items-center gap-2">
              <FileText className="h-4 w-4" />
              Ingest
            </TabsTrigger>
            <TabsTrigger value="documents" className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              Documents
            </TabsTrigger>
          </TabsList>
          
          <TabsContent value="chat">
            <ChatTab />
          </TabsContent>
          
          <TabsContent value="ingest">
            <IngestTab />
          </TabsContent>
          
          <TabsContent value="documents">
            <DocumentsTab />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  )
}

export default App