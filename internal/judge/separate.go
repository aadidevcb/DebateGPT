package judge

import (
	"context"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/debate"
)

// SeparateJudge uses a dedicated model for judging — most objective.
type SeparateJudge struct {
	agent agent.StreamAgent
}

func (j *SeparateJudge) Mode() string { return "separate" }

func (j *SeparateJudge) Synthesize(ctx context.Context, transcript *debate.Transcript) (<-chan agent.StreamEvent, error) {
	messages := []agent.Message{
		{Role: "system", Content: JudgeSystemPrompt},
		{Role: "user", Content: BuildJudgePrompt(transcript)},
	}
	return j.agent.StreamGenerate(ctx, messages)
}
