import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

const resources = {
  en: {
    translation: {
      // Nav
      agent: 'Agent',
      chat: 'Chat',
      skills: 'Skills',
      mcp: 'MCP',
      memory: 'Memory',
      status: 'Status',
      query: 'Query',
      documents: 'Documents',
      settings: 'Settings',

      // Common
      save: 'Save',
      cancel: 'Cancel',
      delete: 'Delete',
      edit: 'Edit',
      add: 'Add',
      close: 'Close',
      loading: 'Loading...',
      error: 'Error',
      success: 'Success',

      // Settings
      workingDirectory: 'Working Directory',
      homeDirectory: 'Home Directory',
      homeDirectoryPlaceholder: '~/.agentgo',
      homeDirectoryDesc: 'Base directory for all AgentGo data',
      mcpFilesystem: 'MCP Filesystem',
      allowedDirectories: 'Allowed Directories',
      allowedDirectoriesPlaceholder: '/Users/username/projects',
      allowedDirectoriesDesc: 'Directories that MCP filesystem server can access',
      saveSettings: 'Save Settings',
      settingsSaved: 'Settings saved successfully!',

      // Documents
      documentList: 'Documents',
      viewDocument: 'View',
      deleteDocument: 'Delete',
      confirmDelete: 'Are you sure you want to delete this document?',
      noDocuments: 'No documents yet',
      uploadDocument: 'Upload Document',
      selectFile: 'Select file to upload',
      content: 'Content',
      path: 'Path',
      created: 'Created',
      metadata: 'Metadata',

      // Status
      systemStatus: 'System Status',
      running: 'Running',
      stopped: 'Stopped',
      version: 'Version',
      enabled: 'Enabled',
      disabled: 'Disabled',
      providers: 'Providers',
      rag: 'RAG',
      chunks: 'Chunks',
      documents_count: 'Documents',

      // Chat
      sendMessage: 'Send',
      typeMessage: 'Type your message...',

      // Skills
      skillList: 'Skills',
      noSkills: 'No skills available',

      // MCP
      mcpServers: 'MCP Servers',
      noServers: 'No MCP servers configured',
      addServer: 'Add Server',
      serverName: 'Server Name',
      serverCommand: 'Command',
      serverArgs: 'Arguments',
      serverType: 'Type',
      serverUrl: 'URL',
      tools: 'Tools',
      callTool: 'Call Tool',
      toolName: 'Tool Name',
      toolArgs: 'Arguments (JSON)',

      // Memory
      memories: 'Memories',
      noMemories: 'No memories yet',
      addMemory: 'Add Memory',
      searchMemories: 'Search Memories...',

      // Query
      testQuery: 'Test Query',
      queryPlaceholder: 'Enter your query...',
      search: 'Search',
      results: 'Results',
      noResults: 'No results found',
    }
  },
  zh: {
    translation: {
      // Nav
      agent: '智能体',
      chat: '对话',
      skills: '技能',
      mcp: 'MCP',
      memory: '记忆',
      status: '状态',
      query: '查询',
      documents: '文档',
      settings: '设置',

      // Common
      save: '保存',
      cancel: '取消',
      delete: '删除',
      edit: '编辑',
      add: '添加',
      close: '关闭',
      loading: '加载中...',
      error: '错误',
      success: '成功',

      // Settings
      workingDirectory: '工作目录',
      homeDirectory: '主目录',
      homeDirectoryPlaceholder: '~/.agentgo',
      homeDirectoryDesc: '所有 AgentGo 数据的基础目录',
      mcpFilesystem: 'MCP 文件系统',
      allowedDirectories: '允许的目录',
      allowedDirectoriesPlaceholder: '/Users/用户名/projects',
      allowedDirectoriesDesc: 'MCP 文件系统服务器可以访问的目录',
      saveSettings: '保存设置',
      settingsSaved: '设置保存成功！',

      // Documents
      documentList: '文档列表',
      viewDocument: '查看',
      deleteDocument: '删除',
      confirmDelete: '确定要删除这个文档吗？',
      noDocuments: '暂无文档',
      uploadDocument: '上传文档',
      selectFile: '选择要上传的文件',
      content: '内容',
      path: '路径',
      created: '创建时间',
      metadata: '元数据',

      // Status
      systemStatus: '系统状态',
      running: '运行中',
      stopped: '已停止',
      version: '版本',
      enabled: '已启用',
      disabled: '已禁用',
      providers: '提供商',
      rag: 'RAG',
      chunks: '分块',
      documents_count: '文档',

      // Chat
      sendMessage: '发送',
      typeMessage: '输入消息...',

      // Skills
      skillList: '技能列表',
      noSkills: '暂无可用技能',

      // MCP
      mcpServers: 'MCP 服务器',
      noServers: '未配置 MCP 服务器',
      addServer: '添加服务器',
      serverName: '服务器名称',
      serverCommand: '命令',
      serverArgs: '参数',
      serverType: '类型',
      serverUrl: 'URL',
      tools: '工具',
      callTool: '调用工具',
      toolName: '工具名称',
      toolArgs: '参数 (JSON)',

      // Memory
      memories: '记忆',
      noMemories: '暂无记忆',
      addMemory: '添加记忆',
      searchMemories: '搜索记忆...',

      // Query
      testQuery: '测试查询',
      queryPlaceholder: '输入查询内容...',
      search: '搜索',
      results: '结果',
      noResults: '未找到结果',
    }
  }
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'zh',
    defaultNS: 'translation',
    ns: ['translation'],
    interpolation: {
      escapeValue: false
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage']
    }
  })

export default i18n
