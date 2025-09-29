import { RAGOClient } from '../client';
import {
  UsageStats,
  RAGAnalytics,
  RAGQueryRecord,
  RAGPerformanceMetrics,
  FilterOptions,
  Conversation,
} from '../types';

/**
 * Analytics and usage statistics module
 */
export class AnalyticsModule {
  constructor(private client: RAGOClient) {}

  // === RAG Analytics ===

  /**
   * Get RAG analytics overview
   */
  async getRAGAnalytics(filters?: FilterOptions): Promise<RAGAnalytics> {
    const response = await this.client.get<RAGAnalytics>(
      '/api/v1/rag/analytics',
      this.client.createConfig(filters)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get RAG analytics');
    }
    return response.data!;
  }

  /**
   * Get RAG performance metrics
   */
  async getRAGPerformance(filters?: FilterOptions): Promise<RAGPerformanceMetrics> {
    const response = await this.client.get<RAGPerformanceMetrics>(
      '/api/v1/rag/performance',
      this.client.createConfig(filters)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get RAG performance metrics');
    }
    return response.data!;
  }

  /**
   * Get RAG query records
   */
  async getRAGQueries(filters?: FilterOptions): Promise<RAGQueryRecord[]> {
    const response = await this.client.get<RAGQueryRecord[]>(
      '/api/v1/rag/queries',
      this.client.createConfig(filters)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get RAG queries');
    }
    return response.data!;
  }

  // === Usage Statistics ===

  /**
   * Get overall usage statistics
   */
  async getUsageStats(): Promise<UsageStats> {
    const response = await this.client.get<UsageStats>('/api/v1/usage/stats');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get usage statistics');
    }
    return response.data!;
  }

  /**
   * Get usage statistics by type
   */
  async getUsageStatsByType(): Promise<Record<string, UsageStats>> {
    const response = await this.client.get<Record<string, UsageStats>>('/api/v1/usage/stats/type');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get usage statistics by type');
    }
    return response.data!;
  }

  /**
   * Get usage statistics by provider
   */
  async getUsageStatsByProvider(): Promise<Record<string, UsageStats>> {
    const response = await this.client.get<Record<string, UsageStats>>('/api/v1/usage/stats/provider');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get usage statistics by provider');
    }
    return response.data!;
  }

  /**
   * Get usage statistics by model
   */
  async getUsageStatsByModel(limit?: number): Promise<Array<{ model: string; stats: UsageStats }>> {
    const params = limit ? { limit } : undefined;
    const response = await this.client.get<Array<{ model: string; stats: UsageStats }>>(
      '/api/v1/usage/stats/models',
      this.client.createConfig(params)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get usage statistics by model');
    }
    return response.data!;
  }

  /**
   * Get daily usage statistics
   */
  async getDailyUsageStats(days?: number): Promise<Array<{ date: string; stats: UsageStats }>> {
    const params = days ? { days } : undefined;
    const response = await this.client.get<Array<{ date: string; stats: UsageStats }>>(
      '/api/v1/usage/stats/daily',
      this.client.createConfig(params)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get daily usage statistics');
    }
    return response.data!;
  }

  /**
   * Get usage cost breakdown
   */
  async getUsageCost(): Promise<{
    total_cost: number;
    cost_by_provider: Record<string, number>;
    cost_by_model: Record<string, number>;
    cost_trend: Array<{ date: string; cost: number }>;
  }> {
    const response = await this.client.get('/api/v1/usage/stats/cost');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get usage cost');
    }
    return response.data!;
  }

  // === Tool Call Analytics ===

  /**
   * Get tool call statistics
   */
  async getToolCallStats(): Promise<{
    total_calls: number;
    success_rate: number;
    avg_execution_time: number;
    most_used_tools: Array<{ tool: string; count: number }>;
  }> {
    const response = await this.client.get('/api/v1/tool-calls/stats');
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tool call statistics');
    }
    return response.data!;
  }

  /**
   * Get tool call records
   */
  async getToolCalls(filters?: FilterOptions): Promise<any[]> {
    const response = await this.client.get<any[]>(
      '/api/v1/tool-calls',
      this.client.createConfig(filters)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tool calls');
    }
    return response.data!;
  }

  /**
   * Get tool call analytics
   */
  async getToolCallAnalytics(filters?: FilterOptions): Promise<{
    total_calls: number;
    success_rate: number;
    avg_execution_time: number;
    calls_by_tool: Record<string, number>;
    performance_trend: Array<{ date: string; avg_time: number; count: number }>;
  }> {
    const response = await this.client.get(
      '/api/v1/tool-calls/analytics',
      this.client.createConfig(filters)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get tool call analytics');
    }
    return response.data!;
  }

  // === Conversations Analytics ===

  /**
   * Get conversations (using v1 API for analytics)
   */
  async getConversations(filters?: FilterOptions): Promise<Conversation[]> {
    const response = await this.client.get<Conversation[]>(
      '/api/v1/conversations',
      this.client.createConfig(filters)
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to get conversations');
    }
    return response.data!;
  }

  /**
   * Get conversation by ID (using v1 API for analytics)
   */
  async getConversation(conversationId: string): Promise<Conversation> {
    const response = await this.client.get<Conversation>(`/api/v1/conversations/${conversationId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get conversation');
    }
    return response.data!;
  }
}