import { useState, useEffect } from 'react'
import { 
  Card, 
  Button, 
  Input, 
  Alert, 
  Space, 
  Typography, 
  Badge, 
  List, 
  Row, 
  Col, 
  Statistic,
  Modal,
  Empty,
  message as antMessage,
  Tooltip
} from 'antd'
import { 
  MessageOutlined, 
  UserOutlined, 
  RobotOutlined, 
  ClockCircleOutlined, 
  DeleteOutlined, 
  DownloadOutlined, 
  SearchOutlined,
  PlusOutlined,
  HistoryOutlined,
  ExclamationCircleOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  FileTextOutlined,
  CopyOutlined
} from '@ant-design/icons'
import { conversationApi, Conversation } from '@/lib/api'

const { Title, Text, Paragraph } = Typography
const { confirm } = Modal

export function ConversationHistoryTab() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [selectedConversation, setSelectedConversation] = useState<Conversation | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [newConversationTitle, setNewConversationTitle] = useState('')
  const [showNewConversationInput, setShowNewConversationInput] = useState(false)

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text).then(() => {
      antMessage.success(`${label} copied to clipboard`)
    }).catch(() => {
      antMessage.error('Failed to copy to clipboard')
    })
  }

  const fetchConversations = async () => {
    setLoading(true)
    setError(null)

    try {
      // Fetch ALL conversations
      const response = await conversationApi.list(1, 100)
      setConversations(response.conversations.map((summary: any) => ({
        id: summary.id,
        title: summary.title,
        messages: [],
        created_at: summary.created_at,
        updated_at: summary.updated_at,
        message_count: summary.message_count
      })))
    } catch (err) {
      console.warn('Failed to fetch conversations:', err)
      setError('Failed to load conversations')
    } finally {
      setLoading(false)
    }
  }

  const fetchConversationDetails = async (conversationId: string) => {
    setLoading(true)
    setError(null)

    try {
      const conversation = await conversationApi.get(conversationId)
      setSelectedConversation(conversation)
    } catch (err) {
      console.warn('Failed to fetch conversation details:', err)
      setError('Failed to load conversation details')
    } finally {
      setLoading(false)
    }
  }

  const createNewConversation = async () => {
    if (!newConversationTitle.trim()) return

    setLoading(true)
    setError(null)

    try {
      const { id } = await conversationApi.createNew()
      await conversationApi.save({
        id,
        title: newConversationTitle,
        messages: []
      })
      setNewConversationTitle('')
      setShowNewConversationInput(false)
      await fetchConversations()
      antMessage.success('Conversation created successfully')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create conversation')
    } finally {
      setLoading(false)
    }
  }

  const deleteConversation = async (conversationId: string) => {
    confirm({
      title: 'Delete Conversation',
      content: 'Are you sure you want to delete this conversation?',
      icon: <ExclamationCircleOutlined />,
      okText: 'Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      async onOk() {
        setLoading(true)
        setError(null)

        try {
          await conversationApi.delete(conversationId)
          
          // Reset selected conversation if it was deleted
          if (selectedConversation?.id === conversationId) {
            setSelectedConversation(null)
          }
          
          await fetchConversations()
          antMessage.success('Conversation deleted successfully')
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Failed to delete conversation')
        } finally {
          setLoading(false)
        }
      }
    })
  }

  const exportConversation = async (conversationId: string) => {
    try {
      const conversation = await conversationApi.get(conversationId)
      
      // Create and download file
      const blob = new Blob([JSON.stringify(conversation, null, 2)], { type: 'application/json' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `conversation-${conversationId}.json`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      antMessage.success('Conversation exported successfully')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to export conversation')
    }
  }

  useEffect(() => {
    fetchConversations()
  }, [])

  const filteredConversations = Array.isArray(conversations) 
    ? conversations.filter(conv =>
        conv?.title?.toLowerCase()?.includes(searchTerm.toLowerCase())
      )
    : []

  const formatDate = (dateInput: string | number) => {
    // Handle both Unix timestamp (number) and ISO string
    const date = typeof dateInput === 'number' ? new Date(dateInput * 1000) : new Date(dateInput)
    return date.toLocaleString()
  }


  if (error) {
    return (
      <Alert
        message={error}
        type="error"
        showIcon
        icon={<ExclamationCircleOutlined />}
        style={{ margin: 16 }}
        closable
        onClose={() => setError(null)}
      />
    )
  }

  return (
    <div style={{ height: '100%', display: 'flex' }}>
      {/* Conversations List */}
      <div style={{ 
        width: '400px', 
        borderRight: '1px solid #f0f0f0', 
        padding: 16, 
        display: 'flex',
        flexDirection: 'column',
        gap: 16 
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <Title level={3} style={{ margin: 0, textAlign: 'left' }}>Conversations</Title>
            <Text type="secondary" style={{ fontSize: 12 }}>
              Showing ALL ({filteredConversations.length} of {conversations.length} total)
            </Text>
          </div>
          <Button
            onClick={() => setShowNewConversationInput(!showNewConversationInput)}
            icon={<PlusOutlined />}
            size="small"
          >
            New
          </Button>
        </div>

        {/* New Conversation Input */}
        {showNewConversationInput && (
          <Space.Compact style={{ width: '100%' }}>
            <Input
              placeholder="Conversation title..."
              value={newConversationTitle}
              onChange={(e) => setNewConversationTitle(e.target.value)}
              onPressEnter={createNewConversation}
              style={{ flex: 1 }}
            />
            <Button onClick={createNewConversation} disabled={loading}>
              Create
            </Button>
          </Space.Compact>
        )}

        {/* Search */}
        <Input
          placeholder="Search conversations..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          prefix={<SearchOutlined />}
        />

        {/* Conversations List */}
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <List
            loading={loading}
            dataSource={filteredConversations}
            style={{ height: '100%', overflow: 'auto' }}
            renderItem={(conversation) => (
              <List.Item
                style={{ 
                  padding: 0,
                  marginBottom: 8,
                  border: selectedConversation?.id === conversation.id ? '1px solid #1890ff' : 'none',
                  borderRadius: 6
                }}
              >
                <Card
                  size="small"
                  hoverable
                  onClick={() => fetchConversationDetails(conversation.id)}
                  style={{ 
                    width: '100%',
                    cursor: 'pointer',
                    backgroundColor: selectedConversation?.id === conversation.id ? '#e6f7ff' : undefined
                  }}
                  bodyStyle={{ padding: 12 }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Title level={5} style={{ margin: 0, fontSize: 14 }} ellipsis={{ tooltip: true }}>
                        {conversation.title}
                      </Title>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 4 }}>
                        <ClockCircleOutlined style={{ fontSize: 10, color: '#8c8c8c' }} />
                        <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                          {formatDate(conversation.updated_at)}
                        </Text>
                      </div>
                      {(conversation as any).message_count !== undefined && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 6 }}>
                          <Badge count={(conversation as any).message_count} size="small" color="#1890ff" />
                          <Text style={{ fontSize: 11 }}>messages</Text>
                        </div>
                      )}
                    </div>
                    <Space size="small">
                      <Tooltip title="Export">
                        <Button
                          type="text"
                          size="small"
                          icon={<DownloadOutlined />}
                          onClick={(e) => {
                            e.stopPropagation()
                            exportConversation(conversation.id)
                          }}
                        />
                      </Tooltip>
                      <Tooltip title="Delete">
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={(e) => {
                            e.stopPropagation()
                            deleteConversation(conversation.id)
                          }}
                        />
                      </Tooltip>
                    </Space>
                  </div>
                </Card>
              </List.Item>
            )}
          />
        </div>

        <Button 
          onClick={fetchConversations} 
          disabled={loading} 
          icon={<ReloadOutlined spin={loading} />}
          block
        >
          Refresh
        </Button>
      </div>

      {/* Conversation Details */}
      <div style={{ flex: 1, padding: 16 }}>
        {selectedConversation ? (
          <Space direction="vertical" style={{ width: '100%' }} size="large">
            {/* Conversation Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <Title level={2} style={{ margin: 0, textAlign: 'left' }}>{selectedConversation.title}</Title>
                <Space direction="vertical" size="small">
                  <Text type="secondary">
                    Created: {formatDate(selectedConversation.created_at)}
                  </Text>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      ID: {selectedConversation.id}
                    </Text>
                    <Tooltip title="Copy conversation ID">
                      <Button
                        type="text"
                        size="small"
                        icon={<CopyOutlined />}
                        onClick={() => copyToClipboard(selectedConversation.id, 'Conversation ID')}
                        style={{ padding: 2, height: 'auto' }}
                      />
                    </Tooltip>
                  </div>
                </Space>
              </div>
              <Space>
                <Button
                  icon={<DownloadOutlined />}
                  onClick={() => exportConversation(selectedConversation.id)}
                >
                  Export
                </Button>
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => deleteConversation(selectedConversation.id)}
                >
                  Delete
                </Button>
              </Space>
            </div>

            {/* Token Usage Stats */}
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} lg={6}>
                <Card size="small">
                  <Statistic
                    title="Messages"
                    value={selectedConversation.messages?.length || 0}
                    prefix={<MessageOutlined />}
                    valueStyle={{ color: '#1890ff' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card size="small">
                  <Statistic
                    title="Total Tokens"
                    value={(selectedConversation as any).total_tokens || 0}
                    prefix={<FileTextOutlined />}
                    valueStyle={{ color: '#52c41a' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card size="small">
                  <Statistic
                    title="API Calls"
                    value={1}
                    prefix={<ThunderboltOutlined />}
                    valueStyle={{ color: '#722ed1' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card size="small">
                  <Statistic
                    title="Total Cost"
                    value="N/A"
                    valueStyle={{ color: '#f5222d' }}
                  />
                </Card>
              </Col>
            </Row>

            {/* Messages */}
            <Card
              title={
                <div style={{ textAlign: 'left' }}>
                  <Space>
                    <HistoryOutlined />
                    Messages ({selectedConversation.messages?.length || 0})
                  </Space>
                </div>
              }
            >
              <div style={{ maxHeight: 400, overflow: 'auto' }}>
                <List
                  dataSource={selectedConversation.messages || []}
                  renderItem={(message) => (
                    <List.Item style={{ border: 'none', padding: '8px 0' }}>
                      <div style={{ 
                        width: '100%',
                        display: 'flex',
                        justifyContent: message.role === 'user' ? 'flex-end' : 'flex-start'
                      }}>
                        <div style={{ 
                          maxWidth: '70%',
                          display: 'flex',
                          gap: 8,
                          alignItems: 'flex-start',
                          flexDirection: message.role === 'user' ? 'row-reverse' : 'row'
                        }}>
                          <div style={{
                            width: 32,
                            height: 32,
                            borderRadius: '50%',
                            backgroundColor: message.role === 'user' ? '#1890ff' : '#52c41a',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            flexShrink: 0
                          }}>
                            {message.role === 'user' ? 
                              <UserOutlined style={{ color: 'white', fontSize: 14 }} /> : 
                              <RobotOutlined style={{ color: 'white', fontSize: 14 }} />
                            }
                          </div>
                          
                          <div style={{
                            padding: 12,
                            borderRadius: 8,
                            backgroundColor: message.role === 'user' ? '#1890ff' : '#f5f5f5',
                            color: message.role === 'user' ? 'white' : 'inherit'
                          }}>
                            <Paragraph 
                              style={{ 
                                margin: 0,
                                color: message.role === 'user' ? 'white' : 'inherit',
                                whiteSpace: 'pre-wrap',
                                wordBreak: 'break-word'
                              }}
                            >
                              {message.content}
                            </Paragraph>
                            <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8 }}>
                              <Text style={{ 
                                fontSize: 11, 
                                color: message.role === 'user' ? 'rgba(255,255,255,0.8)' : '#8c8c8c'
                              }}>
                                {formatDate(message.timestamp || Date.now())}
                              </Text>
                            </div>
                          </div>
                        </div>
                      </div>
                    </List.Item>
                  )}
                />
              </div>
            </Card>
          </Space>
        ) : (
          <div style={{ 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center', 
            height: '100%' 
          }}>
            <Empty
              image={<MessageOutlined style={{ fontSize: 48, color: '#bfbfbf' }} />}
              description={
                <div>
                  <Title level={4} type="secondary" style={{ textAlign: 'left' }}>No conversation selected</Title>
                  <Text type="secondary">Select a conversation from the list to view its details</Text>
                </div>
              }
            />
          </div>
        )}
      </div>
    </div>
  )
}