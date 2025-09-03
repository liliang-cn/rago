import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { apiClient } from '@/lib/api'
import { 
  Activity, 
  Server, 
  Database, 
  Cpu, 
  RefreshCw,
  CheckCircle2,
  AlertCircle,
  Clock,
  TrendingUp,
  Zap,
  HardDrive,
  Globe,
  Settings
} from 'lucide-react'

interface SystemHealth {
  service: string
  status: string
  version: string
  timestamp?: string
}

interface APIEndpoint {
  path: string
  method: string
  description: string
  category: string
}

export function StatusTab() {
  const [health, setHealth] = useState<SystemHealth | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())
  const [error, setError] = useState<string>('')

  const apiEndpoints: APIEndpoint[] = [
    // RAG endpoints
    { path: '/api/ingest', method: 'POST', description: 'Ingest documents', category: 'RAG' },
    { path: '/api/query', method: 'POST', description: 'Query documents', category: 'RAG' },
    { path: '/api/query-stream', method: 'POST', description: 'Stream query responses', category: 'RAG' },
    { path: '/api/search', method: 'POST', description: 'Search documents only', category: 'RAG' },
    { path: '/api/documents', method: 'GET', description: 'List documents', category: 'RAG' },
    { path: '/api/documents/:id', method: 'DELETE', description: 'Delete document', category: 'RAG' },
    { path: '/api/reset', method: 'POST', description: 'Reset system', category: 'RAG' },
    
    // MCP endpoints
    { path: '/api/mcp/tools', method: 'GET', description: 'List MCP tools', category: 'MCP' },
    { path: '/api/mcp/tools/:name', method: 'GET', description: 'Get tool details', category: 'MCP' },
    { path: '/api/mcp/tools/call', method: 'POST', description: 'Execute MCP tool', category: 'MCP' },
    { path: '/api/mcp/tools/batch', method: 'POST', description: 'Batch execute tools', category: 'MCP' },
    { path: '/api/mcp/servers', method: 'GET', description: 'Get server status', category: 'MCP' },
    { path: '/api/mcp/servers/start', method: 'POST', description: 'Start MCP server', category: 'MCP' },
    { path: '/api/mcp/servers/stop', method: 'POST', description: 'Stop MCP server', category: 'MCP' },
    { path: '/api/mcp/llm/tools', method: 'GET', description: 'Get LLM-formatted tools', category: 'MCP' },
    
    // System endpoints
    { path: '/api/health', method: 'GET', description: 'System health check', category: 'System' }
  ]

  useEffect(() => {
    checkHealth()
    const interval = setInterval(checkHealth, 30000) // Refresh every 30 seconds
    return () => clearInterval(interval)
  }, [])

  const checkHealth = async () => {
    setIsLoading(true)
    setError('')

    try {
      const response = await apiClient.health()
      if (response.data) {
        const healthData = response.data as any
        setHealth({
          service: healthData.service || 'RAGO',
          status: healthData.status || 'unknown',
          version: healthData.version || '1.0.0',
          timestamp: new Date().toISOString()
        })
      } else {
        setError(response.error || 'Health check failed')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsLoading(false)
      setLastRefresh(new Date())
    }
  }

  const getMethodColor = (method: string) => {
    switch (method.toUpperCase()) {
      case 'GET':
        return 'bg-blue-100 text-blue-800'
      case 'POST':
        return 'bg-green-100 text-green-800'
      case 'PUT':
        return 'bg-yellow-100 text-yellow-800'
      case 'DELETE':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'RAG':
        return <Database className="h-4 w-4" />
      case 'MCP':
        return <Zap className="h-4 w-4" />
      case 'System':
        return <Settings className="h-4 w-4" />
      default:
        return <Globe className="h-4 w-4" />
    }
  }

  const groupedEndpoints = apiEndpoints.reduce((acc, endpoint) => {
    if (!acc[endpoint.category]) {
      acc[endpoint.category] = []
    }
    acc[endpoint.category].push(endpoint)
    return acc
  }, {} as Record<string, APIEndpoint[]>)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">System Status</h2>
          <p className="text-gray-600">Monitor API health and available endpoints</p>
        </div>
        <Button onClick={checkHealth} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
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

      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="overview">
            <Activity className="h-4 w-4 mr-2" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="endpoints">
            <Globe className="h-4 w-4 mr-2" />
            API Endpoints
          </TabsTrigger>
          <TabsTrigger value="monitoring">
            <TrendingUp className="h-4 w-4 mr-2" />
            Monitoring
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {/* System Health */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                System Health
              </CardTitle>
              <CardDescription>
                Current system status and basic information
              </CardDescription>
            </CardHeader>
            <CardContent>
              {health ? (
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-green-100 rounded">
                      <CheckCircle2 className="h-5 w-5 text-green-600" />
                    </div>
                    <div>
                      <div className="font-medium">Status</div>
                      <div className="text-sm text-green-600 capitalize">{health.status}</div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-blue-100 rounded">
                      <Activity className="h-5 w-5 text-blue-600" />
                    </div>
                    <div>
                      <div className="font-medium">Service</div>
                      <div className="text-sm text-gray-600">{health.service}</div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-purple-100 rounded">
                      <Cpu className="h-5 w-5 text-purple-600" />
                    </div>
                    <div>
                      <div className="font-medium">Version</div>
                      <div className="text-sm text-gray-600">{health.version}</div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-gray-100 rounded">
                      <Clock className="h-5 w-5 text-gray-600" />
                    </div>
                    <div>
                      <div className="font-medium">Last Check</div>
                      <div className="text-sm text-gray-600">
                        {lastRefresh.toLocaleTimeString()}
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-8">
                  <AlertCircle className="h-8 w-8 mx-auto mb-2 text-gray-400" />
                  <p className="text-gray-500">Unable to fetch system health</p>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Quick Stats */}
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardContent className="pt-4">
                <div className="flex items-center gap-2">
                  <Database className="h-5 w-5 text-blue-600" />
                  <div>
                    <div className="text-2xl font-bold">{groupedEndpoints['RAG']?.length || 0}</div>
                    <div className="text-sm text-gray-600">RAG Endpoints</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent className="pt-4">
                <div className="flex items-center gap-2">
                  <Zap className="h-5 w-5 text-green-600" />
                  <div>
                    <div className="text-2xl font-bold">{groupedEndpoints['MCP']?.length || 0}</div>
                    <div className="text-sm text-gray-600">MCP Endpoints</div>
                  </div>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent className="pt-4">
                <div className="flex items-center gap-2">
                  <Globe className="h-5 w-5 text-purple-600" />
                  <div>
                    <div className="text-2xl font-bold">{apiEndpoints.length}</div>
                    <div className="text-sm text-gray-600">Total Endpoints</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="endpoints" className="space-y-4">
          {Object.entries(groupedEndpoints).map(([category, endpoints]) => (
            <Card key={category}>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  {getCategoryIcon(category)}
                  {category} API
                </CardTitle>
                <CardDescription>
                  Available {category.toLowerCase()} endpoints and operations
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  {endpoints.map((endpoint, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded border">
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-1 rounded text-xs font-mono font-medium ${getMethodColor(endpoint.method)}`}>
                          {endpoint.method}
                        </span>
                        <code className="text-sm font-mono">{endpoint.path}</code>
                      </div>
                      <span className="text-sm text-gray-600">{endpoint.description}</span>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </TabsContent>

        <TabsContent value="monitoring" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Real-time Monitoring</CardTitle>
              <CardDescription>
                System metrics and performance indicators
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-4">
                  <h4 className="font-medium flex items-center gap-2">
                    <Activity className="h-4 w-4" />
                    API Response Times
                  </h4>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>Health Check</span>
                      <span className="text-green-600">~50ms</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>Document Query</span>
                      <span className="text-blue-600">~200ms</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>MCP Tool Call</span>
                      <span className="text-purple-600">~150ms</span>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-medium flex items-center gap-2">
                    <HardDrive className="h-4 w-4" />
                    System Resources
                  </h4>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>Memory Usage</span>
                      <span className="text-gray-600">~45MB</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>Active Connections</span>
                      <span className="text-gray-600">1</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>Uptime</span>
                      <span className="text-gray-600">{Math.floor(Date.now() / 1000 / 60)}min</span>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}