import { RAGOClient } from '../client';
import {
  Conversation,
  ConversationSummary,
  ChatMessage,
} from '../types';

/**
 * Conversation management module
 */
export class ConversationsModule {
  constructor(private client: RAGOClient) {}

  /**
   * Create a new conversation
   */
  async createNew(): Promise<Conversation> {
    const response = await this.client.post<Conversation>('/api/conversations/new');
    if (!response.success) {
      throw new Error(response.error || 'Failed to create new conversation');
    }
    return response.data!;
  }

  /**
   * Save a conversation
   */
  async save(conversation: Conversation): Promise<void> {
    const response = await this.client.post('/api/conversations/save', conversation);
    if (!response.success) {
      throw new Error(response.error || 'Failed to save conversation');
    }
  }

  /**
   * List all conversations
   */
  async list(): Promise<ConversationSummary[]> {
    const response = await this.client.get<ConversationSummary[]>('/api/conversations');
    if (!response.success) {
      throw new Error(response.error || 'Failed to list conversations');
    }
    return response.data!;
  }

  /**
   * Get a specific conversation by ID
   */
  async get(conversationId: string): Promise<Conversation> {
    const response = await this.client.get<Conversation>(`/api/conversations/${conversationId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get conversation');
    }
    return response.data!;
  }

  /**
   * Delete a conversation
   */
  async delete(conversationId: string): Promise<void> {
    const response = await this.client.delete(`/api/conversations/${conversationId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to delete conversation');
    }
  }

  /**
   * Search conversations
   */
  async search(query: string): Promise<ConversationSummary[]> {
    const response = await this.client.get<ConversationSummary[]>(
      '/api/conversations/search',
      this.client.createConfig({ q: query })
    );
    if (!response.success) {
      throw new Error(response.error || 'Failed to search conversations');
    }
    return response.data!;
  }
}