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
      memoryNav: 'Memory',
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
      name: 'Name',
      description: 'Description',
      type: 'Type',
      tools: 'Tools',

      // Language
      language: 'Language',

      // Settings
      workingDirectory: 'Working Directory',
      homeDirectory: 'Home Directory',
      homeDirectoryPlaceholder: '~/.agentgo',
      homeDirectoryDesc: 'Base directory for all AgentGo data (default: ~/.agentgo)',
      mcpFilesystem: 'MCP Filesystem',
      allowedDirectories: 'Allowed Directories (one per line)',
      allowedDirectoriesPlaceholder: '/Users/username/projects\n/Users/username/documents',
      allowedDirectoriesDesc: 'Directories that MCP filesystem server can access',
      saveSettings: 'Save Settings',
      settingsSaved: 'Settings saved successfully!',

      // Documents
      documentList: 'Documents',
      viewDocument: 'View',
      deleteDocument: 'Delete',
      confirmDelete: 'Are you sure you want to delete "{name}"?',
      noDocuments: 'No documents yet',
      uploadDocument: 'Upload Document',
      selectFile: 'Select file to upload',
      content: 'Content',
      path: 'Path',
      created: 'Created',
      metadata: 'Metadata',
      fileName: 'File Name',

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
      documentsCount: 'Documents',
      skillsCount: 'Skills',
      memoriesCount: 'Memories',
      llmProviders: 'LLM Providers',
      embeddingProviders: 'Embedding Providers',
      mcpServers: 'MCP Servers',
      showMemory: 'Show Memory',
      showThinking: 'Show Thinking',

      // Chat
      sendMessage: 'Send',
      typeMessage: 'Type your message...',
      chatPlaceholder: 'Ask anything or use commands like /skill-name...',
      thinking: 'Thinking...',
      noMemory: 'No memory yet',

      // Skills
      skillList: 'Skills',
      noSkills: 'No skills available',
      addSkill: 'Add Skill',
      skillName: 'Skill Name',
      skillDescription: 'Description',
      skillPrompt: 'Prompt',
      deleteSkill: 'Delete',
      confirmDeleteSkill: 'Are you sure you want to delete this skill?',
      saveSkill: 'Save Skill',
      editSkill: 'Edit Skill',
      noDescription: 'No description',
      noPrompt: 'No prompt',

      // MCP
      mcpServersTitle: 'MCP Servers',
      noServers: 'No MCP servers configured',
      addServer: 'Add Server',
      serverName: 'Server Name',
      serverCommand: 'Command',
      serverArgs: 'Arguments (space separated)',
      serverType: 'Type',
      serverUrl: 'URL',
      stdio: 'stdio',
      http: 'HTTP',
      callTool: 'Call Tool',
      toolName: 'Tool Name',
      toolArgs: 'Arguments (JSON)',
      toolResult: 'Result',
      running2: 'Running',
      stopped2: 'Stopped',
      toolCount: '{count} tools',

      // Memory
      memories: 'Memories',
      noMemories: 'No memories yet',
      addMemory: 'Add Memory',
      searchMemories: 'Search Memories...',
      memoryContent: 'Content',
      memoryCreated: 'Created',
      deleteMemory: 'Delete',
      confirmDeleteMemory: 'Are you sure you want to delete this memory?',

      // Query
      testQuery: 'Test Query',
      queryPlaceholder: 'Enter your query...',
      search: 'Search',
      results: 'Results',
      noResults: 'No results found',
      score: 'Score',
      source: 'Source',

      // Agent
      runAgent: 'Run',
      stopAgent: 'Stop',
      agentGoal: 'Goal',
      agentGoalPlaceholder: 'Enter your goal...',
      agentRunning: 'Running...',
      agentIdle: 'Idle',
      maxTurns: 'Max Turns',
      temperature: 'Temperature',
      session: 'Session',
      newSession: 'New Session',
      thinking2: 'Thinking',
      action: 'Action',
      result: 'Result',
      sources: 'Sources',
      noSources: 'No sources',
    }
  },
  zh: {
    translation: {
      // Nav
      agent: '智能体',
      chat: '对话',
      skills: '技能',
      mcp: 'MCP',
      memoryNav: '记忆',
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
      name: '名称',
      description: '描述',
      type: '类型',
      tools: '工具',

      // Language
      language: '语言',

      // Settings
      workingDirectory: '工作目录',
      homeDirectory: '主目录',
      homeDirectoryPlaceholder: '~/.agentgo',
      homeDirectoryDesc: '所有 AgentGo 数据的基础目录（默认：~/.agentgo）',
      mcpFilesystem: 'MCP 文件系统',
      allowedDirectories: '允许的目录（每行一个）',
      allowedDirectoriesPlaceholder: '/Users/用户名/projects\n/Users/用户名/documents',
      allowedDirectoriesDesc: 'MCP 文件系统服务器可以访问的目录',
      saveSettings: '保存设置',
      settingsSaved: '设置保存成功！',

      // Documents
      documentList: '文档列表',
      viewDocument: '查看',
      deleteDocument: '删除',
      confirmDelete: '确定要删除 "{name}" 吗？',
      noDocuments: '暂无文档',
      uploadDocument: '上传文档',
      selectFile: '选择要上传的文件',
      content: '内容',
      path: '路径',
      created: '创建时间',
      metadata: '元数据',
      fileName: '文件名',

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
      documentsCount: '文档',
      skillsCount: '技能',
      memoriesCount: '记忆',
      llmProviders: 'LLM 提供商',
      embeddingProviders: 'Embedding 提供商',
      mcpServers: 'MCP 服务器',
      showMemory: '显示记忆',
      showThinking: '显示思考',

      // Chat
      sendMessage: '发送',
      typeMessage: '输入消息...',
      chatPlaceholder: '输入任何内容或使用命令如 /skill-name...',
      thinking: '思考中...',
      noMemory: '暂无记忆',

      // Skills
      skillList: '技能列表',
      noSkills: '暂无可用技能',
      addSkill: '添加技能',
      skillName: '技能名称',
      skillDescription: '描述',
      skillPrompt: '提示词',
      deleteSkill: '删除',
      confirmDeleteSkill: '确定要删除这个技能吗？',
      saveSkill: '保存技能',
      editSkill: '编辑技能',
      noDescription: '无描述',
      noPrompt: '无提示词',

      // MCP
      mcpServersTitle: 'MCP 服务器',
      noServers: '未配置 MCP 服务器',
      addServer: '添加服务器',
      serverName: '服务器名称',
      serverCommand: '命令',
      serverArgs: '参数（空格分隔）',
      serverType: '类型',
      serverUrl: 'URL',
      stdio: 'stdio',
      http: 'HTTP',
      callTool: '调用工具',
      toolName: '工具名称',
      toolArgs: '参数 (JSON)',
      toolResult: '结果',
      running2: '运行中',
      stopped2: '已停止',
      toolCount: '{count} 个工具',

      // Memory
      memories: '记忆列表',
      noMemories: '暂无记忆',
      addMemory: '添加记忆',
      searchMemories: '搜索记忆...',
      memoryContent: '内容',
      memoryCreated: '创建时间',
      deleteMemory: '删除',
      confirmDeleteMemory: '确定要删除这条记忆吗？',

      // Query
      testQuery: '测试查询',
      queryPlaceholder: '输入查询内容...',
      search: '搜索',
      results: '结果',
      noResults: '未找到结果',
      score: '相似度',
      source: '来源',

      // Agent
      runAgent: '运行',
      stopAgent: '停止',
      agentGoal: '目标',
      agentGoalPlaceholder: '输入你的目标...',
      agentRunning: '运行中...',
      agentIdle: '空闲',
      maxTurns: '最大轮次',
      temperature: '温度',
      session: '会话',
      newSession: '新会话',
      thinking2: '思考',
      action: '行动',
      result: '结果',
      sources: '来源',
      noSources: '无来源',
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
