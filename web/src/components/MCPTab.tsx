import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { 
  Wrench, 
  Server, 
  Play, 
  Square, 
  RefreshCw, 
  AlertCircle, 
  CheckCircle,
  Zap
} from 'lucide-react'
import { api, MCPTool, MCPServer, useAsyncOperation } from '@/lib/api'

export function MCPTab() {
  const [servers, setServers] = useState<MCPServer[]>([])
  const [tools, setTools] = useState<MCPTool[]>([])
  const [selectedServer, setSelectedServer] = useState<string>('')
  const [selectedTool, setSelectedTool] = useState<string>('')
  const [toolParams, setToolParams] = useState<string>('{}')
  const [toolResult, setToolResult] = useState<any>(null)

  const serversOp = useAsyncOperation<MCPServer[]>()
  const toolsOp = useAsyncOperation<MCPTool[]>()
  const callToolOp = useAsyncOperation<any>()
  const startServerOp = useAsyncOperation<any>()
  const stopServerOp = useAsyncOperation<any>()

  useEffect(() => {
    loadServers()
    loadTools()
  }, [])

  const loadServers = async () => {
    const result = await serversOp.execute(() => api.listMCPServers())
    if (result.data) {
      setServers(result.data)
      if (result.data.length > 0 && !selectedServer) {
        setSelectedServer(result.data[0].name)
      }
    }
  }

  const loadTools = async () => {
    const result = await toolsOp.execute(() => api.listMCPTools())
    if (result.data) {
      setTools(result.data)
    }
  }

  const handleCallTool = async () => {
    if (!selectedServer || !selectedTool) return

    try {
      const params = JSON.parse(toolParams)
      const result = await callToolOp.execute(() =>
        api.callMCPTool(selectedServer, selectedTool, params)
      )
      
      if (result.data) {
        setToolResult(result.data)
      }
    } catch (error) {
      alert('Invalid JSON parameters')
    }
  }

  const handleStartServer = async (name: string) => {
    const result = await startServerOp.execute(() => api.startMCPServer(name))
    if (result.success) {
      loadServers()
    }
  }

  const handleStopServer = async (name: string) => {
    const result = await stopServerOp.execute(() => api.stopMCPServer(name))
    if (result.success) {
      loadServers()
    }
  }

  const getServerStatusIcon = (status?: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'stopped':
        return <Square className="h-4 w-4 text-gray-500" />
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-gray-500" />
    }
  }

  const getServerStatusBadge = (status?: string) => {
    switch (status) {
      case 'running':
        return <Badge className="bg-green-500 text-white">Running</Badge>
      case 'stopped':
        return <Badge variant="secondary">Stopped</Badge>
      case 'error':
        return <Badge variant="destructive">Error</Badge>
      default:
        return <Badge variant="outline">Unknown</Badge>
    }
  }

  const serverTools = selectedServer 
    ? tools.filter(t => t.server === selectedServer)
    : []

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Wrench className="h-5 w-5" />
            MCP Pillar
          </CardTitle>
          <CardDescription>
            Model Context Protocol - Tool integration and external services
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-3 gap-4">
            <div className="flex items-center gap-2">
              <Server className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">{servers.length}</p>
                <p className="text-xs text-gray-500">MCP Servers</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Wrench className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">{tools.length}</p>
                <p className="text-xs text-gray-500">Available Tools</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Zap className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">
                  {servers.filter(s => s.status === 'running').length}
                </p>
                <p className="text-xs text-gray-500">Active Servers</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="servers" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="servers">Servers</TabsTrigger>
          <TabsTrigger value="tools">Tools</TabsTrigger>
        </TabsList>

        <TabsContent value="servers" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg">MCP Servers</CardTitle>
                  <CardDescription>
                    Manage MCP server connections
                  </CardDescription>
                </div>
                <Button
                  onClick={loadServers}
                  variant="outline"
                  size="sm"
                  disabled={serversOp.loading}
                >
                  <RefreshCw className={`h-4 w-4 ${serversOp.loading ? 'animate-spin' : ''}`} />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {serversOp.loading ? (
                <p className="text-center text-gray-500">Loading servers...</p>
              ) : servers.length === 0 ? (
                <p className="text-center text-gray-500">No MCP servers configured</p>
              ) : (
                <div className="space-y-3">
                  {servers.map((server) => (
                    <Card key={server.name} className="bg-gray-50">
                      <CardContent className="pt-4">
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-2">
                              {getServerStatusIcon(server.status)}
                              <span className="font-medium">{server.name}</span>
                              {getServerStatusBadge(server.status)}
                            </div>
                            <p className="text-sm text-gray-600 mb-2">{server.description}</p>
                            <div className="text-xs text-gray-500 font-mono">
                              {server.command} {server.args?.join(' ')}
                            </div>
                          </div>
                          <div className="flex gap-2">
                            {server.status === 'running' ? (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleStopServer(server.name)}
                                disabled={stopServerOp.loading}
                              >
                                <Square className="h-3 w-3 mr-1" />
                                Stop
                              </Button>
                            ) : (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleStartServer(server.name)}
                                disabled={startServerOp.loading}
                              >
                                <Play className="h-3 w-3 mr-1" />
                                Start
                              </Button>
                            )}
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}

              {serversOp.error && (
                <Alert variant="destructive" className="mt-4">
                  <AlertDescription>{serversOp.error}</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="tools" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Tool Execution</CardTitle>
              <CardDescription>
                Execute MCP tools with custom parameters
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Server</Label>
                <Select value={selectedServer} onValueChange={setSelectedServer}>
                  <SelectTrigger className="mt-2">
                    <SelectValue placeholder="Select server" />
                  </SelectTrigger>
                  <SelectContent>
                    {servers
                      .filter(s => s.status === 'running')
                      .map((server) => (
                        <SelectItem key={server.name} value={server.name}>
                          {server.name}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
              </div>

              {selectedServer && (
                <div>
                  <Label>Tool</Label>
                  <Select value={selectedTool} onValueChange={setSelectedTool}>
                    <SelectTrigger className="mt-2">
                      <SelectValue placeholder="Select tool" />
                    </SelectTrigger>
                    <SelectContent>
                      {serverTools.map((tool) => (
                        <SelectItem key={tool.name} value={tool.name}>
                          {tool.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {selectedTool && (
                    <p className="text-xs text-gray-600 mt-2">
                      {serverTools.find(t => t.name === selectedTool)?.description}
                    </p>
                  )}
                </div>
              )}

              {selectedTool && (
                <div>
                  <Label>Parameters (JSON)</Label>
                  <Textarea
                    value={toolParams}
                    onChange={(e) => setToolParams(e.target.value)}
                    placeholder='{"param1": "value1"}'
                    className="font-mono text-sm min-h-[100px] mt-2"
                  />
                </div>
              )}

              <Button
                onClick={handleCallTool}
                disabled={callToolOp.loading || !selectedServer || !selectedTool}
                className="w-full"
              >
                <Play className="h-4 w-4 mr-2" />
                {callToolOp.loading ? 'Executing...' : 'Execute Tool'}
              </Button>

              {callToolOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{callToolOp.error}</AlertDescription>
                </Alert>
              )}

              {toolResult && (
                <Card className="bg-gray-50">
                  <CardHeader>
                    <CardTitle className="text-sm">Result</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <pre className="text-xs overflow-x-auto">
                      {JSON.stringify(toolResult, null, 2)}
                    </pre>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>

          {/* Tool List */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Available Tools</CardTitle>
              <CardDescription>
                All tools across MCP servers
              </CardDescription>
            </CardHeader>
            <CardContent>
              {toolsOp.loading ? (
                <p className="text-center text-gray-500">Loading tools...</p>
              ) : tools.length === 0 ? (
                <p className="text-center text-gray-500">No tools available</p>
              ) : (
                <div className="space-y-2">
                  {tools.map((tool) => (
                    <div key={`${tool.server}-${tool.name}`} className="border rounded-lg p-3">
                      <div className="flex items-center justify-between mb-1">
                        <span className="font-medium text-sm">{tool.name}</span>
                        <Badge variant="outline" className="text-xs">
                          {tool.server}
                        </Badge>
                      </div>
                      <p className="text-xs text-gray-600">{tool.description}</p>
                      {tool.parameters && (
                        <details className="mt-2">
                          <summary className="text-xs text-gray-500 cursor-pointer">
                            Parameters
                          </summary>
                          <pre className="text-xs mt-1 p-2 bg-gray-100 rounded overflow-x-auto">
                            {JSON.stringify(tool.parameters, null, 2)}
                          </pre>
                        </details>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}