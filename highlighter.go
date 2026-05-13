// Syntax highlighting for shell scripts and execution logs.
// Produces colored segments for use with RichText.AppendColored.
package main

import (
	"regexp"
	"strings"

	"github.com/lxn/walk"
)

// Color constants used for highlighting
var (
	// Log output colors
	colorTimestamp = walk.RGB(128, 128, 128)  // gray
	colorHeader    = walk.RGB(0, 180, 200)     // cyan
	colorSuccess   = walk.RGB(0, 180, 80)      // green
	colorError     = walk.RGB(220, 50, 50)      // red
	colorWarning   = walk.RGB(220, 160, 0)     // amber
	colorStderr    = walk.RGB(220, 80, 80)      // light red
	colorInfo      = walk.RGB(60, 140, 220)     // blue
	colorSep       = walk.RGB(120, 120, 120)    // gray separator
	colorBold      = walk.RGB(220, 220, 220)    // near-white
	colorDefault   = walk.RGB(200, 200, 200)    // light gray

	// Script syntax colors
	colorComment  = walk.RGB(100, 180, 100)   // green
	colorKeyword  = walk.RGB(80, 140, 220)    // blue
	colorString   = walk.RGB(200, 140, 60)    // orange
	colorVariable = walk.RGB(180, 120, 200)   // purple
	colorNumber   = walk.RGB(200, 140, 60)    // orange
)

// Segment is a span of text with a single color.
type Segment struct {
	Text  string
	Color walk.Color
}

// ============================================================
// Log output highlighter
// ============================================================

// LogPrefixStyle describes how to color a line based on its prefix/content.
type LogPrefixStyle struct {
	Prefix   string
	Color    walk.Color
	Bold     bool
	MatchAll bool // if true, prefix match checks anywhere in line
}

var logStyles = []LogPrefixStyle{
	{Prefix: "=====", Color: colorSep},
	{Prefix: "▶ ", Color: colorHeader, Bold: true},
	{Prefix: "✅ ", Color: colorSuccess},
	{Prefix: "❌ ", Color: colorError},
	{Prefix: "⚠️ ", Color: colorWarning},
	{Prefix: "--- STDERR ---", Color: colorStderr},
	{Prefix: "⏹ ", Color: colorWarning},
	{Prefix: "→ ", Color: colorInfo},
}

// HighlightLogLine returns the color for a log line based on its content.
func HighlightLogLine(line string) walk.Color {
	for _, s := range logStyles {
		if strings.HasPrefix(line, s.Prefix) {
			return s.Color
		}
	}
	// Check for error/warning keywords in the line
	lower := strings.ToLower(line)
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") ||
		strings.Contains(lower, "exception") || strings.Contains(lower, "fatal") {
		return colorError
	}
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		return colorWarning
	}
	if strings.Contains(lower, "success") || strings.Contains(lower, "completed") ||
		strings.Contains(lower, "done") || strings.Contains(lower, "ok") {
		return colorSuccess
	}
	return colorDefault
}

// HighlightLogText splits text into colored segments based on line content.
func HighlightLogText(text string) []Segment {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	// Don't highlight trailing empty line
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var segs []Segment
	for i, line := range lines {
		color := HighlightLogLine(line)
		segs = append(segs, Segment{Text: line, Color: color})
		if i < len(lines)-1 {
			segs = append(segs, Segment{Text: "\n", Color: colorDefault})
		}
	}
	return segs
}

// ============================================================
// Shell script syntax highlighter
// ============================================================

// Shell keywords
var shellKeywords = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"for": true, "while": true, "until": true, "do": true, "done": true,
	"case": true, "esac": true, "in": true,
	"function": true, "return": true, "exit": true,
	"local": true, "export": true, "source": true, "unset": true,
	"select": true, "break": true, "continue": true,
	"echo": true, "printf": true, "read": true,
	"exec": true, "eval": true, "set": true, "shift": true,
	"trap": true, "type": true, "command": true,
	"cd": true, "pwd": true, "ls": true, "cp": true, "mv": true, "rm": true,
	"mkdir": true, "touch": true, "chmod": true, "chown": true,
	"cat": true, "grep": true, "sed": true, "awk": true,
	"cut": true, "sort": true, "uniq": true, "wc": true,
	"head": true, "tail": true, "less": true, "more": true,
	"find": true, "xargs": true, "tee": true, "tr": true,
	"ssh": true, "scp": true, "rsync": true,
	"systemctl": true, "service": true, "journalctl": true,
	"docker": true, "docker-compose": true,
	"sudo": true, "su": true, "whoami": true, "id": true,
	"date": true, "tar": true, "gzip": true, "gunzip": true,
	"yum": true, "apt-get": true, "apt": true, "pip": true, "npm": true,
	"ps": true, "kill": true, "nohup": true, "disown": true,
	"crontab": true, "alias": true,
}

