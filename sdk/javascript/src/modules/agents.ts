import { RAGOClient } from '../client';
import {
  AgentRunRequest,
  AgentRunResponse,
  AgentPlanRequest,
  AgentPlanResponse,
} from '../types';

/**
 * Agent operations module
 * Handles AI agent automation and workflow execution
 */
export class AgentsModule {
  constructor(private client: RAGOClient) {}

  /**
   * Run an agent to complete a task
   */
  async run(request: AgentRunRequest): Promise<AgentRunResponse> {
    const response = await this.client.post<AgentRunResponse>('/api/platform/agent/run', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to run agent');
    }
    return response.data!;
  }

  /**
   * Create a plan for a task without executing it
   */
  async plan(request: AgentPlanRequest): Promise<AgentPlanResponse> {
    const response = await this.client.post<AgentPlanResponse>('/api/platform/agent/plan', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to create agent plan');
    }
    return response.data!;
  }
}