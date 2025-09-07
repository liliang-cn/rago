import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LLMTab } from '@/components/LLMTab'
import { RAGTab } from '@/components/RAGTab'
import { MCPTab } from '@/components/MCPTab'
import { AgentsTab } from '@/components/AgentsTab'
import { StatusTab } from '@/components/StatusTab'
import { Brain, Database, Wrench, Bot, Activity, Layers, Github } from 'lucide-react'
import './App.css'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Layers className="h-8 w-8 text-blue-600" />
              <div className="text-left">
                <h1 className="text-2xl font-bold text-gray-900 text-left">RAGO v3</h1>
                <p className="text-sm text-gray-600 text-left">Four-Pillar AI Foundation</p>
              </div>
            </div>
            <div className="flex items-center gap-6">
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <span className="flex items-center gap-1">
                  <Brain className="h-3 w-3" /> LLM
                </span>
                <span className="text-gray-300">•</span>
                <span className="flex items-center gap-1">
                  <Database className="h-3 w-3" /> RAG
                </span>
                <span className="text-gray-300">•</span>
                <span className="flex items-center gap-1">
                  <Wrench className="h-3 w-3" /> MCP
                </span>
                <span className="text-gray-300">•</span>
                <span className="flex items-center gap-1">
                  <Bot className="h-3 w-3" /> Agents
                </span>
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
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <h2 className="text-sm font-semibold text-blue-900 mb-1">V3 Architecture</h2>
          <p className="text-xs text-blue-700">
            RAGO V3 features a unified four-pillar architecture where LLM, RAG, MCP, and Agents work independently or together.
            Each pillar provides specific capabilities that can be composed for complex AI applications.
          </p>
        </div>

        <Tabs defaultValue="llm" className="w-full">
          <TabsList className="grid w-full grid-cols-5 mb-8">
            <TabsTrigger value="llm" className="flex items-center gap-2">
              <Brain className="h-4 w-4" />
              LLM
            </TabsTrigger>
            <TabsTrigger value="rag" className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              RAG
            </TabsTrigger>
            <TabsTrigger value="mcp" className="flex items-center gap-2">
              <Wrench className="h-4 w-4" />
              MCP
            </TabsTrigger>
            <TabsTrigger value="agents" className="flex items-center gap-2">
              <Bot className="h-4 w-4" />
              Agents
            </TabsTrigger>
            <TabsTrigger value="status" className="flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Status
            </TabsTrigger>
          </TabsList>
          
          <TabsContent value="llm">
            <LLMTab />
          </TabsContent>
          
          <TabsContent value="rag">
            <RAGTab />
          </TabsContent>
          
          <TabsContent value="mcp">
            <MCPTab />
          </TabsContent>
          
          <TabsContent value="agents">
            <AgentsTab />
          </TabsContent>
          
          <TabsContent value="status">
            <StatusTab />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  )
}

export default App