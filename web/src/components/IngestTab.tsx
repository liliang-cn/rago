import { useState, useRef } from 'react'
import { Card, Button, Input, Tabs, Space, Typography, message } from 'antd'
import { FileTextOutlined, UploadOutlined, LoadingOutlined } from '@ant-design/icons'
import { apiClient } from '@/lib/api'

const { TextArea } = Input
const { Text } = Typography
const { TabPane } = Tabs

export function IngestTab() {
  const [textContent, setTextContent] = useState('')
  const [textTitle, setTextTitle] = useState('')
  const [textMetadata, setTextMetadata] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleTextSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!textContent.trim()) return

    setIsLoading(true)

    try {
      let metadata: Record<string, any> | undefined
      if (textMetadata.trim()) {
        try {
          metadata = JSON.parse(textMetadata)
        } catch (e) {
          message.error('Invalid JSON in metadata field')
          setIsLoading(false)
          return
        }
      }

      const response = await apiClient.ingestText(textContent, textTitle || undefined, metadata)
      if (response.error) {
        message.error(response.error)
      } else {
        message.success('Text ingested successfully!')
        setTextContent('')
        setTextTitle('')
        setTextMetadata('')
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unknown error')
    } finally {
      setIsLoading(false)
    }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsLoading(true)

    try {
      let metadata: Record<string, any> | undefined
      if (textMetadata.trim()) {
        try {
          metadata = JSON.parse(textMetadata)
        } catch (e) {
          message.error('Invalid JSON in metadata field')
          setIsLoading(false)
          return
        }
      }

      const response = await apiClient.ingestFile(file, metadata)
      if (response.error) {
        message.error(response.error)
      } else {
        message.success(`File "${file.name}" ingested successfully!`)
        setTextMetadata('')
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unknown error')
    } finally {
      setIsLoading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Card
        title={
          <div style={{ textAlign: 'left' }}>
            <Space>
              <FileTextOutlined />
              <span>Ingest Content</span>
            </Space>
          </div>
        }
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column' }}
      >
        <Text type="secondary" style={{ marginBottom: 16, textAlign: 'left', display: 'block' }}>
          Add documents to your knowledge base by uploading files or entering text directly.
        </Text>

        <Tabs defaultActiveKey="text" style={{ flex: 1 }}>
          <TabPane tab="Text Input" key="text">
            <form onSubmit={handleTextSubmit}>
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                <div>
                  <Text strong style={{ textAlign: 'left', display: 'block' }}>Title (optional)</Text>
                  <Input
                    value={textTitle}
                    onChange={(e) => setTextTitle(e.target.value)}
                    placeholder="Document title..."
                    disabled={isLoading}
                    style={{ marginTop: 8 }}
                  />
                </div>
                <div>
                  <Text strong style={{ textAlign: 'left', display: 'block' }}>Content *</Text>
                  <TextArea
                    value={textContent}
                    onChange={(e) => setTextContent(e.target.value)}
                    placeholder="Enter your text content here..."
                    rows={8}
                    disabled={isLoading}
                    style={{ marginTop: 8 }}
                  />
                </div>
                <div>
                  <Text strong style={{ textAlign: 'left', display: 'block' }}>Metadata (optional, JSON format)</Text>
                  <TextArea
                    value={textMetadata}
                    onChange={(e) => setTextMetadata(e.target.value)}
                    placeholder='{"author": "John Doe", "category": "technical", "tags": ["ai", "rag"]}'
                    rows={3}
                    disabled={isLoading}
                    style={{ marginTop: 8 }}
                  />
                  <Text type="secondary" style={{ fontSize: 12, textAlign: 'left', display: 'block' }}>
                    Enter valid JSON for document metadata (e.g., author, category, tags)
                  </Text>
                </div>
                <Button 
                  type="primary" 
                  htmlType="submit" 
                  disabled={isLoading || !textContent.trim()}
                  loading={isLoading}
                  icon={isLoading ? <LoadingOutlined /> : <FileTextOutlined />}
                >
                  Ingest Text
                </Button>
              </Space>
            </form>
          </TabPane>
          
          <TabPane tab="File Upload" key="file">
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <div>
                <Text strong>Metadata (optional, JSON format)</Text>
                <TextArea
                  value={textMetadata}
                  onChange={(e) => setTextMetadata(e.target.value)}
                  placeholder='{"author": "John Doe", "category": "technical", "tags": ["ai", "rag"]}'
                  rows={3}
                  disabled={isLoading}
                  style={{ marginTop: 8 }}
                />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  Enter valid JSON for document metadata (e.g., author, category, tags)
                </Text>
              </div>
              
              <div style={{ 
                border: '2px dashed #d9d9d9', 
                borderRadius: 8, 
                padding: 48, 
                textAlign: 'center',
                backgroundColor: '#fafafa'
              }}>
                <UploadOutlined style={{ fontSize: 48, color: '#bfbfbf', marginBottom: 16 }} />
                <div>
                  <Text strong style={{ textAlign: 'left', display: 'block' }}>Upload a file</Text>
                  <br />
                  <Text type="secondary" style={{ fontSize: 12, textAlign: 'left', display: 'block' }}>
                    Supports: PDF, TXT, DOC, DOCX, MD and more
                  </Text>
                </div>
                <input
                  ref={fileInputRef}
                  type="file"
                  onChange={handleFileUpload}
                  style={{ display: 'none' }}
                  accept=".pdf,.txt,.doc,.docx,.md,.html,.rtf"
                  disabled={isLoading}
                />
                <br />
                <Button
                  type="default"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isLoading}
                  loading={isLoading}
                  icon={isLoading ? <LoadingOutlined /> : <UploadOutlined />}
                  style={{ marginTop: 16 }}
                >
                  Choose File
                </Button>
              </div>
            </Space>
          </TabPane>
        </Tabs>
      </Card>
    </div>
  )
}