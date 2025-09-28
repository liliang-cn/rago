import { useState, useEffect } from 'react'
import { 
  Card, 
  Button, 
  Input, 
  Alert, 
  Space, 
  Typography, 
  List, 
  Row, 
  Col, 
  Statistic,
  Tabs,
  Select,
  Empty,
  Tag,
  message as antMessage,
  Tooltip
} from 'antd'
import { 
  SearchOutlined, 
  ClockCircleOutlined, 
  CheckCircleOutlined, 
  CloseCircleOutlined, 
  WarningOutlined,
  ThunderboltOutlined, 
  ReloadOutlined,
  DownloadOutlined,
  EyeOutlined,
  CloudServerOutlined,
  SettingOutlined,
  ExclamationCircleOutlined,
  ArrowRightOutlined,
  CopyOutlined
} from '@ant-design/icons'
import { 
  apiClient, 
  ToolCallRecord, 
  ToolCallFilter, 
  ToolCallStats, 
  ToolCallAnalytics, 
  ToolCallVisualization 
} from '@/lib/api'

const { Title, Text } = Typography
const { TabPane } = Tabs
const { Option } = Select

export function ToolCallsTab() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [toolCalls, setToolCalls] = useState<ToolCallRecord[]>([])
  const [selectedCall, setSelectedCall] = useState<ToolCallVisualization | null>(null)
  const [stats, setStats] = useState<ToolCallStats | null>(null)
  const [analytics, setAnalytics] = useState<ToolCallAnalytics | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [uuidSearch, setUuidSearch] = useState('')
  const [filter, setFilter] = useState<ToolCallFilter>({
    limit: 50,
    offset: 0
  })
  const [timeRange, setTimeRange] = useState<string>('24h')

  const timeRanges = {
    '1h': { label: '1 Hour', hours: 1 },
    '6h': { label: '6 Hours', hours: 6 },
    '24h': { label: '24 Hours', hours: 24 },
    '7d': { label: '7 Days', hours: 24 * 7 },
    '30d': { label: '30 Days', hours: 24 * 30 },
  }

  const fetchData = async () => {
    setLoading(true)
    setError(null)

    try {
      const now = new Date()
      const startTime = new Date(now.getTime() - timeRanges[timeRange as keyof typeof timeRanges].hours * 60 * 60 * 1000)
      const currentFilter = {
        ...filter,
        start_time: Math.floor(startTime.getTime() / 1000).toString(),
        end_time: Math.floor(now.getTime() / 1000).toString(),
      }

      const [
        callsResponse,
        statsResponse,
        analyticsResponse,
      ] = await Promise.all([
        apiClient.getToolCalls(currentFilter),
        apiClient.getToolCallStats(currentFilter),
        apiClient.getToolCallAnalytics(currentFilter),
      ])

      if (callsResponse.error) throw new Error(callsResponse.error)
      if (statsResponse.error) throw new Error(statsResponse.error)
      if (analyticsResponse.error) throw new Error(analyticsResponse.error)

      setToolCalls(Array.isArray(callsResponse.data) ? callsResponse.data : [])
      setStats(statsResponse.data || null)
      setAnalytics(analyticsResponse.data || null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch tool calls data')
    } finally {
      setLoading(false)
    }
  }

  const searchByUuid = async () => {
    if (!uuidSearch.trim()) {
      await fetchData()
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await apiClient.getToolCall(uuidSearch.trim())
      if (response.error) throw new Error(response.error)
      
      if (response.data) {
        setToolCalls(response.data ? [response.data] : [])
        const vizResponse = await apiClient.getToolCallVisualization(uuidSearch.trim())
        if (vizResponse.data) {
          setSelectedCall(vizResponse.data)
        }
      } else {
        setToolCalls([])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to find tool call')
    } finally {
      setLoading(false)
    }
  }

  const fetchCallDetails = async (uuid: string) => {
    setLoading(true)
    setError(null)

    try {
      const response = await apiClient.getToolCallVisualization(uuid)
      if (response.error) throw new Error(response.error)
      setSelectedCall(response.data || null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch call details')
    } finally {
      setLoading(false)
    }
  }

  const exportToolCalls = async () => {
    try {
      const response = await apiClient.exportToolCalls(filter)
      if (response.error) throw new Error(response.error)
      
      // Create and download file
      const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' })
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `tool-calls-${new Date().toISOString().split('T')[0]}.json`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
      antMessage.success('Tool calls exported successfully')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to export tool calls')
    }
  }

  // Future enhancement: Add filtering and analytics features
  // const handleFilterChange = (newFilter: ToolCallFilter) => {
  //   setFilter(newFilter)
  //   fetchData()
  // }

  // Note: analytics state is prepared for future analytics visualizations
  console.log('Analytics available:', analytics !== null, 'Filter ready:', typeof setFilter === 'function')

  const copyUuid = (uuid: string) => {
    navigator.clipboard.writeText(uuid)
    antMessage.success('UUID copied to clipboard')
  }

  useEffect(() => {
    fetchData()
  }, [timeRange])

  const filteredCalls = Array.isArray(toolCalls) ? toolCalls.filter(call =>
    call?.tool_name?.toLowerCase()?.includes(searchTerm.toLowerCase()) ||
    call?.uuid?.toLowerCase()?.includes(searchTerm.toLowerCase()) ||
    call?.server_name?.toLowerCase()?.includes(searchTerm.toLowerCase())
  ) : []

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`
  }

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString()
  }

  const getStatusColor = (success: boolean) => {
    return success ? '#52c41a' : '#ff4d4f'
  }

  const getTypeColor = (type: string) => {
    const colors = {
      'mcp': 'blue',
      'rag': 'green', 
      'llm': 'purple',
      'system': 'default'
    }
    return colors[type as keyof typeof colors] || 'default'
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
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <Title level={2} style={{ margin: 0, textAlign: 'left' }}>Tool Calls</Title>
            <Text type="secondary" style={{ textAlign: 'left' }}>
              Monitor and analyze tool call execution across all systems
            </Text>
          </div>
          <Space>
            <Select value={timeRange} onChange={setTimeRange} style={{ width: 120 }}>
              {Object.entries(timeRanges).map(([key, range]) => (
                <Option key={key} value={key}>
                  {range.label}
                </Option>
              ))}
            </Select>
            <Button 
              onClick={exportToolCalls} 
              icon={<DownloadOutlined />}
            >
              Export
            </Button>
            <Button 
              onClick={fetchData} 
              disabled={loading} 
              loading={loading}
              icon={<ReloadOutlined />}
            >
              Refresh
            </Button>
          </Space>
        </div>

        {/* Search and Filters */}
        <Row gutter={16}>
          <Col xs={24} md={16}>
            <Input
              placeholder="Search by tool name, UUID, or server..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              prefix={<SearchOutlined />}
            />
          </Col>
          <Col xs={24} md={8}>
            <Space.Compact style={{ width: '100%' }}>
              <Input
                placeholder="Search by UUID..."
                value={uuidSearch}
                onChange={(e) => setUuidSearch(e.target.value)}
                onPressEnter={searchByUuid}
                style={{ flex: 1 }}
              />
              <Button onClick={searchByUuid} disabled={loading} icon={<SearchOutlined />} />
            </Space.Compact>
          </Col>
        </Row>

        {/* Overview Stats */}
        {stats && (
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} md={24} lg={24} xl={24} xxl={24}>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} md={12} lg={8} xl={8} xxl={5}>
                  <Card size="small">
                    <Statistic
                      title="Total Calls"
                      value={stats.total_calls || 0}
                      prefix={<ThunderboltOutlined />}
                      valueStyle={{ color: '#1890ff' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'left' }}>
                      {stats.unique_tools || 0} unique tools
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={12} lg={8} xl={8} xxl={5}>
                  <Card size="small">
                    <Statistic
                      title="Success Rate"
                      value={`${((stats.success_rate || 0) * 100).toFixed(1)}%`}
                      prefix={<CheckCircleOutlined />}
                      valueStyle={{ color: '#52c41a' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'left' }}>
                      {stats.successful_calls || 0} / {stats.total_calls || 0}
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={12} lg={8} xl={8} xxl={5}>
                  <Card size="small">
                    <Statistic
                      title="Avg Duration"
                      value={formatDuration(stats.avg_duration_ms || 0)}
                      prefix={<ClockCircleOutlined />}
                      valueStyle={{ color: '#722ed1' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'left' }}>
                      Total: {formatDuration(stats.total_duration_ms || 0)}
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={12} lg={8} xl={8} xxl={5}>
                  <Card size="small">
                    <Statistic
                      title="Servers"
                      value={stats.unique_servers || 0}
                      prefix={<CloudServerOutlined />}
                      valueStyle={{ color: '#13c2c2' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'left' }}>
                      Active MCP servers
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={12} lg={8} xl={8} xxl={4}>
                  <Card size="small">
                    <Statistic
                      title="Failed Calls"
                      value={stats.failed_calls || 0}
                      prefix={<CloseCircleOutlined />}
                      valueStyle={{ color: '#ff4d4f' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c', textAlign: 'left' }}>
                      {(((stats.failed_calls || 0) / (stats.total_calls || 1)) * 100).toFixed(1)}% failure rate
                    </Text>
                  </Card>
                </Col>
              </Row>
            </Col>
          </Row>
        )}

        {/* Main Content */}
        <div style={{ display: 'flex', height: 600, gap: 24 }}>
          {/* Tool Calls List */}
          <div style={{ width: '50%' }}>
            <Card
              title={
                <div style={{ textAlign: 'left' }}>
                  <Space>
                    <SettingOutlined />
                    Tool Calls ({filteredCalls.length})
                  </Space>
                </div>
              }
              style={{ height: '100%' }}
              bodyStyle={{ height: 'calc(100% - 70px)', overflow: 'hidden' }}
            >
              <Text type="secondary" style={{ display: 'block', marginBottom: 16, textAlign: 'left' }}>
                Recent tool executions with status and performance metrics
              </Text>
              <div style={{ height: 'calc(100% - 40px)', overflow: 'auto' }}>
                <List
                  loading={loading}
                  dataSource={filteredCalls}
                  renderItem={(call) => (
                    <List.Item style={{ padding: 0, marginBottom: 8 }}>
                      <Card
                        size="small"
                        hoverable
                        onClick={() => fetchCallDetails(call.uuid)}
                        style={{ 
                          width: '100%',
                          cursor: 'pointer',
                          backgroundColor: selectedCall?.record.uuid === call.uuid ? '#e6f7ff' : undefined,
                          border: selectedCall?.record.uuid === call.uuid ? '1px solid #1890ff' : undefined
                        }}
                        bodyStyle={{ padding: 16 }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                          <Space>
                            <Tag color={getTypeColor(call.tool_type)}>
                              {call.tool_type}
                            </Tag>
                            <Text strong>{call.tool_name}</Text>
                            {call.success ? (
                              <CheckCircleOutlined style={{ color: '#52c41a' }} />
                            ) : (
                              <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                            )}
                          </Space>
                          <Space>
                            <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                              {formatDuration(call.duration_ms || 0)}
                            </Text>
                            <Tooltip title="Copy UUID">
                              <Button
                                type="text"
                                size="small"
                                icon={<CopyOutlined />}
                                onClick={(e) => {
                                  e.stopPropagation()
                                  copyUuid(call.uuid)
                                }}
                              />
                            </Tooltip>
                          </Space>
                        </div>

                        <Space direction="vertical" style={{ width: '100%' }} size="small">
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <Text style={{ fontSize: 11, color: '#8c8c8c' }}>UUID:</Text>
                            <Text code style={{ fontSize: 11 }}>
                              {call.uuid?.slice(0, 8) || 'Unknown'}...{call.uuid?.slice(-8) || ''}
                            </Text>
                          </div>
                          
                          {call.server_name && (
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <CloudServerOutlined style={{ fontSize: 10, color: '#8c8c8c' }} />
                              <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{call.server_name}</Text>
                            </div>
                          )}

                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <ClockCircleOutlined style={{ fontSize: 10, color: '#8c8c8c' }} />
                            <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                              {formatTimestamp(call.created_at)}
                            </Text>
                          </div>

                          {!call.success && call.error_message && (
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <WarningOutlined style={{ fontSize: 10, color: '#ff4d4f' }} />
                              <Text style={{ fontSize: 11, color: '#ff4d4f' }} ellipsis={{ tooltip: true }}>
                                {call.error_message}
                              </Text>
                            </div>
                          )}
                        </Space>
                      </Card>
                    </List.Item>
                  )}
                />
              </div>
            </Card>
          </div>

          {/* Tool Call Details */}
          <div style={{ width: '50%' }}>
            {selectedCall ? (
              <Tabs defaultActiveKey="details" style={{ height: '100%' }}>
                <TabPane
                  tab={
                    <Space>
                      <EyeOutlined />
                      Details
                    </Space>
                  }
                  key="details"
                >
                  <Card
                    title={<div style={{ textAlign: 'left' }}>Call Details</div>}
                    style={{ height: 520 }}
                    bodyStyle={{ height: 'calc(100% - 70px)', overflow: 'auto' }}
                  >
                    <Text type="secondary" style={{ display: 'block', marginBottom: 16, textAlign: 'left' }}>
                      Detailed information about the selected tool call
                    </Text>
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      {/* Basic Info */}
                      <Row gutter={[16, 16]}>
                        <Col span={12}>
                          <div>
                            <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Tool Name</Text>
                            <div style={{ marginTop: 4 }}>
                              <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                                {selectedCall.record?.tool_name}
                              </Text>
                            </div>
                          </div>
                        </Col>
                        <Col span={12}>
                          <div>
                            <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Type</Text>
                            <div style={{ marginTop: 4 }}>
                              <Tag color={getTypeColor(selectedCall.record?.tool_type)}>
                                {selectedCall.record?.tool_type}
                              </Tag>
                            </div>
                          </div>
                        </Col>
                        <Col span={12}>
                          <div>
                            <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Status</Text>
                            <div style={{ marginTop: 4, display: 'flex', alignItems: 'center', gap: 8 }}>
                              {selectedCall.record?.success ? (
                                <CheckCircleOutlined style={{ color: '#52c41a' }} />
                              ) : (
                                <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                              )}
                              <Text style={{ color: getStatusColor(selectedCall.record?.success || false), fontSize: 12 }}>
                                {selectedCall.record?.success ? 'Success' : 'Failed'}
                              </Text>
                            </div>
                          </div>
                        </Col>
                        <Col span={12}>
                          <div>
                            <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Duration</Text>
                            <div style={{ marginTop: 4 }}>
                              <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                                {formatDuration(selectedCall.record?.duration_ms || 0)}
                              </Text>
                            </div>
                          </div>
                        </Col>
                      </Row>

                      {/* UUID */}
                      <div>
                        <Text strong style={{ fontSize: 12, textAlign: 'left' }}>UUID</Text>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 4 }}>
                          <Text code style={{ fontSize: 11, flex: 1 }}>
                            {selectedCall.record?.uuid}
                          </Text>
                          <Button
                            type="text"
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyUuid(selectedCall.record?.uuid || '')}
                          />
                        </div>
                      </div>

                      {/* Server Info */}
                      {selectedCall.record?.server_name && (
                        <div>
                          <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Server</Text>
                          <div style={{ marginTop: 4 }}>
                            <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                              {selectedCall.record.server_name}
                            </Text>
                          </div>
                        </div>
                      )}

                      {/* Arguments */}
                      <div>
                        <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Arguments</Text>
                        <pre style={{ 
                          fontSize: 10, 
                          backgroundColor: '#f5f5f5', 
                          padding: 8, 
                          borderRadius: 4, 
                          marginTop: 4,
                          overflow: 'auto',
                          maxHeight: 120
                        }}>
                          {JSON.stringify(selectedCall.record?.arguments, null, 2)}
                        </pre>
                      </div>

                      {/* Result */}
                      <div>
                        <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Result</Text>
                        <pre style={{ 
                          fontSize: 10, 
                          backgroundColor: '#f5f5f5', 
                          padding: 8, 
                          borderRadius: 4, 
                          marginTop: 4,
                          overflow: 'auto',
                          maxHeight: 120
                        }}>
                          {JSON.stringify(selectedCall.record?.result, null, 2)}
                        </pre>
                      </div>

                      {/* Error */}
                      {!selectedCall.record?.success && selectedCall.record?.error_message && (
                        <div>
                          <Text strong style={{ fontSize: 12, color: '#ff4d4f', textAlign: 'left' }}>Error</Text>
                          <div style={{ 
                            marginTop: 4, 
                            padding: 8, 
                            backgroundColor: '#fff2f0', 
                            border: '1px solid #ffccc7', 
                            borderRadius: 4 
                          }}>
                            <Text style={{ fontSize: 12, color: '#a8071a' }}>
                              {selectedCall.record.error_message}
                            </Text>
                            {selectedCall.record.error_code && (
                              <div style={{ marginTop: 4 }}>
                                <Text style={{ fontSize: 11, color: '#ff4d4f' }}>
                                  Code: {selectedCall.record.error_code}
                                </Text>
                              </div>
                            )}
                          </div>
                        </div>
                      )}

                      {/* Performance Context */}
                      <Row gutter={[16, 16]}>
                        <Col span={12}>
                          <div>
                            <Text strong style={{ fontSize: 12, textAlign: 'left' }}>Performance Rank</Text>
                            <div style={{ marginTop: 4 }}>
                              <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                                {selectedCall.performance_context?.percentile_rank || 0}th percentile
                              </Text>
                            </div>
                          </div>
                        </Col>
                        <Col span={12}>
                          <div>
                            <Text strong style={{ fontSize: 12, textAlign: 'left' }}>vs Tool Average</Text>
                            <div style={{ marginTop: 4 }}>
                              <Text style={{ 
                                fontSize: 12,
                                color: (selectedCall.performance_context?.compared_to_tool_avg || 0) > 0 
                                  ? '#ff4d4f' 
                                  : '#52c41a'
                              }}>
                                {(selectedCall.performance_context?.compared_to_tool_avg || 0) > 0 ? '+' : ''}
                                {(selectedCall.performance_context?.compared_to_tool_avg || 0).toFixed(1)}ms
                              </Text>
                            </div>
                          </div>
                        </Col>
                      </Row>
                    </Space>
                  </Card>
                </TabPane>

                <TabPane
                  tab={
                    <Space>
                      <ClockCircleOutlined />
                      Timeline
                    </Space>
                  }
                  key="timeline"
                >
                  <Card
                    title={<div style={{ textAlign: 'left' }}>Execution Timeline</div>}
                    style={{ height: 520 }}
                    bodyStyle={{ height: 'calc(100% - 70px)', overflow: 'auto' }}
                  >
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      {Array.isArray(selectedCall?.timeline) ? selectedCall.timeline.map((event, index) => (
                        <div key={index} style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                          <div style={{
                            width: 12,
                            height: 12,
                            borderRadius: '50%',
                            backgroundColor: 
                              event.event_type === 'start' ? '#1890ff' :
                              event.event_type === 'end' ? '#52c41a' : '#ff4d4f'
                          }} />
                          <div style={{ flex: 1 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                              <Text strong style={{ fontSize: 12 }}>{event.message}</Text>
                              <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                                {formatTimestamp(event.timestamp)}
                              </Text>
                            </div>
                          </div>
                        </div>
                      )) : []}
                    </Space>
                  </Card>
                </TabPane>

                <TabPane
                  tab={
                    <Space>
                      <ArrowRightOutlined />
                      Context
                    </Space>
                  }
                  key="context"
                >
                  <Card
                    title={<div style={{ textAlign: 'left' }}>Related Calls</div>}
                    style={{ height: 520 }}
                    bodyStyle={{ height: 'calc(100% - 70px)', overflow: 'auto' }}
                  >
                    <Space direction="vertical" style={{ width: '100%' }} size="small">
                      {Array.isArray(selectedCall?.related_calls) ? selectedCall.related_calls.map((call) => (
                        <Card key={call.uuid} size="small" hoverable>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Space>
                              <Tag color={getTypeColor(call.tool_type)}>
                                {call.tool_type}
                              </Tag>
                              <Text strong style={{ fontSize: 12 }}>{call.tool_name}</Text>
                              {call.success ? (
                                <CheckCircleOutlined style={{ fontSize: 12, color: '#52c41a' }} />
                              ) : (
                                <CloseCircleOutlined style={{ fontSize: 12, color: '#ff4d4f' }} />
                              )}
                            </Space>
                            <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                              {formatDuration(call.duration_ms || 0)}
                            </Text>
                          </div>
                          <div style={{ marginTop: 4 }}>
                            <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                              {formatTimestamp(call.created_at)}
                            </Text>
                          </div>
                        </Card>
                      )) : []}
                    </Space>
                  </Card>
                </TabPane>
              </Tabs>
            ) : (
              <Card style={{ height: '100%' }}>
                <div style={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  justifyContent: 'center', 
                  height: '100%' 
                }}>
                  <Empty
                    image={<SettingOutlined style={{ fontSize: 48, color: '#bfbfbf' }} />}
                    description={
                      <div>
                        <Title level={4} type="secondary" style={{ textAlign: 'left' }}>No tool call selected</Title>
                        <Text type="secondary" style={{ textAlign: 'left' }}>
                          Select a tool call from the list to view detailed information
                        </Text>
                      </div>
                    }
                  />
                </div>
              </Card>
            )}
          </div>
        </div>
      </Space>
    </div>
  )
}