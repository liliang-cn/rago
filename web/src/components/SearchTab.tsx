import { useState } from 'react'
import { Card, Button, Input, Tabs, Space, Typography, Tag, Alert, Slider, Switch, Empty } from 'antd'
import { SearchOutlined, FilterOutlined, ThunderboltOutlined, PlusOutlined } from '@ant-design/icons'
import { apiClient, SearchRequest, SearchResult } from '../lib/api'

const { Text, Title } = Typography
const { TabPane } = Tabs

export function SearchTab() {
  const [searchRequest, setSearchRequest] = useState<SearchRequest>({
    query: '',
    top_k: 10,
    score_threshold: 0,
    hybrid_search: false,
    vector_weight: 0.7,
    include_content: true,
  })
  
  const [searchResults, setSearchResults] = useState<{
    semantic: SearchResult[]
    hybrid: SearchResult[]
    filtered: SearchResult[]
  }>({
    semantic: [],
    hybrid: [],
    filtered: [],
  })
  
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('semantic')
  const [filters, setFilters] = useState<Record<string, string>>({})

  const handleSemanticSearch = async () => {
    if (!searchRequest.query) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await apiClient.semanticSearch(searchRequest)
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setSearchResults(prev => ({
          ...prev,
          semantic: response.data!.results || []
        }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }

  const handleHybridSearch = async () => {
    if (!searchRequest.query) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await apiClient.hybridSearch({
        ...searchRequest,
        hybrid_search: true,
      })
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setSearchResults(prev => ({
          ...prev,
          hybrid: response.data!.results || []
        }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }

  const handleFilteredSearch = async () => {
    if (!searchRequest.query) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await apiClient.filteredSearch({
        ...searchRequest,
        filters: Object.keys(filters).length > 0 ? filters : undefined,
      })
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setSearchResults(prev => ({
          ...prev,
          filtered: response.data!.results || []
        }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }

  const renderSearchResults = (results: SearchResult[]) => {
    if (results.length === 0) {
      return (
        <Empty
          description="No results found. Try adjusting your search parameters."
          style={{ padding: '32px 0' }}
        />
      )
    }

    return (
      <div style={{ maxHeight: 400, overflowY: 'auto', paddingRight: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          {results?.map((result, index) => (
            <Card 
              key={result.id || index} 
              size="small" 
              style={{ borderLeft: '4px solid #1890ff' }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
                <Title level={5} style={{ margin: 0, fontSize: 14, textAlign: 'left' }}>
                  Document: {result.id}
                </Title>
                <Tag color="blue">
                  Score: {result.score?.toFixed(3)}
                </Tag>
              </div>
              
              {searchRequest.include_content && (
                <Text 
                  type="secondary" 
                  style={{ 
                    fontSize: 12, 
                    marginBottom: 8,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    display: '-webkit-box',
                    WebkitLineClamp: 3,
                    WebkitBoxOrient: 'vertical'
                  }}
                >
                  {result.content}
                </Text>
              )}
              
              {result.metadata && Object.keys(result.metadata)?.length > 0 && (
                <Space wrap>
                  {Object.entries(result.metadata)?.map(([key, value]) => (
                    <Tag key={key} color="default" style={{ fontSize: 10 }}>
                      {key}: {String(value)}
                    </Tag>
                  ))}
                </Space>
              )}
            </Card>
          ))}
        </Space>
      </div>
    )
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <Card
          title={
            <div style={{ textAlign: 'left' }}>
              <Space>
                <SearchOutlined />
                <span>Advanced Search</span>
              </Space>
            </div>
          }
        >
          <Text type="secondary" style={{ marginBottom: 16, display: 'block', textAlign: 'left' }}>
            Explore different search strategies with fine-tuned parameters
          </Text>
          
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Text strong style={{ marginBottom: 8, display: 'block', textAlign: 'left' }}>Search Query</Text>
              <Input
                value={searchRequest.query}
                onChange={(e) => setSearchRequest(prev => ({ ...prev, query: e.target.value }))}
                placeholder="Enter your search query..."
                prefix={<SearchOutlined />}
                onPressEnter={() => {
                  if (activeTab === 'semantic') handleSemanticSearch()
                  else if (activeTab === 'hybrid') handleHybridSearch()
                  else if (activeTab === 'filtered') handleFilteredSearch()
                }}
              />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
              <div>
                <Text strong style={{ marginBottom: 8, display: 'block', textAlign: 'left' }}>
                  Top K Results: {searchRequest.top_k}
                </Text>
                <Slider
                  min={1}
                  max={50}
                  step={1}
                  value={searchRequest.top_k || 10}
                  onChange={(value: number) => setSearchRequest(prev => ({ ...prev, top_k: value }))}
                />
              </div>

              <div>
                <Text strong style={{ marginBottom: 8, display: 'block', textAlign: 'left' }}>
                  Score Threshold: {searchRequest.score_threshold?.toFixed(2) || '0.00'}
                </Text>
                <Slider
                  min={0}
                  max={1}
                  step={0.05}
                  value={searchRequest.score_threshold || 0}
                  onChange={(value: number) => setSearchRequest(prev => ({ ...prev, score_threshold: value }))}
                />
              </div>
            </div>

            <Card size="small" style={{ backgroundColor: '#fafafa' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                <Text strong style={{ textAlign: 'left' }}>Enable Hybrid Search</Text>
                <Switch
                  checked={searchRequest.hybrid_search}
                  onChange={(checked: boolean) => setSearchRequest(prev => ({ ...prev, hybrid_search: checked }))}
                />
              </div>
              
              {searchRequest.hybrid_search && (
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  <div>
                    <Text strong style={{ marginBottom: 8, display: 'block', textAlign: 'left' }}>
                      Vector Weight: {((searchRequest.vector_weight || 0.7) * 100).toFixed(0)}%
                    </Text>
                    <Slider
                      min={0}
                      max={1}
                      step={0.05}
                      value={searchRequest.vector_weight || 0.7}
                      onChange={(value: number) => setSearchRequest(prev => ({ ...prev, vector_weight: value }))}
                    />
                    <Text type="secondary" style={{ fontSize: 12, textAlign: 'left', display: 'block' }}>
                      Keyword Weight: {((1 - (searchRequest.vector_weight || 0.7)) * 100).toFixed(0)}%
                    </Text>
                  </div>
                </Space>
              )}
            </Card>

            <Card size="small" style={{ backgroundColor: '#fafafa' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text strong style={{ textAlign: 'left' }}>Include Full Content in Results</Text>
                <Switch
                  checked={searchRequest.include_content}
                  onChange={(checked: boolean) => setSearchRequest(prev => ({ ...prev, include_content: checked }))}
                />
              </div>
            </Card>
          </Space>
        </Card>

        <Card
          title={
            <div style={{ textAlign: 'left' }}>
              <Space>
                <SearchOutlined />
                <span>Search Results</span>
              </Space>
            </div>
          }
          style={{ flex: 1 }}
        >
          <Tabs activeKey={activeTab} onChange={setActiveTab} type="card">
            <TabPane 
              tab={
                <Space>
                  <SearchOutlined />
                  Semantic
                </Space>
              } 
              key="semantic"
            >

              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type="secondary" style={{ textAlign: 'left' }}>
                    Pure vector similarity search
                  </Text>
                  <Button
                    type="primary"
                    onClick={handleSemanticSearch}
                    disabled={loading || !searchRequest.query}
                    loading={loading}
                    icon={<SearchOutlined />}
                  >
                    Search
                  </Button>
                </div>
                {error && (
                  <Alert
                    message={error}
                    type="error"
                    showIcon
                    closable
                    onClose={() => setError(null)}
                  />
                )}
                {renderSearchResults(searchResults.semantic)}
              </Space>
            </TabPane>

            <TabPane 
              tab={
                <Space>
                  <ThunderboltOutlined />
                  Hybrid
                </Space>
              } 
              key="hybrid"
            >
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type="secondary" style={{ textAlign: 'left' }}>
                    Combined vector and keyword search
                  </Text>
                  <Button
                    type="primary"
                    onClick={handleHybridSearch}
                    disabled={loading || !searchRequest.query}
                    loading={loading}
                    icon={<ThunderboltOutlined />}
                  >
                    Search
                  </Button>
                </div>
                {error && (
                  <Alert
                    message={error}
                    type="error"
                    showIcon
                    closable
                    onClose={() => setError(null)}
                  />
                )}
                {renderSearchResults(searchResults.hybrid)}
              </Space>
            </TabPane>

            <TabPane 
              tab={
                <Space>
                  <FilterOutlined />
                  Filtered
                </Space>
              } 
              key="filtered"
            >
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                <div>
                  <Text strong style={{ marginBottom: 8, display: 'block', textAlign: 'left' }}>Metadata Filters</Text>
                  <Space.Compact style={{ width: '100%' }}>
                    <Input
                      placeholder="Key"
                      id="filter-key"
                      style={{ width: '30%' }}
                    />
                    <Input
                      placeholder="Value"
                      id="filter-value"
                      style={{ width: '50%' }}
                    />
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        const key = (document.getElementById('filter-key') as HTMLInputElement)?.value
                        const value = (document.getElementById('filter-value') as HTMLInputElement)?.value
                        if (key && value) {
                          setFilters(prev => ({ ...prev, [key]: value }));
                          (document.getElementById('filter-key') as HTMLInputElement).value = '';
                          (document.getElementById('filter-value') as HTMLInputElement).value = ''
                        }
                      }}
                      style={{ width: '20%' }}
                    >
                      Add
                    </Button>
                  </Space.Compact>
                  <div style={{ marginTop: 8 }}>
                    <Space wrap>
                      {Object.entries(filters)?.map(([key, value]) => (
                        <Tag
                          key={key}
                          closable
                          onClose={() => setFilters(prev => {
                            const newFilters = { ...prev }
                            delete newFilters[key]
                            return newFilters
                          })}
                          color="blue"
                        >
                          {key}: {value}
                        </Tag>
                      ))}
                    </Space>
                  </div>
                </div>
                
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type="secondary" style={{ textAlign: 'left' }}>
                    Search with metadata filtering
                  </Text>
                  <Button
                    type="primary"
                    onClick={handleFilteredSearch}
                    disabled={loading || !searchRequest.query}
                    loading={loading}
                    icon={<FilterOutlined />}
                  >
                    Search
                  </Button>
                </div>
                
                {error && (
                  <Alert
                    message={error}
                    type="error"
                    showIcon
                    closable
                    onClose={() => setError(null)}
                  />
                )}
                {renderSearchResults(searchResults.filtered)}
              </Space>
            </TabPane>
          </Tabs>
        </Card>
      </Space>
    </div>
  )
}