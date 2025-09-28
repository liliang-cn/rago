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
  Tabs,
  Select,
  Empty,
  Tag,
} from 'antd'
import { 
  SearchOutlined, 
  RiseOutlined, 
  FallOutlined, 
  WarningOutlined,
  ClockCircleOutlined, 
  AimOutlined, 
  CheckCircleOutlined,
  CloseCircleOutlined,
  BarChartOutlined,
  EyeOutlined,
  ReloadOutlined,
  CalendarOutlined,
  ExclamationCircleOutlined,
  ThunderboltOutlined,
  BulbOutlined,
  SettingOutlined,
  ArrowRightOutlined,
  DatabaseOutlined
} from '@ant-design/icons'
import { 
  apiClient, 
  RAGQueryRecord, 
  RAGQueryVisualization, 
  RAGAnalytics, 
  RAGPerformanceReport,
  RAGSearchFilter 
} from '@/lib/api'

const { Title, Text, Paragraph } = Typography
const { TabPane } = Tabs
const { Option } = Select

export function RAGVisualizationTab() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [queries, setQueries] = useState<RAGQueryRecord[]>([])
  const [selectedQuery, setSelectedQuery] = useState<RAGQueryVisualization | null>(null)
  const [analytics, setAnalytics] = useState<RAGAnalytics | null>(null)
  const [performanceReport, setPerformanceReport] = useState<RAGPerformanceReport | null>(null)
  
  const [searchTerm, setSearchTerm] = useState('')
  const [timeRange, setTimeRange] = useState<string>('7d')
  const [conversationFilter, setConversationFilter] = useState('')

  const timeRanges = {
    '1d': { label: '1 Day', days: 1 },
    '7d': { label: '7 Days', days: 7 },
    '30d': { label: '30 Days', days: 30 },
    '90d': { label: '90 Days', days: 90 },
  }

  const buildFilter = (): RAGSearchFilter => {
    const now = new Date()
    const startTime = new Date(now.getTime() - timeRanges[timeRange as keyof typeof timeRanges].days * 24 * 60 * 60 * 1000)
    
    return {
      start_time: Math.floor(startTime.getTime() / 1000).toString(),
      end_time: Math.floor(now.getTime() / 1000).toString(),
      conversation_id: conversationFilter || undefined,
      query: searchTerm || undefined,
      limit: 50,
    }
  }

  const fetchData = async () => {
    setLoading(true)
    setError(null)

    try {
      const filter = buildFilter()
      
      const [queriesResponse, analyticsResponse, performanceResponse] = await Promise.all([
        apiClient.getRAGQueries(filter),
        apiClient.getRAGAnalytics(filter),
        apiClient.getRAGPerformanceReport(filter),
      ])

      if (queriesResponse.error) throw new Error(queriesResponse.error)
      if (analyticsResponse.error) throw new Error(analyticsResponse.error)
      if (performanceResponse.error) throw new Error(performanceResponse.error)

      setQueries(Array.isArray(queriesResponse.data) ? queriesResponse.data : [])
      setAnalytics(analyticsResponse.data || null)
      setPerformanceReport(performanceResponse.data || null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch RAG data')
    } finally {
      setLoading(false)
    }
  }

  const fetchQueryVisualization = async (queryId: string) => {
    setLoading(true)
    setError(null)

    try {
      const response = await apiClient.getRAGVisualization(queryId)
      if (response.error) throw new Error(response.error)
      setSelectedQuery(response.data || null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch query visualization')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [timeRange, searchTerm, conversationFilter])

  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
    return num?.toString() || '0'
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'improving':
        return <RiseOutlined style={{ color: '#52c41a' }} />
      case 'declining':
        return <FallOutlined style={{ color: '#ff4d4f' }} />
      default:
        return <ArrowRightOutlined style={{ color: '#8c8c8c' }} />
    }
  }

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'red'
      case 'high':
        return 'red'
      case 'medium':
        return 'orange'
      case 'low':
        return 'blue'
      default:
        return 'default'
    }
  }

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'red'
      case 'medium':
        return 'orange'
      case 'low':
        return 'blue'
      default:
        return 'default'
    }
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
            <Title level={2} style={{ margin: 0 }}>RAG Visualization</Title>
            <Text type="secondary">
              Analyze RAG query performance, chunk retrieval, and answer quality
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
              onClick={fetchData} 
              disabled={loading} 
              loading={loading}
              icon={<ReloadOutlined />}
            >
              Refresh
            </Button>
          </Space>
        </div>

        {/* Filters */}
        <Row gutter={16}>
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="Search queries..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              prefix={<SearchOutlined />}
            />
          </Col>
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="Conversation ID filter"
              value={conversationFilter}
              onChange={(e) => setConversationFilter(e.target.value)}
            />
          </Col>
        </Row>

        <Tabs defaultActiveKey="overview" style={{ flex: 1 }}>
          <TabPane
            tab={
              <Space>
                <BarChartOutlined />
                Overview
              </Space>
            }
            key="overview"
          >
            <Space direction="vertical" style={{ width: '100%' }} size="large">
              {/* Overview Cards */}
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} md={6}>
                  <Card size="small">
                    <Statistic
                      title="Total Queries"
                      value={analytics ? formatNumber(analytics.total_queries || 0) : '—'}
                      prefix={<BulbOutlined />}
                      valueStyle={{ color: '#1890ff' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                      Success rate: {analytics ? ((analytics.success_rate || 0) * 100).toFixed(1) : '—'}%
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={6}>
                  <Card size="small">
                    <Statistic
                      title="Avg Latency"
                      value={analytics ? `${(analytics.avg_latency || 0).toFixed(0)}ms` : '—'}
                      prefix={<ClockCircleOutlined />}
                      valueStyle={{ color: '#52c41a' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                      Response time
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={6}>
                  <Card size="small">
                    <Statistic
                      title="Avg Chunks"
                      value={analytics ? (analytics.avg_chunks || 0).toFixed(1) : '—'}
                      prefix={<DatabaseOutlined />}
                      valueStyle={{ color: '#722ed1' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                      Retrieved per query
                    </Text>
                  </Card>
                </Col>

                <Col xs={24} sm={12} md={6}>
                  <Card size="small">
                    <Statistic
                      title="Avg Score"
                      value={analytics ? (analytics.avg_score || 0).toFixed(3) : '—'}
                      prefix={<AimOutlined />}
                      valueStyle={{ color: '#f5222d' }}
                    />
                    <Text style={{ fontSize: 12, color: '#8c8c8c' }}>
                      Relevance score
                    </Text>
                  </Card>
                </Col>
              </Row>

              {/* Performance Distribution */}
              {analytics && (
                <Row gutter={[24, 24]}>
                  <Col xs={24} lg={12}>
                    <Card
                      title={
                        <Space>
                          <BarChartOutlined />
                          Performance Distribution
                        </Space>
                      }
                    >
                      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                        Query performance breakdown by latency
                      </Text>
                      <Space direction="vertical" style={{ width: '100%' }} size="middle">
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space>
                            <div style={{ width: 12, height: 12, backgroundColor: '#52c41a', borderRadius: 2 }}></div>
                            <Text>Fast (&lt; 1s)</Text>
                          </Space>
                          <Text strong>{analytics.fast_queries || 0}</Text>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space>
                            <div style={{ width: 12, height: 12, backgroundColor: '#faad14', borderRadius: 2 }}></div>
                            <Text>Medium (1-5s)</Text>
                          </Space>
                          <Text strong>{analytics.medium_queries || 0}</Text>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space>
                            <div style={{ width: 12, height: 12, backgroundColor: '#ff4d4f', borderRadius: 2 }}></div>
                            <Text>Slow (&gt; 5s)</Text>
                          </Space>
                          <Text strong>{analytics.slow_queries || 0}</Text>
                        </div>
                      </Space>
                    </Card>
                  </Col>

                  <Col xs={24} lg={12}>
                    <Card
                      title={
                        <Space>
                          <AimOutlined />
                          Quality Distribution
                        </Space>
                      }
                    >
                      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                        Answer quality breakdown by score
                      </Text>
                      <Space direction="vertical" style={{ width: '100%' }} size="middle">
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space>
                            <div style={{ width: 12, height: 12, backgroundColor: '#52c41a', borderRadius: 2 }}></div>
                            <Text>High (&gt; 0.8)</Text>
                          </Space>
                          <Text strong>{analytics.high_quality_queries || 0}</Text>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space>
                            <div style={{ width: 12, height: 12, backgroundColor: '#faad14', borderRadius: 2 }}></div>
                            <Text>Medium (0.5-0.8)</Text>
                          </Space>
                          <Text strong>{analytics.medium_quality_queries || 0}</Text>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space>
                            <div style={{ width: 12, height: 12, backgroundColor: '#ff4d4f', borderRadius: 2 }}></div>
                            <Text>Low (&lt; 0.5)</Text>
                          </Space>
                          <Text strong>{analytics.low_quality_queries || 0}</Text>
                        </div>
                      </Space>
                    </Card>
                  </Col>
                </Row>
              )}
            </Space>
          </TabPane>

          <TabPane
            tab={
              <Space>
                <SearchOutlined />
                Queries
              </Space>
            }
            key="queries"
          >
            <div style={{ display: 'flex', height: 600 }}>
              {/* Queries List */}
              <div style={{ width: '50%', borderRight: '1px solid #f0f0f0', paddingRight: 16 }}>
                <Title level={4} style={{ marginBottom: 16 }}>Recent Queries</Title>
                <div style={{ height: 550, overflow: 'auto' }}>
                  <List
                    loading={loading}
                    dataSource={Array.isArray(queries) ? queries : []}
                    renderItem={(query) => (
                      <List.Item style={{ padding: 0, marginBottom: 8 }}>
                        <Card
                          size="small"
                          hoverable
                          onClick={() => fetchQueryVisualization(query.id)}
                          style={{ 
                            width: '100%',
                            cursor: 'pointer',
                            backgroundColor: selectedQuery?.query.id === query.id ? '#e6f7ff' : undefined,
                            border: selectedQuery?.query.id === query.id ? '1px solid #1890ff' : undefined
                          }}
                          bodyStyle={{ padding: 12 }}
                        >
                          <Space direction="vertical" style={{ width: '100%' }} size="small">
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <Space>
                                {query.success ? (
                                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                                ) : (
                                  <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                                )}
                                <Text strong style={{ fontSize: 12 }}>
                                  {query.id?.slice(0, 8) || 'Unknown'}...
                                </Text>
                              </Space>
                              <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                                {query.total_latency || 0}ms
                              </Text>
                            </div>
                            <Paragraph 
                              style={{ margin: 0, fontSize: 12 }} 
                              ellipsis={{ rows: 1, tooltip: true }}
                            >
                              {query.query}
                            </Paragraph>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <Badge count={query.chunks_found || 0} size="small" color="#1890ff" />
                              <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                                {formatDate(query.created_at)}
                              </Text>
                            </div>
                          </Space>
                        </Card>
                      </List.Item>
                    )}
                  />
                </div>
              </div>

              {/* Query Details */}
              <div style={{ width: '50%', paddingLeft: 16 }}>
                {selectedQuery ? (
                  <div style={{ height: 550, overflow: 'auto' }}>
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      <div>
                        <Title level={4}>Query Details</Title>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {selectedQuery.query?.query}
                        </Text>
                      </div>

                      {/* Metrics */}
                      <Row gutter={[8, 8]}>
                        <Col span={12}>
                          <Card size="small">
                            <div style={{ textAlign: 'center' }}>
                              <Text type="secondary" style={{ fontSize: 12 }}>Total Time</Text>
                              <div style={{ fontSize: 18, fontWeight: 'bold', color: '#1890ff' }}>
                                {selectedQuery.query?.total_latency || 0}ms
                              </div>
                            </div>
                          </Card>
                        </Col>
                        <Col span={12}>
                          <Card size="small">
                            <div style={{ textAlign: 'center' }}>
                              <Text type="secondary" style={{ fontSize: 12 }}>Chunks Found</Text>
                              <div style={{ fontSize: 18, fontWeight: 'bold', color: '#52c41a' }}>
                                {selectedQuery.query?.chunks_found || 0}
                              </div>
                            </div>
                          </Card>
                        </Col>
                      </Row>

                      {/* Retrieval Metrics */}
                      <Card title={<Text strong style={{ fontSize: 14 }}>Retrieval Metrics</Text>} size="small">
                        <Space direction="vertical" style={{ width: '100%' }} size="small">
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text style={{ fontSize: 12 }}>Average Score:</Text>
                            <Text strong style={{ fontSize: 12 }}>
                              {selectedQuery.retrieval_metrics?.average_score?.toFixed(3) || '—'}
                            </Text>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text style={{ fontSize: 12 }}>Top Score:</Text>
                            <Text strong style={{ fontSize: 12 }}>
                              {selectedQuery.retrieval_metrics?.top_score?.toFixed(3) || '—'}
                            </Text>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text style={{ fontSize: 12 }}>Diversity:</Text>
                            <Text strong style={{ fontSize: 12 }}>
                              {selectedQuery.retrieval_metrics?.diversity_score?.toFixed(3) || '—'}
                            </Text>
                          </div>
                        </Space>
                      </Card>

                      {/* Quality Metrics */}
                      <Card title={<Text strong style={{ fontSize: 14 }}>Quality Metrics</Text>} size="small">
                        <Space direction="vertical" style={{ width: '100%' }} size="small">
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text style={{ fontSize: 12 }}>Source Utilization:</Text>
                            <Text strong style={{ fontSize: 12 }}>
                              {((selectedQuery.quality_metrics?.source_utilization || 0) * 100).toFixed(1)}%
                            </Text>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text style={{ fontSize: 12 }}>Confidence:</Text>
                            <Text strong style={{ fontSize: 12 }}>
                              {selectedQuery.quality_metrics?.confidence_score?.toFixed(3) || '—'}
                            </Text>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text style={{ fontSize: 12 }}>Factuality:</Text>
                            <Text strong style={{ fontSize: 12 }}>
                              {selectedQuery.quality_metrics?.factuality_score?.toFixed(3) || '—'}
                            </Text>
                          </div>
                        </Space>
                      </Card>

                      {/* Chunk Hits */}
                      {Array.isArray(selectedQuery.chunk_hits) && selectedQuery.chunk_hits.length > 0 && (
                        <Card 
                          title={
                            <Text strong style={{ fontSize: 14 }}>
                              Chunk Hits ({selectedQuery.chunk_hits.length})
                            </Text>
                          } 
                          size="small"
                        >
                          <Space direction="vertical" style={{ width: '100%' }} size="small">
                            {selectedQuery.chunk_hits.slice(0, 5).map((hit) => (
                              <div key={hit.id} style={{ 
                                border: '1px solid #f0f0f0', 
                                borderRadius: 4, 
                                padding: 8 
                              }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                                  <Tag color="blue" style={{ fontSize: 10 }}>
                                    Rank {hit.rank}
                                  </Tag>
                                  <Space size="small">
                                    <Text strong style={{ fontSize: 11 }}>
                                      {hit.score?.toFixed(3) || '—'}
                                    </Text>
                                    {hit.used_in_generation && (
                                      <Tag color="green" style={{ fontSize: 10 }}>
                                        Used
                                      </Tag>
                                    )}
                                  </Space>
                                </div>
                                <Paragraph 
                                  style={{ margin: 0, fontSize: 11, color: '#8c8c8c' }} 
                                  ellipsis={{ rows: 1, tooltip: true }}
                                >
                                  {hit.content}
                                </Paragraph>
                              </div>
                            ))}
                          </Space>
                        </Card>
                      )}

                      {/* Tool Calls */}
                      {Array.isArray(selectedQuery.tool_calls) && selectedQuery.tool_calls.length > 0 && (
                        <Card 
                          title={
                            <Text strong style={{ fontSize: 14 }}>
                              Tool Calls ({selectedQuery.tool_calls.length})
                            </Text>
                          } 
                          size="small"
                        >
                          <Space direction="vertical" style={{ width: '100%' }} size="small">
                            {selectedQuery.tool_calls.map((call) => (
                              <div key={call.id} style={{ 
                                border: '1px solid #f0f0f0', 
                                borderRadius: 4, 
                                padding: 8 
                              }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                  <Text strong style={{ fontSize: 12 }}>{call.tool_name}</Text>
                                  <Space size="small">
                                    <Text style={{ fontSize: 10 }}>{call.duration}ms</Text>
                                    {call.success ? (
                                      <CheckCircleOutlined style={{ fontSize: 12, color: '#52c41a' }} />
                                    ) : (
                                      <CloseCircleOutlined style={{ fontSize: 12, color: '#ff4d4f' }} />
                                    )}
                                  </Space>
                                </div>
                                {call.error_message && (
                                  <Text style={{ fontSize: 10, color: '#ff4d4f', display: 'block', marginTop: 4 }}>
                                    {call.error_message}
                                  </Text>
                                )}
                              </div>
                            ))}
                          </Space>
                        </Card>
                      )}
                    </Space>
                  </div>
                ) : (
                  <div style={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    justifyContent: 'center', 
                    height: '100%' 
                  }}>
                    <Empty
                      image={<EyeOutlined style={{ fontSize: 48, color: '#bfbfbf' }} />}
                      description={
                        <div>
                          <Title level={4} type="secondary">No query selected</Title>
                          <Text type="secondary">Select a query to view detailed visualization</Text>
                        </div>
                      }
                    />
                  </div>
                )}
              </div>
            </div>
          </TabPane>

          <TabPane
            tab={
              <Space>
                <ThunderboltOutlined />
                Analytics
              </Space>
            }
            key="analytics"
          >
            {analytics && (
              <Row gutter={[24, 24]}>
                <Col xs={24} lg={12}>
                  <Card
                    title={
                      <Space>
                        <ThunderboltOutlined />
                        Performance Metrics
                      </Space>
                    }
                  >
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>Success Rate:</Text>
                        <Text strong>{((analytics.success_rate || 0) * 100).toFixed(1)}%</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>Average Latency:</Text>
                        <Text strong>{(analytics.avg_latency || 0).toFixed(0)}ms</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>Average Chunks:</Text>
                        <Text strong>{(analytics.avg_chunks || 0).toFixed(1)}</Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>Average Score:</Text>
                        <Text strong>{(analytics.avg_score || 0).toFixed(3)}</Text>
                      </div>
                    </Space>
                  </Card>
                </Col>

                <Col xs={24} lg={12}>
                  <Card
                    title={
                      <Space>
                        <CalendarOutlined />
                        Time Range
                      </Space>
                    }
                  >
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>Start Time:</Text>
                        <Text strong style={{ fontSize: 12 }}>
                          {new Date(analytics.start_time).toLocaleString()}
                        </Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>End Time:</Text>
                        <Text strong style={{ fontSize: 12 }}>
                          {new Date(analytics.end_time).toLocaleString()}
                        </Text>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Text>Total Queries:</Text>
                        <Text strong>{analytics.total_queries || 0}</Text>
                      </div>
                    </Space>
                  </Card>
                </Col>
              </Row>
            )}
          </TabPane>

          <TabPane
            tab={
              <Space>
                <BarChartOutlined />
                Performance
              </Space>
            }
            key="performance"
          >
            {performanceReport && (
              <Space direction="vertical" style={{ width: '100%' }} size="large">
                {/* Performance Overview */}
                <Row gutter={[16, 16]}>
                  <Col xs={24} sm={12} md={6}>
                    <Card size="small">
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Text strong style={{ fontSize: 12 }}>P95 Latency</Text>
                        <ClockCircleOutlined style={{ color: '#8c8c8c' }} />
                      </div>
                      <div style={{ fontSize: 20, fontWeight: 'bold', marginTop: 8 }}>
                        {(performanceReport.p95_latency || 0).toFixed(0)}ms
                      </div>
                    </Card>
                  </Col>

                  <Col xs={24} sm={12} md={6}>
                    <Card size="small">
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Text strong style={{ fontSize: 12 }}>Retrieval Ratio</Text>
                        <BarChartOutlined style={{ color: '#8c8c8c' }} />
                      </div>
                      <div style={{ fontSize: 20, fontWeight: 'bold', marginTop: 8 }}>
                        {((performanceReport.retrieval_ratio || 0) * 100).toFixed(1)}%
                      </div>
                    </Card>
                  </Col>

                  <Col xs={24} sm={12} md={6}>
                    <Card size="small">
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Text strong style={{ fontSize: 12 }}>Latency Trend</Text>
                        {getTrendIcon(performanceReport.latency_trend || 'stable')}
                      </div>
                      <div style={{ fontSize: 16, fontWeight: 'bold', marginTop: 8, textTransform: 'capitalize' }}>
                        {performanceReport.latency_trend || 'stable'}
                      </div>
                    </Card>
                  </Col>

                  <Col xs={24} sm={12} md={6}>
                    <Card size="small">
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Text strong style={{ fontSize: 12 }}>Quality Trend</Text>
                        {getTrendIcon(performanceReport.quality_trend || 'stable')}
                      </div>
                      <div style={{ fontSize: 16, fontWeight: 'bold', marginTop: 8, textTransform: 'capitalize' }}>
                        {performanceReport.quality_trend || 'stable'}
                      </div>
                    </Card>
                  </Col>
                </Row>

                {/* Issues */}
                {((Array.isArray(performanceReport.slow_queries) && performanceReport.slow_queries.length > 0) || (Array.isArray(performanceReport.low_quality_queries) && performanceReport.low_quality_queries.length > 0)) && (
                  <Row gutter={[24, 24]}>
                    {/* Slow Queries */}
                    {Array.isArray(performanceReport.slow_queries) && performanceReport.slow_queries.length > 0 && (
                      <Col xs={24} lg={12}>
                        <Card
                          title={
                            <Space>
                              <WarningOutlined style={{ color: '#faad14' }} />
                              Slow Queries ({performanceReport.slow_queries?.length || 0})
                            </Space>
                          }
                        >
                          <Space direction="vertical" style={{ width: '100%' }} size="small">
                            {(performanceReport.slow_queries || []).slice(0, 5).map((issue) => (
                              <div key={issue.query_id} style={{ 
                                border: '1px solid #f0f0f0', 
                                borderRadius: 4, 
                                padding: 12 
                              }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                                  <Tag color={getSeverityColor(issue.severity)}>
                                    {issue.severity}
                                  </Tag>
                                  <Text strong style={{ fontSize: 12 }}>{issue.latency}ms</Text>
                                </div>
                                <Paragraph 
                                  style={{ margin: 0, fontSize: 12, color: '#8c8c8c' }} 
                                  ellipsis={{ rows: 1, tooltip: true }}
                                >
                                  {issue.query}
                                </Paragraph>
                                <Text style={{ fontSize: 10, color: '#8c8c8c', display: 'block', marginTop: 4 }}>
                                  {issue.issue}
                                </Text>
                              </div>
                            ))}
                          </Space>
                        </Card>
                      </Col>
                    )}

                    {/* Quality Issues */}
                    {Array.isArray(performanceReport.low_quality_queries) && performanceReport.low_quality_queries.length > 0 && (
                      <Col xs={24} lg={12}>
                        <Card
                          title={
                            <Space>
                              <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
                              Quality Issues ({performanceReport.low_quality_queries?.length || 0})
                            </Space>
                          }
                        >
                          <Space direction="vertical" style={{ width: '100%' }} size="small">
                            {(performanceReport.low_quality_queries || []).slice(0, 5).map((issue) => (
                              <div key={issue.query_id} style={{ 
                                border: '1px solid #f0f0f0', 
                                borderRadius: 4, 
                                padding: 12 
                              }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                                  <Tag color={getSeverityColor(issue.severity)}>
                                    {issue.severity}
                                  </Tag>
                                  <Text style={{ fontSize: 10, color: '#8c8c8c' }}>
                                    {new Date(issue.timestamp).toLocaleString()}
                                  </Text>
                                </div>
                                <Paragraph 
                                  style={{ margin: 0, fontSize: 12, color: '#8c8c8c' }} 
                                  ellipsis={{ rows: 1, tooltip: true }}
                                >
                                  {issue.query}
                                </Paragraph>
                                <Text style={{ fontSize: 10, color: '#8c8c8c', display: 'block', marginTop: 4 }}>
                                  {issue.issue}
                                </Text>
                              </div>
                            ))}
                          </Space>
                        </Card>
                      </Col>
                    )}
                  </Row>
                )}

                {/* Recommendations */}
                {Array.isArray(performanceReport.recommendations) && performanceReport.recommendations.length > 0 && (
                  <Card
                    title={
                      <Space>
                        <SettingOutlined />
                        Optimization Recommendations
                      </Space>
                    }
                  >
                    <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                      Suggestions to improve RAG performance
                    </Text>
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                      {(performanceReport.recommendations || []).map((rec, index) => (
                        <div key={index} style={{ 
                          border: '1px solid #f0f0f0', 
                          borderRadius: 4, 
                          padding: 16 
                        }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                            <Title level={5} style={{ margin: 0 }}>{rec.title}</Title>
                            <Tag color={getPriorityColor(rec.priority)}>
                              {rec.priority} priority
                            </Tag>
                          </div>
                          <Paragraph style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 8 }}>
                            {rec.description}
                          </Paragraph>
                          <div style={{ fontSize: 12 }}>
                            <Text strong>Expected Impact:</Text> {rec.impact}
                          </div>
                        </div>
                      ))}
                    </Space>
                  </Card>
                )}
              </Space>
            )}
          </TabPane>
        </Tabs>
      </Space>
    </div>
  )
}