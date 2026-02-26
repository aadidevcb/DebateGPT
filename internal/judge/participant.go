package judge

import (
	"context"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/debate"
)

// ParticipantJudge reuses one of the debating agents as the judge.
// Cheapest option but slightly biased toward that agent's perspective.
type ParticipantJudge struct {
	agent agent.StreamAgent
	name  string
}

func (j *ParticipantJudge) Mode() string { return "participant" }

func (j *ParticipantJudge) Synthesize(ctx context.Context, transcript *debate.Transcript) (<-chan agent.StreamEvent, error) {
	messages := []agent.Message{
		{Role: "system", Content: JudgeSystemPrompt},
		{Role: "user", Content: BuildJudgePrompt(transcript)},
	}
	return j.agent.StreamGenerate(ctx, messages)
}
