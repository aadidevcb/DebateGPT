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

// ClaudeAgent implements StreamAgent for Anthropic's Claude API.
type ClaudeAgent struct {
	name      string
	model     string
	apiKey    string
	maxTokens int
	client    *http.Client
}

// NewClaudeAgent creates a new Claude agent from config.
func NewClaudeAgent(cfg agent.AgentConfig) (agent.StreamAgent, error) {
	name := cfg.Name
	if name == "" {
		name = "claude"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	return &ClaudeAgent{
		name:      name,
		model:     cfg.Model,
		apiKey:    cfg.APIKey,
		maxTokens: maxTokens,
		client:    &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (a *ClaudeAgent) Name() string  { return a.name }
func (a *ClaudeAgent) Model() string { return a.model }

type claudeRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []claudeMessage  `json:"messages"`
	Stream    bool             `json:"stream"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta,omitempty"`
	Message *struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

// Generate performs a non-streaming completion.
func (a *ClaudeAgent) Generate(ctx context.Context, messages []agent.Message) (agent.Response, error) {
	start := time.Now()

	system, msgs := a.splitSystemMessage(messages)

	reqBody := claudeRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		System:    system,
		Messages:  msgs,
		Stream:    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return agent.Response{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return agent.Response{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return agent.Response{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return agent.Response{}, fmt.Errorf("claude API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return agent.Response{}, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	if len(result.Content) > 0 {
		content = result.Content[0].Text
	}

	return agent.Response{
		Content:      content,
		TokensIn:     result.Usage.InputTokens,
		TokensOut:    result.Usage.OutputTokens,
		FinishReason: result.StopReason,
		Latency:      time.Since(start),
	}, nil
}

// StreamGenerate performs a streaming completion.
func (a *ClaudeAgent) StreamGenerate(ctx context.Context, messages []agent.Message) (<-chan agent.StreamEvent, error) {
	system, msgs := a.splitSystemMessage(messages)

	reqBody := claudeRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		System:    system,
		Messages:  msgs,
		Stream:    true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claude API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan agent.StreamEvent, 100)
	go a.processStream(ctx, resp.Body, ch)

	return ch, nil
}

func (a *ClaudeAgent) processStream(ctx context.Context, body io.ReadCloser, ch chan<- agent.StreamEvent) {
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

		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				ch <- agent.StreamEvent{Type: agent.EventDelta, Delta: event.Delta.Text}
			}
		case "message_start":
			if event.Message != nil {
				totalIn = event.Message.Usage.InputTokens
			}
		case "message_delta":
			if event.Usage != nil {
				totalOut = event.Usage.OutputTokens
			}
			finishReason = "end_turn"
		case "message_stop":
			// handled below
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

// splitSystemMessage extracts the system message (if any) from the message list.
// Claude uses a top-level "system" field rather than a system role message.
func (a *ClaudeAgent) splitSystemMessage(messages []agent.Message) (string, []claudeMessage) {
	var system string
	var msgs []claudeMessage

	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		} else {
			msgs = append(msgs, claudeMessage{Role: m.Role, Content: m.Content})
		}
	}

	return system, msgs
}
