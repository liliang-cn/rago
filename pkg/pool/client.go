package pool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// Client OpenAI兼容的LLM/Embedding Client
type Client struct {
	baseURL   string
	key       string
	modelName string
	http      *http.Client
}

// NewClient 创建新client
func NewClient(baseURL, key, modelName string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model_name is required")
	}

	return &Client{
		baseURL:   baseURL,
		key:       key,
		modelName: modelName,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Generate 生成文本
func (c *Client) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}

	reqBody := map[string]interface{}{
		"model": c.modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	// 添加可选参数
	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}

	resp, err := c.doRequest(ctx, "/chat/completions", reqBody)
	if err != nil {
		return "", err
	}

	// 解析响应
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

// Stream 流式生成
func (c *Client) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}

	reqBody := map[string]interface{}{
		"model": c.modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": true,
	}

	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	// 处理SSE流
	for {
		var line bytes.Buffer
		for {
			b := make([]byte, 1)
			_, err := resp.Body.Read(b)
			if err != nil {
				return nil
			}
			if b[0] == '\n' {
				break
			}
			if b[0] != '\r' {
				line.Write(b)
			}
		}

		lineStr := line.String()
		if lineStr == "" {
			continue
		}
		if lineStr == "data: [DONE]" {
			break
		}
		if len(lineStr) < 6 || lineStr[:6] != "data: " {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(lineStr[6:]), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			callback(chunk.Choices[0].Delta.Content)
		}
	}

	return nil
}

// GenerateWithTools 使用工具生成
func (c *Client) GenerateWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions) (*domain.GenerationResult, error) {
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}

	// 转换messages格式
	apiMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		apiMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ToolCalls != nil {
			apiMessages[i]["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			apiMessages[i]["tool_call_id"] = msg.ToolCallID
		}
	}

	// 转换tools格式
	apiTools := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		apiTools[i] = map[string]interface{}{
			"type":     "function",
			"function": tool.Function,
		}
	}

	reqBody := map[string]interface{}{
		"model":    c.modelName,
		"messages": apiMessages,
		"tools":    apiTools,
	}

	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}

	resp, err := c.doRequest(ctx, "/chat/completions", reqBody)
	if err != nil {
		return nil, err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string            `json:"content"`
				Role      string            `json:"role"`
				ToolCalls []json.RawMessage `json:"tool_calls,omitempty"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := result.Choices[0]
	response := &domain.GenerationResult{
		Content: choice.Message.Content,
	}

	// 解析tool calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]domain.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var toolCall struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}
			if err := json.Unmarshal(tc, &toolCall); err == nil {
				response.ToolCalls[i] = domain.ToolCall{
					ID:   toolCall.ID,
					Type: toolCall.Type,
					Function: domain.FunctionCall{
						Name: toolCall.Function.Name,
					},
				}
				// Parse arguments as map
				if toolCall.Function.Arguments != "" {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						response.ToolCalls[i].Function.Arguments = args
					}
				}
			}
		}
	}

	return response, nil
}

// StreamWithTools 流式工具调用
func (c *Client) StreamWithTools(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition, opts *domain.GenerationOptions, callback domain.ToolCallCallback) error {
	// 简化实现：先获取完整结果再流式回调
	result, err := c.GenerateWithTools(ctx, messages, tools, opts)
	if err != nil {
		return err
	}

	// 回调内容
	if result.Content != "" {
		// ToolCallCallback签名: func(chunk string, toolCalls []ToolCall) error
		if err := callback(result.Content, nil); err != nil {
			return err
		}
	}

	// 回调工具调用
	if len(result.ToolCalls) > 0 {
		return callback("", result.ToolCalls)
	}

	return nil
}

// GenerateStructured 结构化生成
func (c *Client) GenerateStructured(ctx context.Context, prompt string, schema interface{}, opts *domain.GenerationOptions) (*domain.StructuredResult, error) {
	if opts == nil {
		opts = &domain.GenerationOptions{}
	}

	reqBody := map[string]interface{}{
		"model": c.modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]interface{}{
			"type":        "json_schema",
			"json_schema": schema,
		},
	}

	if opts.Temperature > 0 {
		reqBody["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody["max_tokens"] = opts.MaxTokens
	}

	resp, err := c.doRequest(ctx, "/chat/completions", reqBody)
	if err != nil {
		return nil, err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &domain.StructuredResult{
		Raw: result.Choices[0].Message.Content,
	}, nil
}

// RecognizeIntent 意图识别
func (c *Client) RecognizeIntent(ctx context.Context, request string) (*domain.IntentResult, error) {
	prompt := fmt.Sprintf(`Analyze the following user request and determine the intent. Respond with JSON:
{
  "intent": "question|action|analysis|search|calculation|status|unknown",
  "confidence": 0.0-1.0,
  "key_verbs": ["verb1", "verb2"],
  "entities": ["entity1", "entity2"],
  "needs_tools": true/false,
  "reasoning": "brief explanation"
}

User request: %s`, request)

	result, err := c.Generate(ctx, prompt, &domain.GenerationOptions{Temperature: 0.1})
	if err != nil {
		return nil, err
	}

	var intentResult domain.IntentResult
	if err := json.Unmarshal([]byte(result), &intentResult); err != nil {
		// 失败时返回默认值
		return &domain.IntentResult{
			Intent:     domain.IntentUnknown,
			Confidence: 0.5,
			KeyVerbs:   []string{},
			Entities:   []string{},
		}, nil
	}

	return &intentResult, nil
}

// Embed 向量化 (返回[]float64以兼容domain.Embedder接口)
func (c *Client) Embed(ctx context.Context, texts []string) ([]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	// OpenAI embedding API returns the first embedding's vector
	if len(texts) == 1 {
		return c.embedSingle(ctx, texts[0])
	}

	// For multiple texts, return the first one's vector
	// (This is a simplification - in production you might want to handle this differently)
	return c.embedSingle(ctx, texts[0])
}

// EmbedMultiple 向量化多个文本，返回多个向量
func (c *Client) EmbedMultiple(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	reqBody := map[string]interface{}{
		"model": c.modelName,
		"input": texts,
	}

	resp, err := c.doRequest(ctx, "/embeddings", reqBody)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	embeddings := make([][]float64, len(result.Data))
	for _, item := range result.Data {
		vec := make([]float64, len(item.Embedding))
		for i, v := range item.Embedding {
			vec[i] = float64(v)
		}
		embeddings[item.Index] = vec
	}

	return embeddings, nil
}

// embedSingle 向量化单个文本
func (c *Client) embedSingle(ctx context.Context, text string) ([]float64, error) {
	reqBody := map[string]interface{}{
		"model": c.modelName,
		"input": []string{text},
	}

	resp, err := c.doRequest(ctx, "/embeddings", reqBody)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert []float32 to []float64
	vec := make([]float64, len(result.Data[0].Embedding))
	for i, v := range result.Data[0].Embedding {
		vec[i] = float64(v)
	}

	return vec, nil
}

// Health 健康检查
func (c *Client) Health(ctx context.Context) error {
	// 简单检查：发起一个最小embedding请求
	_, err := c.embedSingle(ctx, "health")
	return err
}

// Close 关闭client
func (c *Client) Close() error {
	c.http.CloseIdleConnections()
	return nil
}

// doRequest 执行HTTP请求
func (c *Client) doRequest(ctx context.Context, path string, body interface{}) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
