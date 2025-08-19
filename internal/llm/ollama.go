package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liliang-cn/ollama-go"
	"github.com/liliang-cn/rago/internal/domain"
)

type OllamaService struct {
	client  *ollama.Client
	model   string
	baseURL string
}

func NewOllamaService(baseURL, model string) (*OllamaService, error) {
	client, err := ollama.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama client: %w", err)
	}

	// The client will automatically use OLLAMA_HOST from env if available.

	return &OllamaService{
		client:  client,
		model:   model,
		baseURL: baseURL,
	}, nil
}

func (s *OllamaService) Generate(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	stream := false
	req := &ollama.GenerateRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: &stream,
	}

	if opts != nil {
		options := &ollama.Options{}
		if opts.Temperature >= 0 {
			options.Temperature = &opts.Temperature
		}
		if opts.MaxTokens > 0 {
			numPredict := opts.MaxTokens
			options.NumPredict = &numPredict
		}
		req.Options = options
	}

	resp, err := s.client.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
	}

	return resp.Response, nil
}

func (s *OllamaService) Stream(ctx context.Context, prompt string, opts *domain.GenerationOptions, callback func(string)) error {
	if prompt == "" {
		return fmt.Errorf("%w: empty prompt", domain.ErrInvalidInput)
	}

	if callback == nil {
		return fmt.Errorf("%w: nil callback", domain.ErrInvalidInput)
	}

	// Use the functional API like in your working example
	options := []func(*ollama.GenerateRequest){}

	if opts != nil {
		if opts.Temperature >= 0 {
			options = append(options, func(req *ollama.GenerateRequest) {
				if req.Options == nil {
					req.Options = &ollama.Options{}
				}
				req.Options.Temperature = &opts.Temperature
			})
		}
		if opts.MaxTokens > 0 {
			options = append(options, func(req *ollama.GenerateRequest) {
				if req.Options == nil {
					req.Options = &ollama.Options{}
				}
				req.Options.NumPredict = &opts.MaxTokens
			})
		}
	}

	// Use the functional API that works correctly
	respCh, errCh := ollama.GenerateStream(ctx, s.model, prompt, options...)

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				// Channel closed, streaming is done
				return nil
			}
			if resp != nil && resp.Response != "" {
				callback(resp.Response)
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("%w: %v", domain.ErrGenerationFailed, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *OllamaService) Health(ctx context.Context) error {
	_, err := s.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}

const metadataExtractionPromptTemplate = `You are an expert data extractor. Analyze the following document content and return ONLY a single valid JSON object with the following fields:
- "summary": A concise, one-sentence summary of the document.
- "keywords": An array of 3 to 5 most relevant keywords.
- "document_type": The type of the document (e.g., "Article", "Meeting Notes", "Technical Manual", "Code Snippet", "Essay").
- "creation_date": The creation date found in the document text in "YYYY-MM-DD" format. If no date is found, use null.

Document Content:
"""
%s
"""

JSON Output:`

func (s *OllamaService) ExtractMetadata(ctx context.Context, content string, model string) (*domain.ExtractedMetadata, error) {
	if content == "" {
		return nil, fmt.Errorf("%w: content cannot be empty", domain.ErrInvalidInput)
	}

	prompt := fmt.Sprintf(metadataExtractionPromptTemplate, content)

	stream := false
	format := "json"
	req := &ollama.GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: &stream,
		Format: &format,
	}

	resp, err := s.client.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: metadata extraction failed: %v", domain.ErrGenerationFailed, err)
	}

	var metadata domain.ExtractedMetadata
	if err := json.Unmarshal([]byte(resp.Response), &metadata); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal metadata response: %v. Raw response: %s", domain.ErrInvalidInput, err, resp.Response)
	}

	return &metadata, nil
}

func ComposePrompt(chunks []domain.Chunk, query string) string {
	if len(chunks) == 0 {
		return fmt.Sprintf("请回答以下问题：\n\n%s", query)
	}

	var contextParts []string
	for i, chunk := range chunks {
		contextParts = append(contextParts, fmt.Sprintf("[文档片段 %d]\n%s", i+1, chunk.Content))
	}

	context := strings.Join(contextParts, "\n\n")

	prompt := fmt.Sprintf(`基于以下文档内容，请回答用户的问题。如果文档中没有相关信息，请说明无法从提供的文档中找到答案。

文档内容：
%s

用户问题：%s

请根据文档内容给出详细、准确的回答：`, context, query)

	return prompt
}
