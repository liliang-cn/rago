import { useState, useEffect } from 'react'
import { Card, Button, Tabs, Input, Space, Typography, Spin, Empty, Tag, Alert } from 'antd'
import { 
  ToolOutlined, 
  CloudServerOutlined, 
  PlayCircleOutlined, 
  StopOutlined, 
  ReloadOutlined, 
  ExclamationCircleOutlined, 
  CheckCircleOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  ThunderboltOutlined
} from '@ant-design/icons'
import { apiClient, MCPTool, MCPToolResult } from '@/lib/api'

const { TextArea } = Input
const { Title, Text, Paragraph } = Typography
const { TabPane } = Tabs

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
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Card
        title={
          <div style={{ textAlign: 'left' }}>
            <Space>
              <ToolOutlined />
              <span>MCP Integration</span>
            </Space>
          </div>
        }
        extra={
          <Button
            icon={<ReloadOutlined spin={isLoadingTools} />}
            onClick={loadMCPData}
            loading={isLoadingTools}
          >
            Refresh
          </Button>
        }
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column' }}
      >
        <Text type="secondary" style={{ marginBottom: 16, textAlign: 'left', display: 'block' }}>
          Manage Model Context Protocol servers and tools
        </Text>

        {error && (
          <Alert
            message={error}
            type="error"
            icon={<ExclamationCircleOutlined />}
            style={{ marginBottom: 16 }}
            closable
            onClose={() => setError('')}
          />
        )}

        <Tabs defaultActiveKey="tools" style={{ flex: 1 }}>

          <TabPane 
            tab={
              <Space>
                <ToolOutlined />
                Tools ({tools?.length || 0})
              </Space>
            } 
            key="tools"
          >
            {isLoadingTools ? (
              <div style={{ textAlign: 'center', padding: '64px 0' }}>
                <Spin size="large" />
                <div style={{ marginTop: 16 }}>
                  <Text type="secondary">Loading tools...</Text>
                </div>
              </div>
            ) : (
              <div style={{ 
                display: 'grid', 
                gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', 
                gap: '16px' 
              }}>
                {tools?.map((tool) => (
                  <Card 
                    key={tool.name} 
                    size="small"
                    hoverable
                    onClick={() => setSelectedTool(tool)}
                    style={{ cursor: 'pointer' }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 8 }}>
                      <Title level={5} style={{ margin: 0, fontSize: 14, textAlign: 'left' }}>
                        {tool.name.replace('mcp_', '').replace(/_/g, ' ')}
                      </Title>
                      <Tag icon={<DatabaseOutlined />} color="blue" style={{ fontSize: 10 }}>
                        {tool.server_name}
                      </Tag>
                    </div>
                    <Paragraph
                      style={{ margin: 0, fontSize: 12, color: '#666' }}
                      ellipsis={{ rows: 2 }}
                    >
                      {tool.description}
                    </Paragraph>
                  </Card>
                )) || []}
              </div>
            )}
          </TabPane>

          <TabPane 
            tab={
              <Space>
                <CloudServerOutlined />
                Servers ({servers?.length || 0})
              </Space>
            } 
            key="servers"
          >
            <div style={{ 
              display: 'grid', 
              gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))', 
              gap: '16px' 
            }}>
              {servers?.map((server) => (
                <Card key={server.name} size="small">
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                    <Space>
                      <CloudServerOutlined />
                      <Title level={5} style={{ margin: 0, textAlign: 'left' }}>{server.name}</Title>
                    </Space>
                    <Space>
                      {server.status ? (
                        <CheckCircleOutlined style={{ color: '#52c41a' }} />
                      ) : (
                        <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
                      )}
                      <Text strong style={{ color: server.status ? '#52c41a' : '#ff4d4f' }}>
                        {server.status ? 'Connected' : 'Disconnected'}
                      </Text>
                    </Space>
                  </div>
                  <Space>
                    <Button
                      size="small"
                      type={server.status ? "default" : "primary"}
                      icon={<PlayCircleOutlined />}
                      onClick={() => startServer(server.name)}
                      disabled={server.status}
                    >
                      Start
                    </Button>
                    <Button
                      size="small"
                      danger={server.status}
                      icon={<StopOutlined />}
                      onClick={() => stopServer(server.name)}
                      disabled={!server.status}
                    >
                      Stop
                    </Button>
                  </Space>
                </Card>
              )) || []}
            </div>
          </TabPane>

          <TabPane 
            tab={
              <Space>
                <PlayCircleOutlined />
                Execute
              </Space>
            } 
            key="execute"
          >
            <div style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: '24px'
            }}>
              <Card title={<div style={{ textAlign: 'left' }}>Execute Tool</div>} size="small">
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  <div>
                    <Text strong style={{ textAlign: 'left', display: 'block' }}>Selected Tool:</Text>
                    <div style={{ 
                      marginTop: 8, 
                      padding: 8, 
                      backgroundColor: '#f5f5f5', 
                      borderRadius: 4, 
                      border: '1px solid #d9d9d9',
                      fontSize: 12
                    }}>
                      {selectedTool ? selectedTool.name : 'No tool selected'}
                    </div>
                  </div>

                  <div>
                    <Text strong style={{ textAlign: 'left', display: 'block' }}>Arguments (JSON):</Text>
                    <TextArea
                      placeholder='{"query": "SELECT * FROM table", "limit": 10}'
                      value={toolArgs}
                      onChange={(e) => setToolArgs(e.target.value)}
                      style={{ marginTop: 8 }}
                      rows={4}
                    />
                  </div>

                  <Button 
                    type="primary"
                    icon={isLoading ? <ReloadOutlined spin /> : <PlayCircleOutlined />}
                    onClick={executeTool} 
                    disabled={!selectedTool || isLoading}
                    block
                    loading={isLoading}
                  >
                    {isLoading ? 'Executing...' : 'Execute Tool'}
                  </Button>
                </Space>
              </Card>

              <Card title={<div style={{ textAlign: 'left' }}>Result</div>} size="small">
                {toolResult ? (
                  <Space direction="vertical" style={{ width: '100%' }} size="middle">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Space>
                        {toolResult.success ? (
                          <CheckCircleOutlined style={{ color: '#52c41a' }} />
                        ) : (
                          <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
                        )}
                        <Text strong style={{ color: toolResult.success ? '#52c41a' : '#ff4d4f' }}>
                          {toolResult.success ? 'Success' : 'Failed'}
                        </Text>
                      </Space>
                      <Space>
                        <ClockCircleOutlined style={{ fontSize: 12, color: '#666' }} />
                        <Text style={{ fontSize: 12, color: '#666' }}>
                          {toolResult.duration}ms
                        </Text>
                      </Space>
                    </div>
                    
                    <div style={{ 
                      backgroundColor: '#f5f5f5', 
                      padding: 12, 
                      borderRadius: 4, 
                      border: '1px solid #d9d9d9',
                      maxHeight: 300,
                      overflow: 'auto'
                    }}>
                      <pre style={{ 
                        fontSize: 11, 
                        margin: 0, 
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word'
                      }}>
                        {toolResult.success 
                          ? JSON.stringify(toolResult.data, null, 2)
                          : toolResult.error
                        }
                      </pre>
                    </div>
                  </Space>
                ) : (
                  <Empty
                    image={<ThunderboltOutlined style={{ fontSize: 48, color: '#bfbfbf' }} />}
                    description="No execution result yet"
                  />
                )}
              </Card>
            </div>
          </TabPane>
        </Tabs>
      </Card>
    </div>
  )
}