package judge

import (
	"context"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/debate"
)

// Judge synthesizes a final document from a debate transcript.
type Judge interface {
	Mode() string
	Synthesize(ctx context.Context, transcript *debate.Transcript) (<-chan agent.StreamEvent, error)
}

// NewJudge creates the appropriate judge based on mode.
func NewJudge(mode string, agents map[string]agent.StreamAgent, agentOrder []string, judgeModel agent.StreamAgent, participantName string) Judge {
	switch mode {
	case "participant":
		if a, ok := agents[participantName]; ok {
			return &ParticipantJudge{agent: a, name: participantName}
		}
		// Fallback to first agent
		return &ParticipantJudge{agent: agents[agentOrder[0]], name: agentOrder[0]}
	case "consensus":
		agentList := make([]agent.StreamAgent, 0, len(agentOrder))
		for _, name := range agentOrder {
			agentList = append(agentList, agents[name])
		}
		merger := judgeModel
		if merger == nil {
			merger = agents[agentOrder[0]]
		}
		return &ConsensusJudge{agents: agentList, merger: merger}
	default: // "separate"
		if judgeModel != nil {
			return &SeparateJudge{agent: judgeModel}
		}
		return &SeparateJudge{agent: agents[agentOrder[0]]}
	}
}
