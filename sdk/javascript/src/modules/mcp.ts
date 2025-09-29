import { RAGOClient } from '../client';
import {
  MCPTool,
  MCPServer,
  MCPToolCall,
  MCPToolResult,
  QueryRequest,
  QueryResponse,
  ChatRequest,
  ChatResponse,
} from '../types';

/**
 * MCP (Model Context Protocol) operations module
 */
export class MCPModule {
  constructor(private client: RAGOClient) {}

  /**
   * List available MCP tools
   */
  async listTools(): Promise<MCPTool[]> {
    const response = await this.client.get<MCPTool[]>('/api/mcp/tools');
    if (!response.success) {
      throw new Error(response.error || 'Failed to list MCP tools');
    }
    return response.data!;
  }

  /**
   * Get a specific MCP tool by name
   */
  async getTool(toolName: string): Promise<MCPTool> {
    const response = await this.client.get<MCPTool>(`/api/mcp/tools/${toolName}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get MCP tool');
    }
    return response.data!;
  }

  /**
   * Call an MCP tool
   */
  async callTool(toolName: string, args: Record<string, any>): Promise<MCPToolResult> {
    const request: MCPToolCall = {
      tool_name: toolName,
      arguments: args,
    };
    
    const response = await this.client.post<MCPToolResult>('/api/mcp/tools/call', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to call MCP tool');
    }
    return response.data!;
  }

  /**
   * Batch call multiple MCP tools
   */
  async batchCallTools(toolCalls: MCPToolCall[]): Promise<MCPToolResult[]> {
    const response = await this.client.post<MCPToolResult[]>('/api/mcp/tools/batch', { tools: toolCalls });
    if (!response.success) {
      throw new Error(response.error || 'Failed to batch call MCP tools');
    }
    return response.data!;
  }

  /**
   * Chat with MCP tools integration
   */
  async chatWithMCP(request: ChatRequest): Promise<ChatResponse> {
    const response = await this.client.post<ChatResponse>('/api/mcp/chat', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to chat with MCP');
    }
    return response.data!;
  }

  /**
   * Query with MCP tools integration
   */
  async queryWithMCP(request: QueryRequest): Promise<QueryResponse> {
    const response = await this.client.post<QueryResponse>('/api/mcp/query', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to query with MCP');
    }
    return response.data!;
  }

  /**
   * Get MCP server status
   */
  async getServerStatus(): Promise<Record<string, MCPServer>> {
    const response = await this.client.get<{ servers: Record<string, MCPServer> }>('/api/mcp/servers');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get MCP server status');
    }
    return response.data!.servers;
  }

  /**
   * Get tools by server
   */
  async getToolsByServer(serverName: string): Promise<MCPTool[]> {
    const response = await this.client.get<MCPTool[]>(`/api/mcp/servers/${serverName}/tools`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tools by server');
    }
    return response.data!;
  }

  /**
   * Start an MCP server
   */
  async startServer(serverName: string): Promise<void> {
    const response = await this.client.post('/api/mcp/servers/start', { server_name: serverName });
    if (!response.success) {
      throw new Error(response.error || 'Failed to start MCP server');
    }
  }

  /**
   * Stop an MCP server
   */
  async stopServer(serverName: string): Promise<void> {
    const response = await this.client.post('/api/mcp/servers/stop', { server_name: serverName });
    if (!response.success) {
      throw new Error(response.error || 'Failed to stop MCP server');
    }
  }

  /**
   * Get tools formatted for LLM integration
   */
  async getToolsForLLM(): Promise<any[]> {
    const response = await this.client.get<any[]>('/api/mcp/llm/tools');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tools for LLM');
    }
    return response.data!;
  }
}