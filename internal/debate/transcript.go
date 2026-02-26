package debate

import "time"

// Transcript holds the complete record of a debate.
type Transcript struct {
	Question  string          `json:"question"`
	Agents    []string        `json:"agents"`
	Rounds    []Round         `json:"rounds"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
}

// Round holds all responses from a single debate round.
type Round struct {
	Number    int                    `json:"number"`
	Phase     string                 `json:"phase"` // "initial", "critique", "defend"
	Responses map[string]RoundEntry  `json:"responses"`
}

// RoundEntry is a single agent's response in a round.
type RoundEntry struct {
	AgentName string `json:"agent_name"`
	Content   string `json:"content"`
	TokensIn  int    `json:"tokens_in"`
	TokensOut int    `json:"tokens_out"`
	Latency   time.Duration `json:"latency"`
}

// NewTranscript creates a new empty transcript.
func NewTranscript(question string, agents []string) *Transcript {
	return &Transcript{
		Question:  question,
		Agents:    agents,
		StartTime: time.Now(),
	}
}

// AddRound appends a completed round to the transcript.
func (t *Transcript) AddRound(round Round) {
	t.Rounds = append(t.Rounds, round)
}

// Finalize marks the transcript as complete.
func (t *Transcript) Finalize() {
	t.EndTime = time.Now()
}

// LastRound returns the most recent round, or nil if none.
func (t *Transcript) LastRound() *Round {
	if len(t.Rounds) == 0 {
		return nil
	}
	return &t.Rounds[len(t.Rounds)-1]
}

// AllResponsesForAgent returns all responses by a given agent across rounds.
func (t *Transcript) AllResponsesForAgent(name string) []RoundEntry {
	var entries []RoundEntry
	for _, r := range t.Rounds {
		if entry, ok := r.Responses[name]; ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// FormatForAgent creates a text summary of all other agents' responses
// in a given round, suitable for inclusion in a critique prompt.
func (t *Transcript) FormatOtherResponses(round int, excludeAgent string) string {
	if round < 1 || round > len(t.Rounds) {
		return ""
	}

	r := t.Rounds[round-1]
	var result string
	for name, entry := range r.Responses {
		if name == excludeAgent {
			continue
		}
		result += "### " + name + "'s Response:\n"
		result += entry.Content + "\n\n"
	}
	return result
}
