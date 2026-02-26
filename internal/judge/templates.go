package judge

import (
	"fmt"

	"github.com/aadidev/debategpt/internal/debate"
)

// JudgeSystemPrompt is the system prompt for the judge.
const JudgeSystemPrompt = `You are the judge in a multi-agent debate. Your role is to synthesize multiple agents' perspectives into a single, actionable brainstorm document.

You must:
- Identify the strongest arguments from each agent
- Note genuine points of consensus (high-confidence recommendations)
- Preserve unresolved disagreements WITHOUT picking a side — present both arguments
- Produce concrete, actionable recommendations
- Be specific and avoid generic advice

Do NOT favor any single agent. Your synthesis must fairly represent all perspectives.`

// BuildJudgePrompt creates the user prompt for the judge with the full transcript.
func BuildJudgePrompt(transcript *debate.Transcript) string {
	prompt := fmt.Sprintf("# Debate Transcript\n\n**Question:** %s\n\n", transcript.Question)

	for _, round := range transcript.Rounds {
		prompt += fmt.Sprintf("---\n\n## Round %d: %s\n\n", round.Number, round.Phase)
		for name, entry := range round.Responses {
			prompt += fmt.Sprintf("### %s\n\n%s\n\n", name, entry.Content)
		}
	}

	prompt += `---

Now synthesize all of the above into a structured brainstorm document. Use EXACTLY this format:

## Executive Summary
One paragraph synthesizing the best answer.

## Recommended Approach
The approach you believe is strongest, with justification from the debate.

## Key Decision Points
For each major decision, list the options discussed with pros/cons.

## Architecture Recommendations
Concrete technical recommendations if applicable.

## Risks & Mitigations
Table format: Risk | Raised By | Mitigation

## Points of Consensus
What all agents agreed on — these are high-confidence recommendations.

## Unresolved Disagreements
Where agents couldn't agree — present both sides fairly so the developer can decide.

## Action Items
Concrete next steps as a checklist.`

	return prompt
}
