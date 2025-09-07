import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Activity, 
  RefreshCw, 
  CheckCircle, 
  XCircle, 
  AlertCircle,
  Brain,
  Database,
  Wrench,
  Bot,
  Layers
} from 'lucide-react'
import { api, HealthStatus, useAsyncOperation } from '@/lib/api'

export function StatusTab() {
  const [health, setHealth] = useState<HealthStatus | null>(null)
  const [config, setConfig] = useState<any>(null)
  const [lastCheck, setLastCheck] = useState<Date | null>(null)
  
  const healthOp = useAsyncOperation<HealthStatus>()
  const configOp = useAsyncOperation<any>()

  useEffect(() => {
    checkHealth()
    loadConfig()
  }, [])

  const checkHealth = async () => {
    const result = await healthOp.execute(() => api.getHealth())
    if (result.data) {
      setHealth(result.data)
      setLastCheck(new Date())
    }
  }

  const loadConfig = async () => {
    const result = await configOp.execute(() => api.getConfig())
    if (result.data) {
      setConfig(result.data)
    }
  }

  const getHealthIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case 'degraded':
        return <AlertCircle className="h-5 w-5 text-yellow-500" />
      case 'unhealthy':
        return <XCircle className="h-5 w-5 text-red-500" />
      default:
        return <AlertCircle className="h-5 w-5 text-gray-500" />
    }
  }

  const getHealthBadgeColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'bg-green-500'
      case 'degraded':
        return 'bg-yellow-500'
      case 'unhealthy':
        return 'bg-red-500'
      default:
        return 'bg-gray-500'
    }
  }


  return (
    <div className="space-y-6">
      {/* Overall Status */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                System Status
              </CardTitle>
              <CardDescription>
                RAGO V3 Four-Pillar Architecture Health
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              {lastCheck && (
                <span className="text-xs text-gray-500">
                  Last check: {lastCheck.toLocaleTimeString()}
                </span>
              )}
              <Button
                onClick={checkHealth}
                variant="outline"
                size="sm"
                disabled={healthOp.loading}
              >
                <RefreshCw className={`h-4 w-4 ${healthOp.loading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {healthOp.error && (
            <Alert variant="destructive" className="mb-4">
              <AlertDescription>Failed to fetch health status: {healthOp.error}</AlertDescription>
            </Alert>
          )}

          {health && (
            <div className="space-y-6">
              {/* Overall Health */}
              <div className="flex items-center gap-4 p-4 bg-gray-50 rounded-lg">
                {getHealthIcon(health.overall)}
                <div className="flex-1">
                  <p className="font-medium">Overall System Health</p>
                  <p className="text-sm text-gray-600">
                    {health.overall === 'healthy' 
                      ? 'All systems operational'
                      : health.overall === 'degraded'
                      ? 'Some services are experiencing issues'
                      : 'System is experiencing problems'}
                  </p>
                </div>
                <Badge className={`${getHealthBadgeColor(health.overall)} text-white`}>
                  {health.overall.toUpperCase()}
                </Badge>
              </div>

              {/* Four Pillars Status */}
              <div>
                <h3 className="text-sm font-medium mb-3">Four Pillars</h3>
                <div className="grid grid-cols-2 gap-3">
                  {/* LLM Pillar */}
                  <Card className="bg-gray-50">
                    <CardContent className="pt-4">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <Brain className="h-4 w-4 text-blue-500" />
                          <span className="font-medium">LLM</span>
                        </div>
                        {health.providers && Object.keys(health.providers).length > 0 ? (
                          <CheckCircle className="h-4 w-4 text-green-500" />
                        ) : (
                          <XCircle className="h-4 w-4 text-red-500" />
                        )}
                      </div>
                      <div className="space-y-1">
                        {health.providers && Object.entries(health.providers).map(([name, status]) => (
                          <div key={name} className="flex items-center justify-between text-xs">
                            <span className="text-gray-600">{name}</span>
                            <Badge 
                              variant={status === 'healthy' ? 'default' : 'destructive'}
                              className="h-4 text-xs px-1"
                            >
                              {status}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>

                  {/* RAG Pillar */}
                  <Card className="bg-gray-50">
                    <CardContent className="pt-4">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <Database className="h-4 w-4 text-green-500" />
                          <span className="font-medium">RAG</span>
                        </div>
                        {health.rag === 'healthy' ? (
                          <CheckCircle className="h-4 w-4 text-green-500" />
                        ) : (
                          <XCircle className="h-4 w-4 text-red-500" />
                        )}
                      </div>
                      <div className="text-xs text-gray-600">
                        <p>Vector Store: SQLite</p>
                        <p>Keyword Store: Bleve</p>
                        <p>Status: {health.rag}</p>
                      </div>
                    </CardContent>
                  </Card>

                  {/* MCP Pillar */}
                  <Card className="bg-gray-50">
                    <CardContent className="pt-4">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <Wrench className="h-4 w-4 text-purple-500" />
                          <span className="font-medium">MCP</span>
                        </div>
                        {health.mcp?.status === 'healthy' ? (
                          <CheckCircle className="h-4 w-4 text-green-500" />
                        ) : (
                          <XCircle className="h-4 w-4 text-red-500" />
                        )}
                      </div>
                      <div className="space-y-1">
                        {health.mcp?.servers && Object.entries(health.mcp.servers).map(([name, status]) => (
                          <div key={name} className="flex items-center justify-between text-xs">
                            <span className="text-gray-600">{name}</span>
                            <Badge 
                              variant={status === 'running' ? 'default' : 'secondary'}
                              className="h-4 text-xs px-1"
                            >
                              {status}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>

                  {/* Agents Pillar */}
                  <Card className="bg-gray-50">
                    <CardContent className="pt-4">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <Bot className="h-4 w-4 text-orange-500" />
                          <span className="font-medium">Agents</span>
                        </div>
                        {health.agents === 'healthy' ? (
                          <CheckCircle className="h-4 w-4 text-green-500" />
                        ) : (
                          <XCircle className="h-4 w-4 text-red-500" />
                        )}
                      </div>
                      <div className="text-xs text-gray-600">
                        <p>Workflow Engine: Active</p>
                        <p>Scheduler: Memory Backend</p>
                        <p>Status: {health.agents}</p>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Layers className="h-5 w-5" />
            Configuration
          </CardTitle>
          <CardDescription>
            Current system configuration
          </CardDescription>
        </CardHeader>
        <CardContent>
          {configOp.loading ? (
            <p className="text-center text-gray-500">Loading configuration...</p>
          ) : config ? (
            <div className="space-y-4">
              <div className="p-4 bg-gray-50 rounded-lg">
                <h3 className="text-sm font-medium mb-2">Data Directory</h3>
                <p className="text-sm text-gray-600 font-mono">{config.data_dir || '/tmp/rago'}</p>
              </div>
              
              <div className="p-4 bg-gray-50 rounded-lg">
                <h3 className="text-sm font-medium mb-2">Active Features</h3>
                <div className="flex gap-2 mt-2">
                  {config.mode?.llm_only === false && (
                    <>
                      {config.mode?.rag_enabled && (
                        <Badge variant="outline">RAG Enabled</Badge>
                      )}
                      {config.mode?.mcp_enabled && (
                        <Badge variant="outline">MCP Enabled</Badge>
                      )}
                      {config.mode?.agents_enabled && (
                        <Badge variant="outline">Agents Enabled</Badge>
                      )}
                    </>
                  )}
                  {config.mode?.llm_only && (
                    <Badge variant="outline">LLM Only Mode</Badge>
                  )}
                </div>
              </div>

              <div className="p-4 bg-gray-50 rounded-lg">
                <h3 className="text-sm font-medium mb-2">Log Level</h3>
                <Badge>{config.log_level || 'info'}</Badge>
              </div>
            </div>
          ) : (
            <Alert>
              <AlertDescription>Unable to load configuration</AlertDescription>
            </Alert>
          )}

          {configOp.error && (
            <Alert variant="destructive">
              <AlertDescription>Failed to load configuration: {configOp.error}</AlertDescription>
            </Alert>
          )}
        </CardContent>
      </Card>
    </div>
  )
}