var (
	reComment   = regexp.MustCompile(`(^|\s+)(#[^\n]*)`)
	reVariable  = regexp.MustCompile(`\$\{?[a-zA-Z_][a-zA-Z0-9_]*\}?`)
	reDollarVar = regexp.MustCompile(`\$[a-zA-Z_][a-zA-Z0-9_]*`)
	reNumber    = regexp.MustCompile(`\b[0-9]+\b`)
)

// highlightShellLine applies syntax highlighting to a single line.
// This is a simplified line-by-line approach (not fully accurate for multi-line strings).
func highlightShellLine(line string) []Segment {
	if line == "" {
		return nil
	}

	var segs []Segment
	remaining := line

	// Process the line character by character
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	buf := ""

	flush := func(color walk.Color) {
		if buf != "" {
			segs = append(segs, Segment{Text: buf, Color: color})
			buf = ""
		}
	}

	for i := 0; i < len(remaining); i++ {
		ch := remaining[i]

		if escaped {
			escaped = false
			buf += string(ch)
			continue
		}

		if ch == '\\' && inDoubleQuote {
			escaped = true
			buf += string(ch)
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			if inSingleQuote {
				flush(colorString)
				inSingleQuote = false
				continue
			} else {
				flush(colorDefault)
				inSingleQuote = true
				buf += string(ch)
				continue
			}
		}

		if ch == '"' && !inSingleQuote {
			if inDoubleQuote {
				flush(colorString)
				inDoubleQuote = false
				continue
			} else {
				flush(colorDefault)
				inDoubleQuote = true
				buf += string(ch)
				continue
			}
		}

		if inSingleQuote || inDoubleQuote {
			buf += string(ch)
			continue
		}

		// Comment
		if ch == '#' && (i == 0 || remaining[i-1] == ' ' || remaining[i-1] == '\t') {
			flush(colorDefault)
			buf = remaining[i:]
			flush(colorComment)
			break
		}

		// Variable $VAR or ${VAR}
		if ch == '$' && i+1 < len(remaining) {
			flush(colorDefault)
			buf = string(ch)
			j := i + 1
			if remaining[j] == '{' {
				// ${VAR}
				buf += string(remaining[j])
				j++
				for j < len(remaining) && remaining[j] != '}' {
					buf += string(remaining[j])
					j++
				}
				if j < len(remaining) {
					buf += string(remaining[j])
				}
				flush(colorVariable)
				i = j
				continue
			} else if isIdentChar(remaining[j]) {
				for j < len(remaining) && isIdentChar(remaining[j]) {
					buf += string(remaining[j])
					j++
				}
				flush(colorVariable)
				i = j - 1
				continue
			}
		}

		// Word boundary check for keywords and numbers
		if isIdentChar(ch) {
			buf += string(ch)
		} else {
			if buf != "" {
				// Check if buf is a keyword
				if shellKeywords[buf] {
					flush(colorKeyword)
				} else if isNumber(buf) {
					flush(colorNumber)
				} else {
					flush(colorDefault)
				}
			}
			buf = string(ch)
			flush(colorDefault)
		}
	}

	// Flush remaining buffer
	if buf != "" {
		if shellKeywords[buf] {
			flush(colorKeyword)
		} else if isNumber(buf) {
			flush(colorNumber)
		} else if strings.HasPrefix(buf, "#") {
			flush(colorComment)
		} else {
			flush(colorDefault)
		}
	}

	return segs
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}

func isNumber(s string) bool {
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return len(s) > 0
}

// HighlightShellScript applies syntax highlighting to shell script content.
func HighlightShellScript(script string) []Segment {
	if script == "" {
		return nil
	}

	lines := strings.Split(script, "\n")
	var segs []Segment
	for i, line := range lines {
		lineSegs := highlightShellLine(line)
		if len(lineSegs) == 0 {
			segs = append(segs, Segment{Text: "", Color: colorDefault})
		} else {
			segs = append(segs, lineSegs...)
		}
		if i < len(lines)-1 {
			segs = append(segs, Segment{Text: "\n", Color: colorDefault})
		}
	}
	return segs
}

// ============================================================
// Convenience formatting helpers
// ============================================================

// FormatLogOutput takes raw execution output and writes it to a RichText
// with appropriate coloring.
func FormatLogOutput(rt *RichText, text string) {
	if text == "" {
		return
	}
	segs := HighlightLogText(text)
	for _, seg := range segs {
		if seg.Text == "\n" {
			rt.AppendText("\n")
		} else {
			rt.AppendColored(seg.Text, seg.Color)
		}
	}
}

// FormatScriptContent takes shell script content and writes it to a RichText
// with syntax highlighting.
func FormatScriptContent(rt *RichText, text string) {
	if text == "" {
		return
	}
	segs := HighlightShellScript(text)
	for _, seg := range segs {
		if seg.Text == "\n" {
			rt.AppendText("\n")
		} else {
			rt.AppendColored(seg.Text, seg.Color)
		}
	}
}
