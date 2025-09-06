import { 
  ApiResponse, 
  PaginatedResponse,
  Provider,
  Document,
  QueryResult,
  SystemMetrics,
  ComponentStatus,
  Alert,
  Workflow,
  WorkflowExecution,
  Job,
  JobExecution,
  AgentTemplate,
  MCPServer,
  Config
} from '@/types';

// Base API configuration
const API_BASE = '/api';
const WS_BASE = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;

// HTTP client with error handling
class APIClient {
  private baseURL: string;

  constructor(baseURL: string = API_BASE) {
    this.baseURL = baseURL;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;
    
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || `HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  }

  async get<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  async post<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }

  async upload<T>(endpoint: string, file: File, additionalData?: Record<string, any>): Promise<ApiResponse<T>> {
    const formData = new FormData();
    formData.append('file', file);
    
    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, JSON.stringify(value));
      });
    }

    return this.request<T>(endpoint, {
      method: 'POST',
      body: formData,
      headers: {}, // Don't set Content-Type for FormData
    });
  }
}

const apiClient = new APIClient();

// System APIs
export const systemAPI = {
  // Health check
  health: () => apiClient.get<{ status: string; uptime: number }>('/health'),
  
  // Configuration
  getConfig: () => apiClient.get<Config>('/config'),
  updateConfig: (config: Partial<Config>) => apiClient.put<Config>('/config', config),
  
  // System metrics
  getMetrics: () => apiClient.get<SystemMetrics>('/metrics'),
  getComponentStatuses: () => apiClient.get<ComponentStatus[]>('/status'),
  
  // Alerts
  getAlerts: (params?: { limit?: number; severity?: string }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<Alert[]>(`/alerts${query ? `?${query}` : ''}`);
  },
  acknowledgeAlert: (id: string) => apiClient.put<Alert>(`/alerts/${id}/acknowledge`),
  resolveAlert: (id: string) => apiClient.put<Alert>(`/alerts/${id}/resolve`),
};

// Provider APIs
export const providerAPI = {
  // List providers
  getProviders: () => apiClient.get<Provider[]>('/providers'),
  
  // Add provider
  addProvider: (provider: Omit<Provider, 'id' | 'status'>) => 
    apiClient.post<Provider>('/providers', provider),
  
  // Update provider
  updateProvider: (id: string, updates: Partial<Provider>) =>
    apiClient.put<Provider>(`/providers/${id}`, updates),
  
  // Remove provider
  removeProvider: (id: string) => apiClient.delete<void>(`/providers/${id}`),
  
  // Test provider connection
  testProvider: (id: string) => apiClient.post<{ status: string; latency: number }>(`/providers/${id}/test`),
  
  // Get provider metrics
  getProviderMetrics: (id: string, timeRange?: string) =>
    apiClient.get<any>(`/providers/${id}/metrics${timeRange ? `?range=${timeRange}` : ''}`),
};

// Document APIs
export const documentAPI = {
  // List documents
  getDocuments: (params?: { page?: number; limit?: number; type?: string }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<PaginatedResponse<Document>>(`/documents${query ? `?${query}` : ''}`);
  },
  
  // Upload document
  uploadDocument: (file: File, metadata?: Record<string, any>) =>
    apiClient.upload<Document>('/documents/upload', file, metadata),
  
  // Get document details
  getDocument: (id: string) => apiClient.get<Document>(`/documents/${id}`),
  
  // Update document
  updateDocument: (id: string, updates: Partial<Document>) =>
    apiClient.put<Document>(`/documents/${id}`, updates),
  
  // Delete document
  deleteDocument: (id: string) => apiClient.delete<void>(`/documents/${id}`),
  
  // Reprocess document
  reprocessDocument: (id: string) => apiClient.post<Document>(`/documents/${id}/reprocess`),
  
  // Query documents
  query: (query: string, options?: {
    limit?: number;
    threshold?: number;
    model?: string;
    showSources?: boolean;
  }) => apiClient.post<QueryResult>('/query', { query, ...options }),
  
  // Get query history
  getQueryHistory: (limit?: number) =>
    apiClient.get<QueryResult[]>(`/query/history${limit ? `?limit=${limit}` : ''}`),
};

// Workflow APIs
export const workflowAPI = {
  // List workflows
  getWorkflows: (params?: { page?: number; limit?: number; status?: string }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<PaginatedResponse<Workflow>>(`/workflows${query ? `?${query}` : ''}`);
  },
  
  // Create workflow
  createWorkflow: (workflow: Omit<Workflow, 'id' | 'createdAt' | 'updatedAt'>) =>
    apiClient.post<Workflow>('/workflows', workflow),
  
  // Get workflow
  getWorkflow: (id: string) => apiClient.get<Workflow>(`/workflows/${id}`),
  
  // Update workflow
  updateWorkflow: (id: string, updates: Partial<Workflow>) =>
    apiClient.put<Workflow>(`/workflows/${id}`, updates),
  
  // Delete workflow
  deleteWorkflow: (id: string) => apiClient.delete<void>(`/workflows/${id}`),
  
  // Execute workflow
  executeWorkflow: (id: string, inputs?: Record<string, any>) =>
    apiClient.post<WorkflowExecution>(`/workflows/${id}/execute`, { inputs }),
  
  // List executions
  getExecutions: (workflowId?: string, params?: { page?: number; limit?: number; status?: string }) => {
    const query = new URLSearchParams(params as any).toString();
    const endpoint = workflowId 
      ? `/workflows/${workflowId}/executions`
      : '/executions';
    return apiClient.get<PaginatedResponse<WorkflowExecution>>(`${endpoint}${query ? `?${query}` : ''}`);
  },
  
  // Get execution details
  getExecution: (id: string) => apiClient.get<WorkflowExecution>(`/executions/${id}`),
  
  // Cancel execution
  cancelExecution: (id: string) => apiClient.post<WorkflowExecution>(`/executions/${id}/cancel`),
  
  // Get execution logs
  getExecutionLogs: (id: string) => apiClient.get<any>(`/executions/${id}/logs`),
};

// Job Scheduler APIs
export const jobAPI = {
  // List jobs
  getJobs: (params?: { page?: number; limit?: number; enabled?: boolean }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<PaginatedResponse<Job>>(`/jobs${query ? `?${query}` : ''}`);
  },
  
  // Create job
  createJob: (job: Omit<Job, 'id' | 'createdAt' | 'updatedAt'>) =>
    apiClient.post<Job>('/jobs', job),
  
  // Get job
  getJob: (id: string) => apiClient.get<Job>(`/jobs/${id}`),
  
  // Update job
  updateJob: (id: string, updates: Partial<Job>) =>
    apiClient.put<Job>(`/jobs/${id}`, updates),
  
  // Delete job
  deleteJob: (id: string) => apiClient.delete<void>(`/jobs/${id}`),
  
  // Enable/disable job
  toggleJob: (id: string, enabled: boolean) =>
    apiClient.put<Job>(`/jobs/${id}`, { enabled }),
  
  // Trigger job manually
  triggerJob: (id: string) => apiClient.post<JobExecution>(`/jobs/${id}/trigger`),
  
  // Get job executions
  getJobExecutions: (jobId: string, params?: { page?: number; limit?: number }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<PaginatedResponse<JobExecution>>(`/jobs/${jobId}/executions${query ? `?${query}` : ''}`);
  },
  
  // Get execution details
  getExecution: (id: string) => apiClient.get<JobExecution>(`/job-executions/${id}`),
};

// Agent Marketplace APIs
export const marketplaceAPI = {
  // List templates
  getTemplates: (params?: { 
    page?: number; 
    limit?: number; 
    category?: string; 
    search?: string;
    sort?: 'name' | 'downloads' | 'rating' | 'updated';
    order?: 'asc' | 'desc';
  }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<PaginatedResponse<AgentTemplate>>(`/marketplace/templates${query ? `?${query}` : ''}`);
  },
  
  // Get template details
  getTemplate: (id: string) => apiClient.get<AgentTemplate>(`/marketplace/templates/${id}`),
  
  // Install template
  installTemplate: (id: string) => apiClient.post<Workflow>(`/marketplace/templates/${id}/install`),
  
  // Publish template
  publishTemplate: (template: Omit<AgentTemplate, 'id' | 'createdAt' | 'updatedAt' | 'downloads' | 'rating'>) =>
    apiClient.post<AgentTemplate>('/marketplace/templates', template),
  
  // Update template
  updateTemplate: (id: string, updates: Partial<AgentTemplate>) =>
    apiClient.put<AgentTemplate>(`/marketplace/templates/${id}`, updates),
  
  // Delete template
  deleteTemplate: (id: string) => apiClient.delete<void>(`/marketplace/templates/${id}`),
  
  // Get template reviews
  getReviews: (templateId: string, params?: { page?: number; limit?: number }) => {
    const query = new URLSearchParams(params as any).toString();
    return apiClient.get<PaginatedResponse<any>>(`/marketplace/templates/${templateId}/reviews${query ? `?${query}` : ''}`);
  },
  
  // Add review
  addReview: (templateId: string, review: { rating: number; comment: string }) =>
    apiClient.post<any>(`/marketplace/templates/${templateId}/reviews`, review),
  
  // Get categories
  getCategories: () => apiClient.get<string[]>('/marketplace/categories'),
};

// MCP APIs
export const mcpAPI = {
  // List MCP servers
  getServers: () => apiClient.get<MCPServer[]>('/mcp/servers'),
  
  // Get server details
  getServer: (name: string) => apiClient.get<MCPServer>(`/mcp/servers/${name}`),
  
  // Start server
  startServer: (name: string) => apiClient.post<MCPServer>(`/mcp/servers/${name}/start`),
  
  // Stop server
  stopServer: (name: string) => apiClient.post<MCPServer>(`/mcp/servers/${name}/stop`),
  
  // Restart server
  restartServer: (name: string) => apiClient.post<MCPServer>(`/mcp/servers/${name}/restart`),
  
  // Get server tools
  getServerTools: (name: string) => apiClient.get<any[]>(`/mcp/servers/${name}/tools`),
  
  // Call tool
  callTool: (serverName: string, toolName: string, args: any) =>
    apiClient.post<any>(`/mcp/servers/${serverName}/tools/${toolName}`, args),
  
  // Add server
  addServer: (server: Omit<MCPServer, 'status' | 'tools' | 'lastPing'>) =>
    apiClient.post<MCPServer>('/mcp/servers', server),
  
  // Remove server
  removeServer: (name: string) => apiClient.delete<void>(`/mcp/servers/${name}`),
};

// WebSocket client for real-time updates
export class WebSocketClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private listeners: Map<string, Set<(data: any) => void>> = new Map();

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      this.ws = new WebSocket(WS_BASE);
      
      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
      };
      
      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
      
      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.reconnect();
      };
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect WebSocket:', error);
      this.reconnect();
    }
  }

  private reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    setTimeout(() => {
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      this.connect();
    }, delay);
  }

  private handleMessage(message: any) {
    const { type, payload } = message;
    const listeners = this.listeners.get(type);
    
    if (listeners) {
      listeners.forEach(callback => callback(payload));
    }
  }

  subscribe(eventType: string, callback: (data: any) => void) {
    if (!this.listeners.has(eventType)) {
      this.listeners.set(eventType, new Set());
    }
    
    this.listeners.get(eventType)!.add(callback);
    
    // Return unsubscribe function
    return () => {
      const listeners = this.listeners.get(eventType);
      if (listeners) {
        listeners.delete(callback);
        if (listeners.size === 0) {
          this.listeners.delete(eventType);
        }
      }
    };
  }

  send(message: any) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.error('WebSocket is not connected');
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.listeners.clear();
  }
}

export const wsClient = new WebSocketClient();