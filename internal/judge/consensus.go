package judge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/debate"
)

// ConsensusJudge asks all agents to summarize, then merges their summaries.
// Most expensive but captures every perspective.
type ConsensusJudge struct {
	agents []agent.StreamAgent
	merger agent.StreamAgent
}

func (j *ConsensusJudge) Mode() string { return "consensus" }

func (j *ConsensusJudge) Synthesize(ctx context.Context, transcript *debate.Transcript) (<-chan agent.StreamEvent, error) {
	// Phase 1: Each agent produces a summary
	summaries := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, a := range j.agents {
		wg.Add(1)
		go func(ag agent.StreamAgent) {
			defer wg.Done()
			summary, err := j.agentSummarize(ctx, ag, transcript)
			if err != nil {
				return
			}
			mu.Lock()
			summaries[ag.Name()] = summary
			mu.Unlock()
		}(a)
	}
	wg.Wait()

	// Phase 2: Merger combines all summaries
	mergePrompt := j.buildMergePrompt(transcript.Question, summaries)
	messages := []agent.Message{
		{Role: "system", Content: JudgeSystemPrompt + "\n\nYou are merging multiple independent summaries from different agents into one unified document."},
		{Role: "user", Content: mergePrompt},
	}

	return j.merger.StreamGenerate(ctx, messages)
}

func (j *ConsensusJudge) agentSummarize(ctx context.Context, a agent.StreamAgent, transcript *debate.Transcript) (string, error) {
	messages := []agent.Message{
		{Role: "system", Content: "Summarize the key conclusions from this debate. Focus on actionable recommendations and note any unresolved tensions."},
		{Role: "user", Content: BuildJudgePrompt(transcript)},
	}

	resp, err := a.Generate(ctx, messages)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (j *ConsensusJudge) buildMergePrompt(question string, summaries map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Original question: %s\n\n", question))
	sb.WriteString("Here are independent summaries from each agent:\n\n")

	for name, summary := range summaries {
		sb.WriteString(fmt.Sprintf("### %s's Summary\n%s\n\n", name, summary))
	}

	sb.WriteString("Merge these summaries into a single unified brainstorm document, preserving points of agreement and disagreement.")

	return sb.String()
}
