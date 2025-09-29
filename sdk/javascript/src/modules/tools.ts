import { RAGOClient } from '../client';
import {
  Tool,
  ToolExecution,
} from '../types';

/**
 * Tools operations module
 * Handles built-in tool management and execution
 */
export class ToolsModule {
  constructor(private client: RAGOClient) {}

  /**
   * List all available tools
   */
  async list(): Promise<Tool[]> {
    const response = await this.client.get<Tool[]>('/api/tools');
    if (!response.success) {
      throw new Error(response.error || 'Failed to list tools');
    }
    return response.data!;
  }

  /**
   * Get a specific tool by name
   */
  async get(toolName: string): Promise<Tool> {
    const response = await this.client.get<Tool>(`/api/tools/${toolName}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tool');
    }
    return response.data!;
  }

  /**
   * Execute a tool
   */
  async execute(toolName: string, args: Record<string, any>): Promise<any> {
    const response = await this.client.post<any>(`/api/tools/${toolName}/execute`, args);
    if (!response.success) {
      throw new Error(response.error || 'Failed to execute tool');
    }
    return response.data!;
  }

  /**
   * Get tool execution statistics
   */
  async getStats(): Promise<{
    total_executions: number;
    success_rate: number;
    avg_execution_time: number;
    most_used_tools: Array<{ tool: string; count: number }>;
  }> {
    const response = await this.client.get('/api/tools/stats');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tool stats');
    }
    return response.data!;
  }

  /**
   * Get tool registry statistics
   */
  async getRegistryStats(): Promise<{
    total_tools: number;
    enabled_tools: number;
    disabled_tools: number;
    tools_by_category: Record<string, number>;
  }> {
    const response = await this.client.get('/api/tools/registry/stats');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get registry stats');
    }
    return response.data!;
  }

  /**
   * List tool executions
   */
  async listExecutions(): Promise<ToolExecution[]> {
    const response = await this.client.get<ToolExecution[]>('/api/tools/executions');
    if (!response.success) {
      throw new Error(response.error || 'Failed to list tool executions');
    }
    return response.data!;
  }

  /**
   * Get a specific tool execution
   */
  async getExecution(executionId: string): Promise<ToolExecution> {
    const response = await this.client.get<ToolExecution>(`/api/tools/executions/${executionId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tool execution');
    }
    return response.data!;
  }

  /**
   * Cancel a tool execution
   */
  async cancelExecution(executionId: string): Promise<void> {
    const response = await this.client.delete(`/api/tools/executions/${executionId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to cancel tool execution');
    }
  }
}