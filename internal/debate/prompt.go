package debate

import "fmt"

// PromptBuilder constructs prompts for each debate phase.
type PromptBuilder struct {
	perspectives map[string]string // agent name → perspective/role prompt
	constraints  []string
	context      string
}

// NewPromptBuilder creates a new prompt builder.
func NewPromptBuilder(perspectives map[string]string, constraints []string, context string) *PromptBuilder {
	return &PromptBuilder{
		perspectives: perspectives,
		constraints:  constraints,
		context:      context,
	}
}

// SystemPrompt builds the system prompt for an agent based on its assigned perspective.
func (pb *PromptBuilder) SystemPrompt(agentName string) string {
	perspective := pb.perspectives[agentName]
	if perspective == "" {
		perspective = "You are a thoughtful software engineering expert. Provide detailed, well-reasoned analysis."
	}

	prompt := fmt.Sprintf(`You are participating in a multi-agent debate about a software engineering question.

Your assigned perspective: %s

You must defend your perspective even when other agents disagree. Only concede a point if presented with a genuinely compelling argument you cannot counter. Be specific, cite trade-offs, and avoid vague generalities.`, perspective)

	if pb.context != "" {
		prompt += fmt.Sprintf("\n\nProject context:\n%s", pb.context)
	}

	if len(pb.constraints) > 0 {
		prompt += "\n\nConstraints that must be respected:"
		for _, c := range pb.constraints {
			prompt += "\n- " + c
		}
	}

	return prompt
}

// InitialPrompt builds the prompt for Round 1 (open brainstorm).
func (pb *PromptBuilder) InitialPrompt(question string) string {
	return fmt.Sprintf(`Question: %s

Provide a thorough, well-structured answer from your assigned perspective. Include:
- Your recommended approach
- Key trade-offs and considerations
- Potential risks
- Concrete next steps

Be specific and opinionated — generic answers are not useful.`, question)
}

// CritiquePrompt builds the prompt for critique rounds.
func (pb *PromptBuilder) CritiquePrompt(question string, otherResponses string) string {
	return fmt.Sprintf(`The original question was: %s

Here are the other agents' responses from the previous round:

%s

Now critique these responses from your perspective. You MUST use this exact structure:

## Agreements
Specific points you agree with from other agents, and WHY you agree.

## Disagreements
Specific points you disagree with, with detailed counterarguments. Do not be polite — be rigorous.

## Blind Spots
Important considerations that NO agent has mentioned yet.

## Revised Position
Your updated stance on the original question, incorporating any valid critiques while defending your core perspective.`, question, otherResponses)
}

// DefendPrompt builds the prompt for defend-or-concede rounds.
func (pb *PromptBuilder) DefendPrompt(question string, otherResponses string) string {
	return fmt.Sprintf(`The original question was: %s

Here are the other agents' critiques from the previous round:

%s

This is the final round. You must be explicit:

## Points I've Changed My Mind On
List specific points where other agents convinced you, and explain why.

## Points I'm Doubling Down On
List specific points you still defend despite criticism, and provide your strongest argument.

## Final Recommendation
Your definitive recommendation for the original question, incorporating everything discussed.`, question, otherResponses)
}

// DisagreementInjection returns a meta-prompt to inject when agents are converging too fast.
func (pb *PromptBuilder) DisagreementInjection() string {
	return `

[MODERATOR NOTE: The agents are converging too quickly. Please reconsider whether there are:
- Alternative approaches not yet explored
- Hidden risks that haven't been discussed
- Trade-offs that are being glossed over
- Unconventional solutions worth considering
Push back harder on the consensus. The value of this debate comes from genuine disagreement.]`
}
