import { useState, useEffect, useRef } from 'react'
import { Card, Button, Tabs, Input, Space, Typography, Spin, Empty, Tag, Alert, Avatar, message } from 'antd'
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
  ThunderboltOutlined,
  MessageOutlined,
  SendOutlined,
  UserOutlined,
  RobotOutlined,
  ClearOutlined
} from '@ant-design/icons'
import { apiClient, MCPTool, MCPToolResult, conversationApi } from '@/lib/api'

const { TextArea } = Input
const { Title, Text, Paragraph } = Typography
const { TabPane } = Tabs

interface ServerStatus {
  name: string
  status: boolean
}

interface MCPMessage {
  role: 'user' | 'assistant'
  content: string
  thinking?: string
  tool_calls?: Array<{
    tool_name: string
    args: Record<string, any>
    result: any
    success: boolean
    error?: string
    duration?: string
  }>
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
  
  // Chat state
  const [messages, setMessages] = useState<MCPMessage[]>([])
  const [chatInput, setChatInput] = useState<string>('')
  const [isChatLoading, setIsChatLoading] = useState(false)
  const [conversationId, setConversationId] = useState<string>('')
  const [showThinking, setShowThinking] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadMCPData()
    initializeConversation()
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const initializeConversation = async () => {
    try {
      const { id } = await conversationApi.createNew()
      setConversationId(id)
    } catch (error) {
      console.error('Failed to create conversation:', error)
    }
  }

