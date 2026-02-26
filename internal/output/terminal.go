package output

import (
	"fmt"

	"github.com/charmbracelet/glamour"
)

// RenderTerminal renders markdown content for terminal display.
func RenderTerminal(markdown string) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return "", fmt.Errorf("create renderer: %w", err)
	}

	rendered, err := renderer.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}

	return rendered, nil
}
