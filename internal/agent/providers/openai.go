package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aadidev/debategpt/internal/agent"
)

// OpenAIAgent implements StreamAgent for OpenAI's API.
type OpenAIAgent struct {
	name      string
	model     string
	apiKey    string
	baseURL   string
	maxTokens int
	client    *http.Client
}

// NewOpenAIAgent creates a new OpenAI agent from config.
func NewOpenAIAgent(cfg agent.AgentConfig) (agent.StreamAgent, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	name := cfg.Name
	if name == "" {
		name = "openai"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	return &OpenAIAgent{
		name:      name,
		model:     cfg.Model,
		apiKey:    cfg.APIKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		maxTokens: maxTokens,
		client:    &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (a *OpenAIAgent) Name() string  { return a.name }
func (a *OpenAIAgent) Model() string { return a.model }

// openAIChatRequest is the request body for chat completions.
type openAIChatRequest struct {
	Model     string           `json:"model"`
	Messages  []openAIMessage  `json:"messages"`
	MaxTokens int              `json:"max_tokens,omitempty"`
	Stream    bool             `json:"stream"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Generate performs a non-streaming chat completion.
func (a *OpenAIAgent) Generate(ctx context.Context, messages []agent.Message) (agent.Response, error) {
	start := time.Now()

	msgs := make([]openAIMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := openAIChatRequest{
		Model:     a.model,
		Messages:  msgs,
		MaxTokens: a.maxTokens,
		Stream:    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return agent.Response{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return agent.Response{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return agent.Response{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return agent.Response{}, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return agent.Response{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return agent.Response{}, fmt.Errorf("no choices in response")
	}

	return agent.Response{
		Content:      result.Choices[0].Message.Content,
		TokensIn:     result.Usage.PromptTokens,
		TokensOut:    result.Usage.CompletionTokens,
		FinishReason: result.Choices[0].FinishReason,
		Latency:      time.Since(start),
	}, nil
}

// StreamGenerate performs a streaming chat completion.
func (a *OpenAIAgent) StreamGenerate(ctx context.Context, messages []agent.Message) (<-chan agent.StreamEvent, error) {
	msgs := make([]openAIMessage, len(messages))
	for i, m := range messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := openAIChatRequest{
		Model:     a.model,
		Messages:  msgs,
		MaxTokens: a.maxTokens,
		Stream:    true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan agent.StreamEvent, 100)
	go a.processStream(ctx, resp.Body, ch)

	return ch, nil
}

func (a *OpenAIAgent) processStream(ctx context.Context, body io.ReadCloser, ch chan<- agent.StreamEvent) {
	defer close(ch)
	defer body.Close()

	start := time.Now()
	scanner := bufio.NewScanner(body)
	var totalIn, totalOut int
	var finishReason string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- agent.StreamEvent{Type: agent.EventError, Error: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				ch <- agent.StreamEvent{Type: agent.EventDelta, Delta: delta}
			}
			if chunk.Choices[0].FinishReason != nil {
				finishReason = *chunk.Choices[0].FinishReason
			}
		}

		if chunk.Usage != nil {
			totalIn = chunk.Usage.PromptTokens
			totalOut = chunk.Usage.CompletionTokens
		}
	}

	ch <- agent.StreamEvent{
		Type: agent.EventDone,
		Metrics: &agent.Metrics{
			TokensIn:     totalIn,
			TokensOut:    totalOut,
			Latency:      time.Since(start),
			FinishReason: finishReason,
		},
	}
}
