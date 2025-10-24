import { RAGOClient } from './client';
import { RAGModule } from './modules/rag';
import { LLMModule } from './modules/llm';
import { ConversationsModule } from './modules/conversations';
import { MCPModule } from './modules/mcp';
import { AnalyticsModule } from './modules/analytics';
import { ToolsModule } from './modules/tools';
import { RAGOConfig } from './types';

/**
 * Main RAGO SDK class
 * Provides access to all RAGO functionality through organized modules
 */
export class RAGO {
  private client: RAGOClient;

  // Module instances
  public readonly rag: RAGModule;
  public readonly llm: LLMModule;
  public readonly conversations: ConversationsModule;
  public readonly mcp: MCPModule;
  public readonly analytics: AnalyticsModule;
  public readonly tools: ToolsModule;

  constructor(config: RAGOConfig) {
    this.client = new RAGOClient(config);
    
    // Initialize modules
    this.rag = new RAGModule(this.client);
    this.llm = new LLMModule(this.client);
    this.conversations = new ConversationsModule(this.client);
    this.mcp = new MCPModule(this.client);
    this.analytics = new AnalyticsModule(this.client);
    this.tools = new ToolsModule(this.client);
  }

  /**
   * Get the base URL of the RAGO server
   */
  getBaseURL(): string {
    return this.client.getBaseURL();
  }

  /**
   * Perform a health check on the RAGO server
   */
  async healthCheck(): Promise<boolean> {
    return this.client.healthCheck();
  }

  /**
   * Get server information
   */
  async getServerInfo(): Promise<{
    status: string;
    version?: string;
    uptime?: number;
    features?: string[];
  }> {
    try {
      const response = await this.client.get('/api/platform/info');
      if (!response.success) {
        throw new Error(response.error || 'Failed to get server info');
      }
      return response.data!;
    } catch (error) {
      return {
        status: 'unavailable',
      };
    }
  }
}

/**
 * Create a new RAGO SDK instance
 */
export function createRAGO(config: RAGOConfig): RAGO {
  return new RAGO(config);
}

// Re-export all types for convenience
export * from './types';