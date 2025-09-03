import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { apiClient, MCPTool, MCPToolResult } from '@/lib/api'
import { 
  Wrench, 
  Server, 
  Play, 
  Square, 
  RefreshCw, 
  AlertCircle, 
  CheckCircle2,
  Clock,
  Database,
  Activity
} from 'lucide-react'

interface ServerStatus {
  name: string
  status: boolean
}

export function MCPTab() {
  const [tools, setTools] = useState<MCPTool[]>([])
  const [servers, setServers] = useState<ServerStatus[]>([])
  const [selectedTool, setSelectedTool] = useState<MCPTool | null>(null)
  const [toolArgs, setToolArgs] = useState<string>('')
  const [toolResult, setToolResult] = useState<MCPToolResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingTools, setIsLoadingTools] = useState(true)
  const [error, setError] = useState<string>('')

  useEffect(() => {
    loadMCPData()
  }, [])

  const loadMCPData = async () => {
    setIsLoadingTools(true)
    setError('')

    try {
      // Load tools and servers in parallel
      const [toolsResponse, serversResponse] = await Promise.all([
        apiClient.getMCPTools(),
        apiClient.getMCPServers()
      ])

      if (toolsResponse.data) {
        setTools(toolsResponse.data.tools)
      }

      if (serversResponse.data) {
        const serverList = Object.entries(serversResponse.data.servers).map(([name, status]) => ({
          name,
          status: Boolean(status)
        }))
        setServers(serverList)
      }
    } catch (err) {
      setError('Failed to load MCP data')
    } finally {
      setIsLoadingTools(false)
    }
  }

  const executeTool = async () => {
    if (!selectedTool) return

    setIsLoading(true)
    setError('')
    setToolResult(null)

    try {
      let args = {}
      if (toolArgs.trim()) {
        args = JSON.parse(toolArgs)
      }

      const response = await apiClient.callMCPTool(selectedTool.name, args)
      
      if (response.data) {
        setToolResult(response.data)
      } else {
        setError(response.error || 'Failed to execute tool')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid JSON arguments')
    } finally {
      setIsLoading(false)
    }
  }

  const startServer = async (serverName: string) => {
    try {
      await apiClient.startMCPServer(serverName)
      await loadMCPData() // Refresh data
    } catch (err) {
      setError(`Failed to start server: ${serverName}`)
    }
  }

  const stopServer = async (serverName: string) => {
    try {
      await apiClient.stopMCPServer(serverName)
      await loadMCPData() // Refresh data
    } catch (err) {
      setError(`Failed to stop server: ${serverName}`)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">MCP Integration</h2>
          <p className="text-gray-600">Manage Model Context Protocol servers and tools</p>
        </div>
        <Button onClick={loadMCPData} disabled={isLoadingTools}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoadingTools ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      {error && (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-4">
            <div className="flex items-center gap-2 text-red-800">
              <AlertCircle className="h-4 w-4" />
              <span>{error}</span>
            </div>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue="tools" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="tools">
            <Wrench className="h-4 w-4 mr-2" />
            Tools ({tools.length})
          </TabsTrigger>
          <TabsTrigger value="servers">
            <Server className="h-4 w-4 mr-2" />
            Servers ({servers.length})
          </TabsTrigger>
          <TabsTrigger value="execute">
            <Play className="h-4 w-4 mr-2" />
            Execute
          </TabsTrigger>
        </TabsList>

        <TabsContent value="tools" className="space-y-4">
          {isLoadingTools ? (
            <div className="flex items-center justify-center h-32">
              <RefreshCw className="h-6 w-6 animate-spin mr-2" />
              <span>Loading tools...</span>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {tools.map((tool) => (
                <Card key={tool.name} className="cursor-pointer hover:shadow-md transition-shadow"
                      onClick={() => setSelectedTool(tool)}>
                  <CardHeader className="pb-3">
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-sm font-medium truncate">
                        {tool.name.replace('mcp_', '').replace(/_/g, ' ')}
                      </CardTitle>
                      <div className="flex items-center gap-1 text-xs text-gray-500">
                        <Database className="h-3 w-3" />
                        {tool.server_name}
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <CardDescription className="text-xs line-clamp-2">
                      {tool.description}
                    </CardDescription>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="servers" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {servers.map((server) => (
              <Card key={server.name}>
                <CardHeader className="flex flex-row items-center justify-between pb-3">
                  <div className="flex items-center gap-2">
                    <Server className="h-5 w-5" />
                    <CardTitle className="text-lg">{server.name}</CardTitle>
                  </div>
                  <div className="flex items-center gap-2">
                    {server.status ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600" />
                    ) : (
                      <AlertCircle className="h-4 w-4 text-red-600" />
                    )}
                    <span className={`text-sm font-medium ${
                      server.status ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {server.status ? 'Connected' : 'Disconnected'}
                    </span>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant={server.status ? "secondary" : "default"}
                      onClick={() => startServer(server.name)}
                      disabled={server.status}
                    >
                      <Play className="h-3 w-3 mr-1" />
                      Start
                    </Button>
                    <Button
                      size="sm"
                      variant={server.status ? "destructive" : "secondary"}
                      onClick={() => stopServer(server.name)}
                      disabled={!server.status}
                    >
                      <Square className="h-3 w-3 mr-1" />
                      Stop
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="execute" className="space-y-4">
          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Execute Tool</CardTitle>
                <CardDescription>
                  Select a tool and provide arguments to execute
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="text-sm font-medium">Selected Tool:</label>
                  <div className="mt-1 p-2 bg-gray-50 rounded border text-sm">
                    {selectedTool ? selectedTool.name : 'No tool selected'}
                  </div>
                </div>

                <div>
                  <label className="text-sm font-medium">Arguments (JSON):</label>
                  <Textarea
                    placeholder='{"query": "SELECT * FROM table", "limit": 10}'
                    value={toolArgs}
                    onChange={(e) => setToolArgs(e.target.value)}
                    className="mt-1"
                    rows={4}
                  />
                </div>

                <Button 
                  onClick={executeTool} 
                  disabled={!selectedTool || isLoading}
                  className="w-full"
                >
                  {isLoading ? (
                    <>
                      <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                      Executing...
                    </>
                  ) : (
                    <>
                      <Play className="h-4 w-4 mr-2" />
                      Execute Tool
                    </>
                  )}
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Result</CardTitle>
                <CardDescription>
                  Tool execution result will appear here
                </CardDescription>
              </CardHeader>
              <CardContent>
                {toolResult ? (
                  <div className="space-y-3">
                    <div className="flex items-center gap-2">
                      {toolResult.success ? (
                        <CheckCircle2 className="h-4 w-4 text-green-600" />
                      ) : (
                        <AlertCircle className="h-4 w-4 text-red-600" />
                      )}
                      <span className={`text-sm font-medium ${
                        toolResult.success ? 'text-green-600' : 'text-red-600'
                      }`}>
                        {toolResult.success ? 'Success' : 'Failed'}
                      </span>
                      <div className="flex items-center gap-1 text-xs text-gray-500 ml-auto">
                        <Clock className="h-3 w-3" />
                        {toolResult.duration}ms
                      </div>
                    </div>
                    
                    <div className="bg-gray-50 p-3 rounded border">
                      <pre className="text-xs overflow-auto whitespace-pre-wrap">
                        {toolResult.success 
                          ? JSON.stringify(toolResult.data, null, 2)
                          : toolResult.error
                        }
                      </pre>
                    </div>
                  </div>
                ) : (
                  <div className="text-center text-gray-500 py-8">
                    <Activity className="h-8 w-8 mx-auto mb-2 opacity-50" />
                    <p className="text-sm">No execution result yet</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}