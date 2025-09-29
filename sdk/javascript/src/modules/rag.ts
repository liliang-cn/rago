import { RAGOClient } from '../client';
import {
  IngestRequest,
  IngestResponse,
  QueryRequest,
  QueryResponse,
  Document,
  SearchRequest,
  SearchResponse,
  HybridSearchRequest,
  StreamOptions,
} from '../types';

/**
 * RAG (Retrieval-Augmented Generation) operations module
 */
export class RAGModule {
  constructor(private client: RAGOClient) {}

  /**
   * Ingest content into the RAG system
   */
  async ingest(request: IngestRequest): Promise<IngestResponse> {
    const response = await this.client.post<IngestResponse>('/api/rag/ingest', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to ingest content');
    }
    return response.data!;
  }

  /**
   * Query the RAG system
   */
  async query(request: QueryRequest): Promise<QueryResponse> {
    const response = await this.client.post<QueryResponse>('/api/rag/query', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to query RAG system');
    }
    return response.data!;
  }

  /**
   * Stream query the RAG system
   */
  async queryStream(request: QueryRequest, options: StreamOptions): Promise<void> {
    const streamRequest = { ...request, stream: true };
    
    return this.client.postStream(
      '/api/rag/query-stream',
      streamRequest,
      (chunk: string) => {
        options.onData?.(chunk);
      },
      options.onComplete,
      options.onError
    );
  }

  /**
   * Search without generation (retrieval only)
   */
  async search(request: SearchRequest): Promise<SearchResponse> {
    const response = await this.client.post<SearchResponse>('/api/rag/search', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to search');
    }
    return response.data!;
  }

  /**
   * Semantic search
   */
  async semanticSearch(request: SearchRequest): Promise<SearchResponse> {
    const response = await this.client.post<SearchResponse>('/api/rag/search/semantic', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to perform semantic search');
    }
    return response.data!;
  }

  /**
   * Hybrid search (semantic + keyword)
   */
  async hybridSearch(request: HybridSearchRequest): Promise<SearchResponse> {
    const response = await this.client.post<SearchResponse>('/api/rag/search/hybrid', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to perform hybrid search');
    }
    return response.data!;
  }

  /**
   * Filtered search
   */
  async filteredSearch(request: SearchRequest): Promise<SearchResponse> {
    const response = await this.client.post<SearchResponse>('/api/rag/search/filtered', request);
    if (!response.success) {
      throw new Error(response.error || 'Failed to perform filtered search');
    }
    return response.data!;
  }

  /**
   * List all documents
   */
  async listDocuments(): Promise<Document[]> {
    const response = await this.client.get<Document[]>('/api/rag/documents');
    if (!response.success) {
      throw new Error(response.error || 'Failed to list documents');
    }
    return response.data!;
  }

  /**
   * List documents with detailed information
   */
  async listDocumentsWithInfo(): Promise<Document[]> {
    const response = await this.client.get<Document[]>('/api/rag/documents/info');
    if (!response.success) {
      throw new Error(response.error || 'Failed to list documents with info');
    }
    return response.data!;
  }

  /**
   * Get document information by ID
   */
  async getDocument(documentId: string): Promise<Document> {
    const response = await this.client.get<Document>(`/api/rag/documents/${documentId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to get document');
    }
    return response.data!;
  }

  /**
   * Delete a document by ID
   */
  async deleteDocument(documentId: string): Promise<void> {
    const response = await this.client.delete(`/api/rag/documents/${documentId}`);
    if (!response.success) {
      throw new Error(response.error || 'Failed to delete document');
    }
  }

  /**
   * Reset the RAG system (clear all data)
   */
  async reset(): Promise<void> {
    const response = await this.client.post('/api/rag/reset');
    if (!response.success) {
      throw new Error(response.error || 'Failed to reset RAG system');
    }
  }
}