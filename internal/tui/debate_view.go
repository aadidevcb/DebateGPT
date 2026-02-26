package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aadidev/debategpt/internal/agent"
	"github.com/aadidev/debategpt/internal/stream"
	"github.com/charmbracelet/lipgloss"
)

// DebateView renders the live debate state in the terminal.
type DebateView struct {
	mu           sync.Mutex
	question     string
	agentOrder   []string
	agentRoles   map[string]string
	currentRound int
	totalRounds  int

	// Per-agent streaming state
	agentBuffers  map[string]*strings.Builder
	agentStatus   map[string]string // "streaming", "done", "waiting", "error"
	agentMetrics  map[string]*agent.Metrics

	// Cost state
	totalCost float64
	budget    float64
}

// NewDebateView creates a new debate view.
func NewDebateView(question string, agentOrder []string, agentRoles map[string]string, totalRounds int, budget float64) *DebateView {
	buffers := make(map[string]*strings.Builder)
	status := make(map[string]string)
	metrics := make(map[string]*agent.Metrics)

	for _, name := range agentOrder {
		buffers[name] = &strings.Builder{}
		status[name] = "waiting"
	}

	return &DebateView{
		question:     question,
		agentOrder:   agentOrder,
		agentRoles:   agentRoles,
		totalRounds:  totalRounds,
		agentBuffers: buffers,
		agentStatus:  status,
		agentMetrics: metrics,
		budget:       budget,
	}
}

// SetRound updates the current round number and resets agent buffers.
func (v *DebateView) SetRound(round int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.currentRound = round
	for _, name := range v.agentOrder {
		v.agentBuffers[name] = &strings.Builder{}
		v.agentStatus[name] = "waiting"
		v.agentMetrics[name] = nil
	}
}

// ProcessEvent handles a multiplexed stream event.
func (v *DebateView) ProcessEvent(event stream.MuxEvent) {
	v.mu.Lock()
	defer v.mu.Unlock()

	switch event.Event.Type {
	case agent.EventDelta:
		if buf, ok := v.agentBuffers[event.AgentName]; ok {
			buf.WriteString(event.Event.Delta)
		}
		v.agentStatus[event.AgentName] = "streaming"
	case agent.EventDone:
		v.agentStatus[event.AgentName] = "done"
		v.agentMetrics[event.AgentName] = event.Event.Metrics
	case agent.EventError:
		v.agentStatus[event.AgentName] = "error"
	}
}

// SetCost updates the running cost total.
func (v *DebateView) SetCost(cost float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.totalCost = cost
}

// Render produces the current terminal output.
func (v *DebateView) Render() string {
	v.mu.Lock()
	defer v.mu.Unlock()

	var sb strings.Builder

	// Question header
	sb.WriteString(TitleStyle.Render("🎯 "+v.question) + "\n")
	sb.WriteString(strings.Repeat("━", 60) + "\n\n")

	// Round header
	phaseName := v.phaseName()
	sb.WriteString(RoundHeaderStyle.Render(
		fmt.Sprintf("Round %d of %d: %s", v.currentRound, v.totalRounds, phaseName),
	) + "\n\n")

	// Agent panels
	for i, name := range v.agentOrder {
		icon := GetAgentIcon(i)
		color := GetAgentColor(i)
		role := v.agentRoles[name]
		if role == "" {
			role = name
		}

		nameStyle := AgentNameStyle.Foreground(color)
		sb.WriteString(fmt.Sprintf("  %s %s", icon, nameStyle.Render(fmt.Sprintf("%s (%s)", name, role))))

		status := v.agentStatus[name]
		switch status {
		case "streaming":
			sb.WriteString(StreamingStyle.Render(" ▍streaming..."))
		case "done":
			metrics := v.agentMetrics[name]
			if metrics != nil {
				sb.WriteString(SuccessStyle.Render(
					fmt.Sprintf(" ✓ done (%.1fs, %d tokens)", metrics.Latency.Seconds(), metrics.TokensOut),
				))
			} else {
				sb.WriteString(SuccessStyle.Render(" ✓ done"))
			}
		case "error":
			sb.WriteString(ErrorStyle.Render(" ✗ error"))
		default:
			sb.WriteString(StreamingStyle.Render(" ⏳ waiting..."))
		}
		sb.WriteString("\n")

		// Show preview of content (last 2 lines)
		if buf, ok := v.agentBuffers[name]; ok {
			content := buf.String()
			if content != "" {
				preview := v.truncateToLines(content, 3)
				previewStyle := lipgloss.NewStyle().Foreground(mutedColor).PaddingLeft(4)
				sb.WriteString(previewStyle.Render("┊ "+strings.ReplaceAll(preview, "\n", "\n  ┊ ")) + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Status bar
	costStr := fmt.Sprintf("💰 $%.3f", v.totalCost)
	if v.budget > 0 {
		costStr += fmt.Sprintf(" / $%.2f", v.budget)
	}
	roundStr := fmt.Sprintf("⏱ Round %d of %d", v.currentRound, v.totalRounds)
	agentStr := fmt.Sprintf("👥 %d agents", len(v.agentOrder))

	sb.WriteString(StatusBarStyle.Render(
		fmt.Sprintf("%s  │  %s  │  %s", costStr, roundStr, agentStr),
	))

	return sb.String()
}

func (v *DebateView) phaseName() string {
	switch v.currentRound {
	case 1:
		return "Initial Responses"
	case v.totalRounds:
		return "Defend or Concede"
	default:
		return "Critique"
	}
}

func (v *DebateView) truncateToLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[len(lines)-maxLines:], "\n")
}
