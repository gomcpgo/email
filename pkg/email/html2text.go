package email

import (
	"strings"

	"github.com/k3a/html2text"
)

// ConvertHTMLToText converts HTML content to plain text.
// It strips HTML tags and converts entities to their text equivalents.
func ConvertHTMLToText(htmlContent string) (string, error) {
	if htmlContent == "" {
		return "", nil
	}

	text := html2text.HTML2Text(htmlContent)

	// Clean up excessive whitespace while preserving paragraph breaks
	text = cleanupWhitespace(text)

	return text, nil
}

// cleanupWhitespace removes excessive blank lines while preserving structure
func cleanupWhitespace(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			// Allow max 2 consecutive blank lines
			if blankCount <= 2 {
				result = append(result, "")
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}
