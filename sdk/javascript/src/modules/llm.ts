import { RAGOClient } from '../client';
import {
  GenerateRequest,
  GenerateResponse,
  ChatRequest,
  ChatResponse,
  StructuredGenerateRequest,
  StructuredGenerateResponse,
  StreamOptions,
} from '../types';

/**
 * LLM (Large Language Model) operations module
 * Direct LLM operations without RAG context
 */
export class LLMModule {
  constructor(private client: RAGOClient) {}

  /**
   * Generate text from a prompt
   */
  async generate(request: GenerateRequest): Promise<GenerateResponse> {
    const response = await this.client.post<GenerateResponse>('/api/llm/generate', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to generate text');
    }
    return response.data!;
  }

  /**
   * Generate text with streaming
   */
  async generateStream(request: GenerateRequest, options: StreamOptions): Promise<void> {
    const streamRequest = { ...request, stream: true };
    
    return this.client.postStream(
      '/api/llm/generate',
      streamRequest,
      (chunk: string) => {
        options.onData?.(chunk);
      },
      options.onComplete,
      options.onError
    );
  }

  /**
   * Chat with the LLM
   */
  async chat(request: ChatRequest): Promise<ChatResponse> {
    const response = await this.client.post<ChatResponse>('/api/llm/chat', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to chat with LLM');
    }
    return response.data!;
  }

  /**
   * Chat with streaming
   */
  async chatStream(request: ChatRequest, options: StreamOptions): Promise<void> {
    const streamRequest = { ...request, stream: true };
    
    return this.client.postStream(
      '/api/llm/chat',
      streamRequest,
      (chunk: string) => {
        options.onData?.(chunk);
      },
      options.onComplete,
      options.onError
    );
  }

  /**
   * Generate structured output matching a JSON schema
   */
  async generateStructured(request: StructuredGenerateRequest): Promise<StructuredGenerateResponse> {
    const response = await this.client.post<StructuredGenerateResponse>('/api/llm/structured', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to generate structured output');
    }
    return response.data!;
  }
}