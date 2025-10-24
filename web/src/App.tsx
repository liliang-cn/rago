import React, { useState, useEffect } from 'react';
import { Layout, Menu, theme, ConfigProvider, Typography } from 'antd';
import { BrowserRouter as Router, Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import {
  MessageOutlined,
  FileTextOutlined,
  DatabaseOutlined,
  SearchOutlined,
  RobotOutlined,
  ApiOutlined,
  MonitorOutlined,
  BarChartOutlined,
  HistoryOutlined,
  EyeOutlined,
  ToolOutlined,
  GithubOutlined,
  AppstoreOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { ChatTab } from './components/ChatTab';
import { IngestTab } from './components/IngestTab';
import { DocumentsTab } from './components/DocumentsTab';
import { SearchTab } from './components/SearchTab';
import { LLMTab } from './components/LLMTab';
import { MCPTab } from './components/MCPTab';
import { StatusTab } from './components/StatusTab';
import { TokenAnalysisTab } from './components/TokenAnalysisTab';
import { ConversationHistoryTab } from './components/ConversationHistoryTab';
import { RAGVisualizationTab } from './components/RAGVisualizationTab';
import { ToolCallsTab } from './components/ToolCallsTab';
import './App.css';

const { Header, Sider, Content } = Layout;
const { Title, Text } = Typography;

type MenuItem = Required<MenuProps>['items'][number];

function getItem(
  label: React.ReactNode,
  key: React.Key,
  icon?: React.ReactNode,
  children?: MenuItem[],
): MenuItem {
  return {
    key,
    icon,
    children,
    label,
  } as MenuItem;
}

const menuItems: MenuItem[] = [
  getItem('Chat with Documents', 'chat-with-docs', <MessageOutlined />),
  getItem('Ingest', 'ingest', <FileTextOutlined />),
  getItem('Documents', 'documents', <DatabaseOutlined />),
  getItem('Search', 'search', <SearchOutlined />),
  getItem('LLM Settings', 'llm', <RobotOutlined />),
  getItem('MCP Tools', 'mcp', <ApiOutlined />),
  getItem('System Status', 'status', <MonitorOutlined />),
  getItem('Token Analysis', 'tokens', <BarChartOutlined />),
  getItem('History', 'history', <HistoryOutlined />),
  getItem('RAG Visualization', 'rag', <EyeOutlined />),
  getItem('Tool Calls', 'tools', <ToolOutlined />),
];

function AppContent() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // Get current route from URL
  const getCurrentKey = () => {
    const path = location.pathname;
    if (path === '/' || path === '/chat-with-docs') return 'chat-with-docs';
    if (path.startsWith('/')) return path.slice(1);
    return 'chat-with-docs';
  };

  const [selectedKey, setSelectedKey] = useState(getCurrentKey());

  useEffect(() => {
    setSelectedKey(getCurrentKey());
  }, [location.pathname]);


  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#1890ff',
          borderRadius: 6,
        },
        algorithm: theme.defaultAlgorithm,
      }}
    >
      <Layout style={{ minHeight: '100vh' }}>
        <Sider 
          trigger={null} 
          collapsible 
          collapsed={collapsed}
          width={250}
          style={{
            background: colorBgContainer,
            borderRight: '1px solid #f0f0f0'
          }}
        >
          <div style={{ 
            height: 64, 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            borderBottom: '1px solid #f0f0f0'
          }}>
            <AppstoreOutlined style={{ fontSize: 28, color: '#1890ff' }} />
            {!collapsed && (
              <Title level={4} style={{ margin: '0 0 0 12px', color: '#000', textAlign: 'left' }}>RAGO v2</Title>
            )}
          </div>
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            items={menuItems}
            onClick={(e) => {
              setSelectedKey(e.key);
              navigate(`/${e.key === 'chat-with-docs' ? '' : e.key}`);
            }}
            style={{ borderRight: 0, marginTop: 16, textAlign: 'left' }}
          />
        </Sider>
        <Layout>
          <Header
            style={{
              padding: 0,
              background: colorBgContainer,
              borderBottom: '1px solid #f0f0f0',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center' }}>
              {React.createElement(collapsed ? MenuUnfoldOutlined : MenuFoldOutlined, {
                className: 'trigger',
                onClick: () => setCollapsed(!collapsed),
                style: {
                  fontSize: 18,
                  padding: '0 24px',
                  cursor: 'pointer',
                  transition: 'color 0.3s',
                },
              })}
              <Text style={{ fontSize: 14, color: '#666' }}>
                Advanced RAG with MCP Integration
              </Text>
            </div>
            <a
              href="https://github.com/liliang-cn/rago"
              target="_blank"
              rel="noopener noreferrer"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                marginRight: 24,
                color: '#666',
                textDecoration: 'none',
              }}
            >
              <GithubOutlined style={{ fontSize: 20 }} />
              <span>GitHub</span>
            </a>
          </Header>
          <Content
            style={{
              margin: '24px',
              padding: 24,
              minHeight: 280,
              background: colorBgContainer,
              borderRadius: borderRadiusLG,
            }}
          >
            <Routes>
              <Route path="/" element={<ChatTab />} />
              <Route path="/chat-with-docs" element={<ChatTab />} />
              <Route path="/ingest" element={<IngestTab />} />
              <Route path="/documents" element={<DocumentsTab />} />
              <Route path="/search" element={<SearchTab />} />
              <Route path="/llm" element={<LLMTab />} />
              <Route path="/mcp" element={<MCPTab />} />
              <Route path="/status" element={<StatusTab />} />
              <Route path="/tokens" element={<TokenAnalysisTab />} />
              <Route path="/history" element={<ConversationHistoryTab />} />
              <Route path="/rag" element={<RAGVisualizationTab />} />
              <Route path="/tools" element={<ToolCallsTab />} />
              <Route path="*" element={<ChatTab />} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
}

function App() {
  return (
    <Router>
      <AppContent />
    </Router>
  );
}

export default App;