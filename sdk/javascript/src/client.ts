import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { RAGOConfig, APIResponse } from './types';

/**
 * Core HTTP client for RAGO API
 */
export class RAGOClient {
  private readonly axios: AxiosInstance;
  private readonly baseURL: string;

  constructor(config: RAGOConfig) {
    this.baseURL = config.baseURL.replace(/\/$/, ''); // Remove trailing slash
    
    this.axios = axios.create({
      baseURL: this.baseURL,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
        ...config.headers,
        ...(config.apiKey && { 'Authorization': `Bearer ${config.apiKey}` }),
      },
    });

    // Request interceptor for logging (if needed)
    this.axios.interceptors.request.use((config) => {
      return config;
    });

    // Response interceptor for error handling
    this.axios.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response) {
          // Server responded with error status
          const apiError = new Error(
            error.response.data?.error || 
            error.response.data?.message || 
            `HTTP ${error.response.status}: ${error.response.statusText}`
          );
          (apiError as any).status = error.response.status;
          (apiError as any).response = error.response.data;
          throw apiError;
        } else if (error.request) {
          // Network error
          throw new Error(`Network error: ${error.message}`);
        } else {
          // Request setup error
          throw new Error(`Request error: ${error.message}`);
        }
      }
    );
  }

  /**
   * Make a GET request
   */
  async get<T = any>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>> {
    const response: AxiosResponse<APIResponse<T>> = await this.axios.get(url, config);
    return response.data;
  }

  /**
   * Make a POST request
   */
  async post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<APIResponse<T>> {
    const response: AxiosResponse<APIResponse<T>> = await this.axios.post(url, data, config);
    return response.data;
  }

  /**
   * Make a PUT request
   */
  async put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<APIResponse<T>> {
    const response: AxiosResponse<APIResponse<T>> = await this.axios.put(url, data, config);
    return response.data;
  }

  /**
   * Make a DELETE request
   */
  async delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<APIResponse<T>> {
    const response: AxiosResponse<APIResponse<T>> = await this.axios.delete(url, config);
    return response.data;
  }

  /**
   * Make a streaming POST request with callback
   */
  async postStream(
    url: string, 
    data: any, 
    onData: (chunk: string) => void,
    onComplete?: () => void,
    onError?: (error: Error) => void
  ): Promise<void> {
    try {
      const response = await this.axios.post(url, data, {
        responseType: 'stream',
        headers: {
          'Accept': 'text/event-stream',
        },
      });

      let buffer = '';
      
      response.data.on('data', (chunk: Buffer) => {
        buffer += chunk.toString();
        
        // Process complete lines
        const lines = buffer.split('\n');
        buffer = lines.pop() || ''; // Keep incomplete line in buffer
        
        for (const line of lines) {
          if (line.trim()) {
            try {
              // Handle server-sent events format
              if (line.startsWith('data: ')) {
                const data = line.substring(6);
                if (data === '[DONE]') {
                  onComplete?.();
                  return;
                }
                onData(data);
              } else {
                // Handle raw text streams
                onData(line);
              }
            } catch (e) {
              // If not JSON, treat as raw text
              onData(line);
            }
          }
        }
      });

      response.data.on('end', () => {
        // Process any remaining buffer
        if (buffer.trim()) {
          onData(buffer);
        }
        onComplete?.();
      });

      response.data.on('error', (error: Error) => {
        onError?.(error);
      });

    } catch (error) {
      onError?.(error as Error);
    }
  }

  /**
   * Get the base URL
   */
  getBaseURL(): string {
    return this.baseURL;
  }

  /**
   * Health check
   */
  async healthCheck(): Promise<boolean> {
    try {
      const response = await this.get('/api/health');
      return response.success;
    } catch {
      return false;
    }
  }

  /**
   * Create request config with query parameters
   */
  createConfig(params?: Record<string, any>): AxiosRequestConfig {
    return params ? { params } : {};
  }
}