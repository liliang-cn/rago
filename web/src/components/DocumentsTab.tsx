import { useState, useEffect } from 'react'
import { Card, Button, Input, Space, Typography, Spin, Empty, Tooltip, Tag } from 'antd'
import { FileTextOutlined, DeleteOutlined, ExclamationCircleOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import { apiClient, Document } from '@/lib/api'

const { Title, Text, Paragraph } = Typography

export function DocumentsTab() {
  const [documents, setDocuments] = useState<Document[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [deleteLoading, setDeleteLoading] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [filteredDocuments, setFilteredDocuments] = useState<Document[]>([])

  const loadDocuments = async () => {
    setIsLoading(true)
    try {
      const response = await apiClient.getDocuments()
      if (response.success && response.data) {
        const docs = Array.isArray(response.data) ? response.data : []
        setDocuments(docs)
        setFilteredDocuments(docs)
      } else {
        setDocuments([])
        setFilteredDocuments([])
      }
    } catch (error) {
      console.error('Failed to load documents:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this document?')) return

    setDeleteLoading(id)
    try {
      await apiClient.deleteDocument(id)
      setDocuments(prev => prev.filter(doc => doc.id !== id))
      setFilteredDocuments(prev => prev.filter(doc => doc.id !== id))
    } catch (error) {
      console.error('Failed to delete document:', error)
    } finally {
      setDeleteLoading(null)
    }
  }

  const handleReset = async () => {
    if (!confirm('Are you sure you want to delete ALL documents? This action cannot be undone.')) return

    setIsLoading(true)
    try {
      await apiClient.reset()
      setDocuments([])
      setFilteredDocuments([])
    } catch (error) {
      console.error('Failed to reset documents:', error)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadDocuments()
  }, [])

  useEffect(() => {
    if (!searchQuery.trim()) {
      setFilteredDocuments(documents)
    } else {
      const filtered = documents.filter(doc => 
        (doc.title || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        doc.content.toLowerCase().includes(searchQuery.toLowerCase())
      )
      setFilteredDocuments(filtered)
    }
  }, [searchQuery, documents])

  const formatDate = (dateString: string) => {
    try {
      const date = new Date(dateString)
      if (isNaN(date.getTime())) {
        return 'Invalid date'
      }
      return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })
    } catch (error) {
      return 'Invalid date'
    }
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Card
        title={
          <div style={{ textAlign: 'left' }}>
            <Space>
              <FileTextOutlined />
              <span>Document Management</span>
            </Space>
          </div>
        }
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column' }}
      >
        <Text type="secondary" style={{ marginBottom: 16, textAlign: 'left', display: 'block' }}>
          View, search, and manage your ingested documents.
        </Text>

        <Space direction="vertical" style={{ width: '100%', flex: 1 }} size="middle">
          <Space style={{ width: '100%' }}>
            <Input
              placeholder="Search documents..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              prefix={<SearchOutlined />}
              style={{ flex: 1, width: 300 }}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={loadDocuments}
              loading={isLoading}
            >
              Refresh
            </Button>
            <Button
              danger
              icon={<ExclamationCircleOutlined />}
              onClick={handleReset}
              disabled={isLoading || documents?.length === 0}
            >
              Reset All
            </Button>
          </Space>

          {isLoading ? (
            <div style={{ textAlign: 'center', padding: '64px 0' }}>
              <Spin size="large" />
              <div style={{ marginTop: 16 }}>
                <Text type="secondary">Loading documents...</Text>
              </div>
            </div>
          ) : filteredDocuments?.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                documents?.length === 0
                  ? 'No documents found. Start by ingesting some content.'
                  : 'No documents match your search.'
              }
            />
          ) : (
            <div style={{ flex: 1, overflowY: 'auto' }}>
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                <Text type="secondary">
                  Showing {filteredDocuments?.length || 0} of {documents?.length || 0} documents
                </Text>
                {filteredDocuments?.map((doc) => (
                  <Card key={doc.id} size="small" style={{ width: '100%' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div style={{ flex: 1, textAlign: 'left' }}>
                        <Title level={5} style={{ margin: 0, textAlign: 'left' }}>
                          {doc.title || `Document ${doc.id}`}
                        </Title>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          Created: {formatDate(doc.created)}
                        </Text>
                        <Paragraph
                          style={{ marginTop: 8, marginBottom: 8, textAlign: 'left' }}
                          ellipsis={{ rows: 3, expandable: true }}
                        >
                          {doc.content}
                        </Paragraph>
                        {doc.metadata && Object.keys(doc.metadata)?.length > 0 && (
                          <div style={{ textAlign: 'left' }}>
                            <Text strong style={{ fontSize: 11 }}>Metadata: </Text>
                            <Space wrap>
                              {Object.entries(doc.metadata)?.map(([key, value]) => (
                                <Tag key={key} color="blue" style={{ fontSize: 10 }}>
                                  {key}: {JSON.stringify(value)}
                                </Tag>
                              ))}
                            </Space>
                          </div>
                        )}
                      </div>
                      <Tooltip title="Delete document">
                        <Button
                          danger
                          size="small"
                          icon={<DeleteOutlined />}
                          onClick={() => handleDelete(doc.id)}
                          loading={deleteLoading === doc.id}
                        />
                      </Tooltip>
                    </div>
                  </Card>
                ))}
              </Space>
            </div>
          )}
        </Space>
      </Card>
    </div>
  )
}