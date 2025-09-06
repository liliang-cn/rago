// Core RAGO Types
export interface Provider {
  id: string;
  name: string;
  type: 'llm' | 'embedder' | 'both';
  endpoint: string;
  status: 'active' | 'inactive' | 'error';
  models: string[];
  cost?: {
    input: number;
    output: number;
    unit: string;
  };
  performance?: {
    latency: number;
    throughput: number;
    reliability: number;
  };
  usage?: {
    requests: number;
    tokens: number;
    cost: number;
  };
}

export interface Document {
  id: string;
  name: string;
  type: string;
  size: number;
  uploadedAt: string;
  status: 'processing' | 'ready' | 'error';
  chunks: number;
  metadata?: Record<string, any>;
  embedding?: {
    model: string;
    dimensions: number;
    tokens: number;
  };
}

export interface QueryResult {
  id: string;
  query: string;
  response: string;
  sources: DocumentChunk[];
  timestamp: string;
  model: string;
  processingTime: number;
  relevanceScore: number;
}

export interface DocumentChunk {
  id: string;
  documentId: string;
  content: string;
  metadata: Record<string, any>;
  score: number;
  startOffset: number;
  endOffset: number;
}

// Monitoring Types
export interface MetricPoint {
  timestamp: string;
  value: number;
  label?: string;
}

export interface SystemMetrics {
  cpu: MetricPoint[];
  memory: MetricPoint[];
  disk: MetricPoint[];
  network: MetricPoint[];
}

export interface ComponentStatus {
  name: string;
  status: 'healthy' | 'warning' | 'error';
  uptime: number;
  lastCheck: string;
  message?: string;
  details?: Record<string, any>;
}

export interface Alert {
  id: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  title: string;
  message: string;
  timestamp: string;
  source: string;
  acknowledged: boolean;
  resolved: boolean;
}

// Workflow Types
export interface WorkflowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: {
    label: string;
    config: Record<string, any>;
    inputs?: WorkflowPort[];
    outputs?: WorkflowPort[];
  };
}

export interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

export interface WorkflowPort {
  id: string;
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
  required: boolean;
  description?: string;
}

export interface Workflow {
  id: string;
  name: string;
  description: string;
  version: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  variables: Record<string, any>;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  status: 'draft' | 'active' | 'archived';
}

export interface WorkflowExecution {
  id: string;
  workflowId: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  startTime: string;
  endTime?: string;
  inputs: Record<string, any>;
  outputs?: Record<string, any>;
  logs: ExecutionLog[];
  metrics: {
    duration: number;
    nodesExecuted: number;
    totalNodes: number;
  };
}

export interface ExecutionLog {
  timestamp: string;
  level: 'info' | 'warn' | 'error' | 'debug';
  nodeId?: string;
  message: string;
  details?: any;
}

// Agent Marketplace Types
export interface AgentTemplate {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  category: string;
  tags: string[];
  icon?: string;
  screenshots: string[];
  readme: string;
  workflow: Workflow;
  downloads: number;
  rating: number;
  reviews: number;
  createdAt: string;
  updatedAt: string;
  verified: boolean;
  dependencies: string[];
  license: string;
}

export interface AgentReview {
  id: string;
  templateId: string;
  author: string;
  rating: number;
  comment: string;
  timestamp: string;
  helpful: number;
}

// Job Scheduler Types
export interface Job {
  id: string;
  name: string;
  description: string;
  workflowId: string;
  schedule: {
    type: 'cron' | 'interval' | 'once';
    expression: string;
    timezone?: string;
  };
  enabled: boolean;
  inputs: Record<string, any>;
  webhook?: {
    url: string;
    method: 'GET' | 'POST' | 'PUT';
    headers: Record<string, string>;
    body?: string;
  };
  retryPolicy: {
    maxRetries: number;
    backoff: 'fixed' | 'exponential' | 'linear';
    delay: number;
  };
  notifications: {
    onSuccess: boolean;
    onFailure: boolean;
    channels: string[];
  };
  tags: string[];
  createdAt: string;
  updatedAt: string;
  lastRun?: string;
  nextRun?: string;
}

export interface JobExecution {
  id: string;
  jobId: string;
  workflowExecutionId: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  startTime: string;
  endTime?: string;
  duration?: number;
  retryAttempt: number;
  output?: any;
  error?: string;
  logs: ExecutionLog[];
}

// MCP Tool Types
export interface MCPTool {
  name: string;
  description: string;
  inputSchema: {
    type: string;
    properties: Record<string, any>;
    required?: string[];
  };
}

export interface MCPServer {
  name: string;
  command: string[];
  args?: string[];
  env?: Record<string, string>;
  status: 'running' | 'stopped' | 'error';
  tools: MCPTool[];
  lastPing?: string;
}

// UI State Types
export interface UIState {
  theme: 'light' | 'dark' | 'system';
  sidebarCollapsed: boolean;
  notifications: Notification[];
  activeView: string;
}

export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message: string;
  timestamp: string;
  read: boolean;
  action?: {
    label: string;
    onClick: () => void;
  };
}

// API Response Types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
  meta?: {
    total?: number;
    page?: number;
    limit?: number;
  };
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  meta: {
    total: number;
    page: number;
    limit: number;
    pages: number;
  };
}

// WebSocket Message Types
export interface WebSocketMessage {
  type: string;
  payload: any;
  timestamp: string;
}

export interface MetricsUpdate extends WebSocketMessage {
  type: 'metrics_update';
  payload: {
    component: string;
    metrics: MetricPoint[];
  };
}

export interface StatusUpdate extends WebSocketMessage {
  type: 'status_update';
  payload: {
    component: string;
    status: ComponentStatus;
  };
}

export interface WorkflowUpdate extends WebSocketMessage {
  type: 'workflow_update';
  payload: {
    executionId: string;
    status: WorkflowExecution['status'];
    logs?: ExecutionLog[];
  };
}

// Configuration Types
export interface Config {
  server: {
    port: number;
    host: string;
    cors: boolean;
  };
  database: {
    path: string;
    maxConnections: number;
  };
  providers: Provider[];
  features: {
    mcpEnabled: boolean;
    agentsEnabled: boolean;
    schedulerEnabled: boolean;
    monitoringEnabled: boolean;
  };
  ui: {
    title: string;
    branding: {
      logo?: string;
      favicon?: string;
      primaryColor?: string;
    };
  };
}