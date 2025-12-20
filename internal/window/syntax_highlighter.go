package window

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/PixPMusic/gopher-automate/internal/actions"
)

// SyntaxHighlighter provides basic syntax highlighting for code
type SyntaxHighlighter struct{}

// NewSyntaxHighlighter creates a new syntax highlighter
func NewSyntaxHighlighter() *SyntaxHighlighter {
	return &SyntaxHighlighter{}
}

// HighlightCode returns a RichText widget with syntax-highlighted code
func (h *SyntaxHighlighter) HighlightCode(code string, actionType actions.ActionType) *widget.RichText {
	if code == "" {
		return widget.NewRichText(&widget.TextSegment{
			Text:  "(no code)",
			Style: widget.RichTextStyle{Inline: true, TextStyle: fyne.TextStyle{Italic: true}},
		})
	}

	var segments []widget.RichTextSegment

	switch actionType {
	case actions.ActionTypeAppleScript:
		segments = h.highlightAppleScript(code)
	case actions.ActionTypeShellCommand:
		segments = h.highlightShell(code)
	default:
		segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: code},
		}
	}

	return widget.NewRichText(segments...)
}

// highlightAppleScript applies basic AppleScript syntax highlighting
func (h *SyntaxHighlighter) highlightAppleScript(code string) []widget.RichTextSegment {
	// AppleScript keywords
	keywords := []string{
		"tell", "end tell", "if", "then", "else", "end if",
		"repeat", "end repeat", "set", "to", "get", "return",
		"on", "end", "try", "on error", "end try",
		"true", "false", "application", "property", "handler",
		"with timeout", "giving up after", "as", "of", "the", "a",
		"do shell script", "display dialog", "display notification",
	}

	return h.highlightWithKeywords(code, keywords, "--")
}

// highlightShell applies basic shell syntax highlighting
func (h *SyntaxHighlighter) highlightShell(code string) []widget.RichTextSegment {
	// Shell keywords
	keywords := []string{
		"if", "then", "else", "elif", "fi", "for", "do", "done",
		"while", "until", "case", "esac", "function",
		"return", "exit", "break", "continue",
		"export", "local", "readonly", "declare",
		"echo", "read", "source", "eval",
		"true", "false", "in",
	}

	return h.highlightWithKeywords(code, keywords, "#")
}

// highlightWithKeywords applies highlighting for keywords, strings, comments
func (h *SyntaxHighlighter) highlightWithKeywords(code string, keywords []string, commentPrefix string) []widget.RichTextSegment {
	var segments []widget.RichTextSegment

	// Process line by line to handle comments properly
	lines := strings.Split(code, "\n")

	for lineIdx, line := range lines {
		// Check for comment
		commentStart := strings.Index(line, commentPrefix)
		codePart := line
		commentPart := ""

		if commentStart >= 0 {
			// Check if it's inside a string (simple heuristic)
			beforeComment := line[:commentStart]
			singleQuotes := strings.Count(beforeComment, "'") - strings.Count(beforeComment, "\\'")
			doubleQuotes := strings.Count(beforeComment, "\"") - strings.Count(beforeComment, "\\\"")

			if singleQuotes%2 == 0 && doubleQuotes%2 == 0 {
				codePart = line[:commentStart]
				commentPart = line[commentStart:]
			}
		}

		// Highlight the code part
		if codePart != "" {
			segments = append(segments, h.highlightCodePart(codePart, keywords)...)
		}

		// Add comment in gray/italic
		if commentPart != "" {
			segments = append(segments, &widget.TextSegment{
				Text: commentPart,
				Style: widget.RichTextStyle{
					Inline:    true,
					TextStyle: fyne.TextStyle{Italic: true},
					ColorName: "disabled",
				},
			})
		}

		// Add newline if not last line
		if lineIdx < len(lines)-1 {
			segments = append(segments, &widget.TextSegment{Text: "\n"})
		}
	}

	return segments
}

// highlightCodePart highlights keywords and strings in a code fragment
func (h *SyntaxHighlighter) highlightCodePart(code string, keywords []string) []widget.RichTextSegment {
	var segments []widget.RichTextSegment
	remaining := code

	for len(remaining) > 0 {
		// Check for string literals (start with " or ')
		quote := ""
		if strings.HasPrefix(remaining, "\"") {
			quote = "\""
		} else if strings.HasPrefix(remaining, "'") {
			quote = "'"
		}

		if quote != "" {
			end := h.findStringEnd(remaining[1:], quote)
			if end > 0 {
				length := end + 2 // quote + content + quote
				segments = append(segments, &widget.TextSegment{
					Text: remaining[:length],
					Style: widget.RichTextStyle{
						Inline:    true,
						ColorName: "success",
					},
				})
				remaining = remaining[length:]
				continue
			}
		}

		// Check for keywords
		foundKeyword := false
		for _, kw := range keywords {
			// Regex to match keyword at start with boundary
			// optimization: check prefix first
			if len(remaining) >= len(kw) && strings.EqualFold(remaining[:len(kw)], kw) {
				// check boundary
				if len(remaining) == len(kw) || isBoundary(remaining[len(kw)]) {
					kwLen := len(kw)
					segments = append(segments, &widget.TextSegment{
						Text: remaining[:kwLen],
						Style: widget.RichTextStyle{
							Inline:    true,
							TextStyle: fyne.TextStyle{Bold: true},
							ColorName: "primary",
						},
					})
					remaining = remaining[kwLen:]
					foundKeyword = true
					break
				}
			}
		}
		if foundKeyword {
			continue
		}

		// It's plain text.
		// Consume until next potential keyword start (alpha) or string start (quote)
		// To keep it simple and correct, we can just consume one char, or better, consume run of non-special chars.
		// For now, let's just properly set Inline on the single char or chunk.

		// Optimization: Find next special char
		nextSpecial := -1
		for i, r := range remaining {
			if i == 0 {
				continue
			} // always consume at least 1 or we loop
			if r == '"' || r == '\'' || isBoundary(byte(r)) { // Boundary check is loose here
				nextSpecial = i
				break
			}
		}

		chunk := ""
		if nextSpecial == -1 {
			chunk = remaining
			remaining = ""
		} else {
			// Actually, we should only break on things that COULD start a keyword or string
			// But for now, let's just emit 1 char with Inline=true to be safe and fix the bug.
			// Optimizing to chunk is better but regex matching above expects start of string.
			// If we define "plain" as anything until next whitespace or punctuation...

			// Let's stick to 1-char consumption for safety but set Inline=true
			// To improve perf slightly, verify if it's a letter.
			chunk = string(remaining[0])
			remaining = remaining[1:]
		}

		segments = append(segments, &widget.TextSegment{
			Text: chunk,
			Style: widget.RichTextStyle{
				Inline: true, // This was missing!
			},
		})
	}

	return segments
}

func isBoundary(b uint8) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' ||
		b == '(' || b == ')' || b == '[' || b == ']' || b == '{' || b == '}' ||
		b == ';' || b == ',' || b == '.'
}

// findStringEnd finds the end of a string literal, handling escapes
func (h *SyntaxHighlighter) findStringEnd(s string, quote string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++ // Skip escaped character
			continue
		}
		if string(s[i]) == quote {
			return i
		}
	}
	return -1
}
