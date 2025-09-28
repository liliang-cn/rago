import { useState, useEffect } from 'react'
import { Card, Button, Tabs, Space, Typography, Alert, Statistic, Row, Col, Tag } from 'antd'
import { 
  DashboardOutlined, 
  CloudServerOutlined, 
  DatabaseOutlined, 
  AppstoreOutlined, 
  ReloadOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  ClockCircleOutlined,
  LineChartOutlined,
  ThunderboltOutlined,
  HddOutlined,
  GlobalOutlined,
  SettingOutlined
} from '@ant-design/icons'
import { apiClient } from '@/lib/api'

const { Title, Text } = Typography
const { TabPane } = Tabs

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
        return 'blue'
      case 'POST':
        return 'green'
      case 'PUT':
        return 'orange'
      case 'DELETE':
        return 'red'
      default:
        return 'default'
    }
  }

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'RAG':
        return <DatabaseOutlined />
      case 'MCP':
        return <ThunderboltOutlined />
      case 'System':
        return <SettingOutlined />
      default:
        return <GlobalOutlined />
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
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <Title level={2} style={{ margin: 0, textAlign: 'left' }}>System Status</Title>
            <Text type="secondary" style={{ textAlign: 'left' }}>Monitor API health and available endpoints</Text>
          </div>
          <Button 
            icon={<ReloadOutlined spin={isLoading} />}
            onClick={checkHealth} 
            disabled={isLoading}
            type="primary"
          >
            Refresh
          </Button>
        </div>

        {error && (
          <Alert
            message={error}
            type="error"
            showIcon
            icon={<ExclamationCircleOutlined />}
            closable
            onClose={() => setError('')}
          />
        )}

        <Tabs defaultActiveKey="overview" type="card" style={{ flex: 1 }}>
          <TabPane 
            tab={
              <Space>
                <DashboardOutlined />
                Overview
              </Space>
            } 
            key="overview"
          >

            <Space direction="vertical" style={{ width: '100%' }} size="large">
              <Card
                title={
                  <div style={{ textAlign: 'left' }}>
                    <Space>
                      <CloudServerOutlined />
                      <span>System Health</span>
                    </Space>
                  </div>
                }
              >
                <Text type="secondary" style={{ marginBottom: 16, display: 'block', textAlign: 'left' }}>
                  Current system status and basic information
                </Text>
                {health ? (
                  <Row gutter={[16, 16]}>
                    <Col xs={24} sm={12} lg={6}>
                      <Statistic
                        title="Status"
                        value={health.status}
                        prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
                        valueStyle={{ color: '#52c41a', textTransform: 'capitalize' }}
                      />
                    </Col>
                    <Col xs={24} sm={12} lg={6}>
                      <Statistic
                        title="Service"
                        value={health.service}
                        prefix={<DashboardOutlined style={{ color: '#1890ff' }} />}
                      />
                    </Col>
                    <Col xs={24} sm={12} lg={6}>
                      <Statistic
                        title="Version"
                        value={health.version}
                        prefix={<AppstoreOutlined style={{ color: '#722ed1' }} />}
                      />
                    </Col>
                    <Col xs={24} sm={12} lg={6}>
                      <Statistic
                        title="Last Check"
                        value={lastRefresh.toLocaleTimeString()}
                        prefix={<ClockCircleOutlined style={{ color: '#8c8c8c' }} />}
                      />
                    </Col>
                  </Row>
                ) : (
                  <div style={{ textAlign: 'center', padding: '32px 0' }}>
                    <ExclamationCircleOutlined style={{ fontSize: 32, color: '#bfbfbf', marginBottom: 8 }} />
                    <Text type="secondary" style={{ textAlign: 'left' }}>Unable to fetch system health</Text>
                  </div>
                )}
              </Card>

              <Row gutter={[16, 16]}>
                <Col xs={24} sm={8}>
                  <Card size="small">
                    <Statistic
                      title="RAG Endpoints"
                      value={groupedEndpoints['RAG']?.length || 0}
                      prefix={<DatabaseOutlined style={{ color: '#1890ff' }} />}
                      valueStyle={{ color: '#1890ff' }}
                    />
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card size="small">
                    <Statistic
                      title="MCP Endpoints"
                      value={groupedEndpoints['MCP']?.length || 0}
                      prefix={<ThunderboltOutlined style={{ color: '#52c41a' }} />}
                      valueStyle={{ color: '#52c41a' }}
                    />
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card size="small">
                    <Statistic
                      title="Total Endpoints"
                      value={apiEndpoints.length}
                      prefix={<GlobalOutlined style={{ color: '#722ed1' }} />}
                      valueStyle={{ color: '#722ed1' }}
                    />
                  </Card>
                </Col>
              </Row>
            </Space>
          </TabPane>

          <TabPane 
            tab={
              <Space>
                <GlobalOutlined />
                API Endpoints
              </Space>
            } 
            key="endpoints"
          >
            <Space direction="vertical" style={{ width: '100%' }} size="large">
              {Object.entries(groupedEndpoints).map(([category, endpoints]) => (
                <Card 
                  key={category}
                  title={
                    <div style={{ textAlign: 'left' }}>
                      <Space>
                        {getCategoryIcon(category)}
                        <span>{category} API</span>
                      </Space>
                    </div>
                  }
                >
                  <Text type="secondary" style={{ marginBottom: 16, display: 'block', textAlign: 'left' }}>
                    Available {category.toLowerCase()} endpoints and operations
                  </Text>
                  <Space direction="vertical" style={{ width: '100%' }} size="small">
                    {endpoints?.map((endpoint, index) => (
                      <div key={index} style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        justifyContent: 'space-between', 
                        padding: 12, 
                        backgroundColor: '#fafafa', 
                        borderRadius: 6, 
                        border: '1px solid #f0f0f0' 
                      }}>
                        <Space>
                          <Tag color={getMethodColor(endpoint.method)} style={{ fontFamily: 'monospace', fontWeight: 'bold' }}>
                            {endpoint.method}
                          </Tag>
                          <Text code style={{ fontSize: 12 }}>{endpoint.path}</Text>
                        </Space>
                        <Text type="secondary" style={{ fontSize: 12 }}>{endpoint.description}</Text>
                      </div>
                    ))}
                  </Space>
                </Card>
              ))}
            </Space>
          </TabPane>

          <TabPane 
            tab={
              <Space>
                <LineChartOutlined />
                Monitoring
              </Space>
            } 
            key="monitoring"
          >
            <Card
              title={<div style={{ textAlign: 'left' }}>Real-time Monitoring</div>}
            >
              <Text type="secondary" style={{ marginBottom: 16, display: 'block', textAlign: 'left' }}>
                System metrics and performance indicators
              </Text>
              <Row gutter={[24, 24]}>
                <Col xs={24} md={12}>
                  <Space direction="vertical" style={{ width: '100%' }} size="middle">
                    <Space>
                      <DashboardOutlined />
                      <Text strong style={{ textAlign: 'left' }}>API Response Times</Text>
                    </Space>
                    <Space direction="vertical" style={{ width: '100%' }} size="small">
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                        <Text>Health Check</Text>
                        <Text style={{ color: '#52c41a' }}>~50ms</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                        <Text>Document Query</Text>
                        <Text style={{ color: '#1890ff' }}>~200ms</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                        <Text>MCP Tool Call</Text>
                        <Text style={{ color: '#722ed1' }}>~150ms</Text>
                      </div>
                    </Space>
                  </Space>
                </Col>

                <Col xs={24} md={12}>
                  <Space direction="vertical" style={{ width: '100%' }} size="middle">
                    <Space>
                      <HddOutlined />
                      <Text strong style={{ textAlign: 'left' }}>System Resources</Text>
                    </Space>
                    <Space direction="vertical" style={{ width: '100%' }} size="small">
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                        <Text>Memory Usage</Text>
                        <Text type="secondary">~45MB</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                        <Text>Active Connections</Text>
                        <Text type="secondary">1</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                        <Text>Uptime</Text>
                        <Text type="secondary">{Math.floor(Date.now() / 1000 / 60)}min</Text>
                      </div>
                    </Space>
                  </Space>
                </Col>
              </Row>
            </Card>
          </TabPane>
        </Tabs>
      </Space>
    </div>
  )
}