  const loadMCPData = async () => {
    setIsLoadingTools(true)
    setError('')

    try {
      // Load tools and servers in parallel
      const [toolsResponse, serversResponse] = await Promise.all([
        apiClient.getMCPTools(),
        apiClient.getMCPServers()
      ])

      console.log('Tools response:', toolsResponse)
      console.log('Servers response:', serversResponse)

      if (toolsResponse.data) {
        // Handle both formats: array directly or { tools: array }
        const toolsData = Array.isArray(toolsResponse.data) 
          ? toolsResponse.data 
          : toolsResponse.data.tools || []
        setTools(toolsData)
      }

      if (serversResponse.data) {
        const serversData = serversResponse.data.servers || {}
        const serverList = Object.entries(serversData).map(([name, status]) => ({
          name,
          status: Boolean(status)
        }))
        setServers(serverList)
      }
    } catch (err) {
      console.error('Error loading MCP data:', err)
      const errorMessage = err instanceof Error ? err.message : 'Failed to load MCP data'
      setError(errorMessage)
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

  const sendChatMessage = async () => {
    if (!chatInput.trim() || isChatLoading) return

    const userMessage = chatInput.trim()
    setChatInput('')
    setIsChatLoading(true)

    // Add user message to chat
    const newUserMessage: MCPMessage = {
      role: 'user',
      content: userMessage
    }
    setMessages(prev => [...prev, newUserMessage])

    try {
      const response = await apiClient.mcpChat({
        message: userMessage,
        conversation_id: conversationId,
        options: {
          show_thinking: showThinking,
          temperature: 0.7,
          max_tokens: 2000
        }
      })

      if (response.data) {
        const assistantMessage: MCPMessage = {
          role: 'assistant',
          content: response.data.final_response || response.data.content,
          thinking: response.data.thinking,
          tool_calls: response.data.tool_calls
        }
        setMessages(prev => [...prev, assistantMessage])

        // Update conversation ID if it was generated
        if (response.data.conversation_id && response.data.conversation_id !== conversationId) {
          setConversationId(response.data.conversation_id)
        }
      } else {
        message.error('Failed to get response from MCP chat')
      }
    } catch (error) {
      console.error('Error sending MCP chat message:', error)
      message.error('Failed to send message. Please try again.')
    } finally {
      setIsChatLoading(false)
    }
  }

  const clearChat = async () => {
    setMessages([])
    try {
      const { id } = await conversationApi.createNew()
      setConversationId(id)
      message.success('Chat cleared and new conversation started')
    } catch (error) {
      message.error('Failed to create new conversation')
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

        <Tabs defaultActiveKey="chat" style={{ flex: 1 }}>

          <TabPane 
            tab={
              <Space>
                <MessageOutlined />
                Chat
              </Space>
            } 
            key="chat"
          >
            {/* Chat Interface */}
            <div style={{ 
              display: 'flex', 
              flexDirection: 'column', 
              height: '600px' 
            }}>
              {/* Chat Header */}
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center',
                marginBottom: 16,
                padding: '8px 12px',
                backgroundColor: '#f5f5f5',
                borderRadius: 4
              }}>
                <Space>
                  <Text strong>MCP Chat</Text>
                  {conversationId && (
                    <Tag color="blue">ID: {conversationId.slice(0, 8)}</Tag>
                  )}
                </Space>
                <Space>
                  <Button 
                    size="small" 
                    onClick={() => setShowThinking(!showThinking)}
                    type={showThinking ? "primary" : "default"}
                  >
                    {showThinking ? "Hide" : "Show"} Thinking
                  </Button>
                  <Button
                    size="small"
                    icon={<ClearOutlined />}
                    onClick={clearChat}
                    disabled={messages.length === 0}
                  >
                    Clear
                  </Button>
                </Space>
              </div>

              {/* Messages Area */}
              <div style={{
                flex: 1,
                overflowY: 'auto',
                padding: 16,
                backgroundColor: '#f9f9f9',
                borderRadius: 8,
                marginBottom: 16
              }}>
                {messages.length === 0 ? (
                  <Empty
                    description="Start a conversation with MCP tools"
                    style={{ marginTop: 50 }}
                  />
                ) : (
                  <Space direction="vertical" style={{ width: '100%' }} size="middle">
                    {messages.map((msg, index) => (
                      <div
                        key={index}
                        style={{
                          display: 'flex',
                          justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start'
                        }}
                      >
                        <Space
                          align="start"
                          style={{
                            maxWidth: '80%',
                            flexDirection: msg.role === 'user' ? 'row-reverse' : 'row'
                          }}
                        >
                          <Avatar
                            icon={msg.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
                            style={{
                              backgroundColor: msg.role === 'user' ? '#1890ff' : '#52c41a'
                            }}
                          />
                          <Card
                            size="small"
                            style={{
                              backgroundColor: msg.role === 'user' ? '#e6f7ff' : '#ffffff',
                              border: msg.role === 'user' ? '1px solid #91d5ff' : '1px solid #e8e8e8'
                            }}
                          >
                            <Paragraph
                              style={{
                                margin: 0,
                                whiteSpace: 'pre-wrap',
                                textAlign: 'left'
                              }}
                            >
                              {msg.content}
                            </Paragraph>

                            {/* Tool Calls */}
                            {msg.tool_calls && msg.tool_calls.length > 0 && (
                              <div style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid #f0f0f0' }}>
                                <Text type="secondary" style={{ fontSize: 12, fontWeight: 500 }}>
                                  Tool Calls:
                                </Text>
                                <Space direction="vertical" size="small" style={{ width: '100%', marginTop: 8 }}>
                                  {msg.tool_calls.map((toolCall, idx) => (
                                    <Card key={idx} size="small" style={{ backgroundColor: '#f9f9f9' }}>
                                      <Space>
                                        <ToolOutlined />
                                        <Text style={{ fontSize: 12 }}>{toolCall.tool_name}</Text>
                                        <Tag 
                                          color={toolCall.success ? "green" : "red"} 
                                          style={{ fontSize: 10 }}
                                        >
                                          {toolCall.success ? "Success" : "Failed"}
                                        </Tag>
                                        {toolCall.duration && (
                                          <Text style={{ fontSize: 10, color: '#666' }}>
                                            {toolCall.duration}ms
                                          </Text>
                                        )}
                                      </Space>
                                    </Card>
                                  ))}
                                </Space>
                              </div>
                            )}

                            {/* Thinking */}
                            {msg.thinking && showThinking && (
                              <div style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid #f0f0f0' }}>
                                <Text type="secondary" style={{ fontSize: 11 }}>
                                  Thinking: {msg.thinking}
                                </Text>
                              </div>
                            )}
                          </Card>
                        </Space>
                      </div>
                    ))}
                    {isChatLoading && (
                      <div style={{ display: 'flex', justifyContent: 'flex-start' }}>
                        <Space align="start">
                          <Avatar icon={<RobotOutlined />} style={{ backgroundColor: '#52c41a' }} />
                          <Card size="small">
                            <Spin size="small" />
                            <Text style={{ marginLeft: 8 }}>Processing with MCP tools...</Text>
                          </Card>
                        </Space>
                      </div>
                    )}
                    <div ref={messagesEndRef} />
                  </Space>
                )}
              </div>

              {/* Input Area */}
              <Space.Compact style={{ width: '100%' }}>
                <Input
                  placeholder="Ask me to use MCP tools..."
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  onPressEnter={(e) => {
                    if (!e.shiftKey) {
                      e.preventDefault()
                      sendChatMessage()
                    }
                  }}
                  disabled={isChatLoading}
                />
                <Button
                  type="primary"
                  icon={<SendOutlined />}
                  onClick={sendChatMessage}
                  loading={isChatLoading}
                >
                  Send
                </Button>
              </Space.Compact>
            </div>
          </TabPane>

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