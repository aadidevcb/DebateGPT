package output

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// WriteFile saves content to a file, creating directories as needed.
func WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}

	return nil
}

// GenerateFilename creates a filename from the question.
func GenerateFilename(question string, directory string) string {
	// Sanitize the question for a filename
	re := regexp.MustCompile(`[^a-zA-Z0-9-_]+`)
	slug := re.ReplaceAllString(strings.ToLower(question), "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	slug = strings.Trim(slug, "-")

	timestamp := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("%s-%s.md", slug, timestamp)

	return filepath.Join(directory, filename)
}
