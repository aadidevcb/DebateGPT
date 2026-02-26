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

// GeminiAgent implements StreamAgent for Google's Gemini API.
type GeminiAgent struct {
	name      string
	model     string
	apiKey    string
	maxTokens int
	client    *http.Client
}

// NewGeminiAgent creates a new Gemini agent from config.
func NewGeminiAgent(cfg agent.AgentConfig) (agent.StreamAgent, error) {
	name := cfg.Name
	if name == "" {
		name = "gemini"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	return &GeminiAgent{
		name:      name,
		model:     cfg.Model,
		apiKey:    cfg.APIKey,
		maxTokens: maxTokens,
		client:    &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (a *GeminiAgent) Name() string  { return a.name }
func (a *GeminiAgent) Model() string { return a.model }

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

type geminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason,omitempty"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

// Generate performs a non-streaming completion.
func (a *GeminiAgent) Generate(ctx context.Context, messages []agent.Message) (agent.Response, error) {
	start := time.Now()

	gemReq := a.buildRequest(messages)

	body, err := json.Marshal(gemReq)
	if err != nil {
		return agent.Response{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", a.model, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return agent.Response{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return agent.Response{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return agent.Response{}, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result geminiStreamChunk
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return agent.Response{}, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	finishReason := ""
	if len(result.Candidates) > 0 {
		if len(result.Candidates[0].Content.Parts) > 0 {
			content = result.Candidates[0].Content.Parts[0].Text
		}
		finishReason = result.Candidates[0].FinishReason
	}

	var tokensIn, tokensOut int
	if result.UsageMetadata != nil {
		tokensIn = result.UsageMetadata.PromptTokenCount
		tokensOut = result.UsageMetadata.CandidatesTokenCount
	}

	return agent.Response{
		Content:      content,
		TokensIn:     tokensIn,
		TokensOut:    tokensOut,
		FinishReason: finishReason,
		Latency:      time.Since(start),
	}, nil
}

// StreamGenerate performs a streaming completion using Gemini's SSE stream.
func (a *GeminiAgent) StreamGenerate(ctx context.Context, messages []agent.Message) (<-chan agent.StreamEvent, error) {
	gemReq := a.buildRequest(messages)

	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", a.model, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan agent.StreamEvent, 100)
	go a.processStream(ctx, resp.Body, ch)

	return ch, nil
}

func (a *GeminiAgent) processStream(ctx context.Context, body io.ReadCloser, ch chan<- agent.StreamEvent) {
	defer close(ch)
	defer body.Close()

	start := time.Now()
	scanner := bufio.NewScanner(body)
	// Gemini can return large chunks
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
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

		var chunk geminiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) > 0 {
			candidate := chunk.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				text := candidate.Content.Parts[0].Text
				if text != "" {
					ch <- agent.StreamEvent{Type: agent.EventDelta, Delta: text}
				}
			}
			if candidate.FinishReason != "" {
				finishReason = candidate.FinishReason
			}
		}

		if chunk.UsageMetadata != nil {
			totalIn = chunk.UsageMetadata.PromptTokenCount
			totalOut = chunk.UsageMetadata.CandidatesTokenCount
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

func (a *GeminiAgent) buildRequest(messages []agent.Message) geminiRequest {
	req := geminiRequest{
		GenerationConfig: &geminiGenConfig{
			MaxOutputTokens: a.maxTokens,
		},
	}

	for _, m := range messages {
		if m.Role == "system" {
			req.SystemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: m.Content}},
			}
		} else {
			role := m.Role
			if role == "assistant" {
				role = "model"
			}
			req.Contents = append(req.Contents, geminiContent{
				Role:  role,
				Parts: []geminiPart{{Text: m.Content}},
			})
		}
	}

	return req
}
