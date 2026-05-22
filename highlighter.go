// Syntax highlighting for shell scripts, Python, JSON, YAML and execution logs.
// Produces colored segments for use with RichText.AppendColored.
package main

import (
	"encoding/json"
	"path/filepath"
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
	colorFunc     = walk.RGB(160, 80, 180)    // magenta for functions
)

// GetFileType determines the file type based on extension
func GetFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".sh", ".bash":
		return "sh"
	case ".py":
		return "py"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "txt"
	}
}

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

// FormatCodeByType formats and highlights code based on file type
func FormatCodeByType(rt *RichText, text string, fileType string) {
	if text == "" {
		return
	}
	
	var segs []Segment
	switch fileType {
	case "py":
		segs = HighlightPython(text)
	case "json":
		segs = HighlightJSON(text)
	case "yaml", "yml":
		segs = HighlightYAML(text)
	default:
		segs = HighlightShellScript(text)
	}
	
	for _, seg := range segs {
		if seg.Text == "\n" {
			rt.AppendText("\n")
		} else {
			rt.AppendColored(seg.Text, seg.Color)
		}
	}
}

// FormatCode formats code based on file type (indentation cleanup)
func FormatCode(fileType string, content string) string {
	switch fileType {
	case "json":
		return formatJSON(content)
	case "sh", "bash":
		return formatShell(content)
	case "py":
		return formatPython(content)
	case "yaml", "yml":
		return formatYAML(content)
	default:
		return content
	}
}

// ============================================================
// Python syntax highlighter
// ============================================================

var pythonKeywords = map[string]bool{
	"def": true, "class": true, "import": true, "from": true, "as": true,
	"if": true, "elif": true, "else": true, "while": true, "for": true, "in": true,
	"try": true, "except": true, "finally": true, "with": true,
	"return": true, "yield": true, "lambda": true,
	"pass": true, "break": true, "continue": true,
	"and": true, "or": true, "not": true,
	"True": true, "False": true, "None": true,
	"print": true, "len": true, "range": true,
	"str": true, "int": true, "float": true,
	"list": true, "dict": true, "set": true, "tuple": true,
}

var (
	rePyComment  = regexp.MustCompile(`#[^\n]*`)
	rePyString3  = regexp.MustCompile(`"""[\s\S]*?"""|'''[\s\S]*?'''`)
	rePyString   = regexp.MustCompile(`"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'`)
	rePyNumber   = regexp.MustCompile(`\b\d+\.?\d*(?:[eE][+-]?\d+)?\b`)
	rePyFuncCall = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)(?=\()`)
)

func highlightPythonLine(line string) []Segment {
	if line == "" {
		return nil
	}

	var segs []Segment
	remaining := line
	inSingleQuote := false
	inDoubleQuote := false
	inTripleSingle := false
	inTripleDouble := false
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

		if ch == '\\' && (inDoubleQuote || inSingleQuote) {
			escaped = true
			buf += string(ch)
			continue
		}

		// Check for triple quotes
		if i+2 < len(remaining) {
			triple := remaining[i : i+3]
			if triple == `"""` && !inSingleQuote && !inTripleSingle {
				if inTripleDouble {
					buf += triple
					flush(colorString)
					inTripleDouble = false
				} else {
					flush(colorDefault)
					buf = triple
					inTripleDouble = true
				}
				i += 2
				continue
			}
			if triple == `'''` && !inDoubleQuote && !inTripleDouble {
				if inTripleSingle {
					buf += triple
					flush(colorString)
					inTripleSingle = false
				} else {
					flush(colorDefault)
					buf = triple
					inTripleSingle = true
				}
				i += 2
				continue
			}
		}

		if inTripleSingle || inTripleDouble {
			buf += string(ch)
			continue
		}

		if ch == '\'' && !inDoubleQuote {
			if inSingleQuote {
				buf += string(ch)
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
				buf += string(ch)
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
		if ch == '#' {
			flush(colorDefault)
			buf = remaining[i:]
			flush(colorComment)
			break
		}

		// Word boundary
		if isIdentChar(ch) {
			buf += string(ch)
		} else {
			if buf != "" {
				if pythonKeywords[buf] {
					flush(colorKeyword)
				} else if rePyFuncCall.MatchString(buf + "(") {
					flush(colorFunc)
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

	if buf != "" {
		if pythonKeywords[buf] {
			flush(colorKeyword)
		} else if isNumber(buf) {
			flush(colorNumber)
		} else {
			flush(colorDefault)
		}
	}

	return segs
}

// HighlightPython applies syntax highlighting to Python code
func HighlightPython(script string) []Segment {
	if script == "" {
		return nil
	}

	lines := strings.Split(script, "\n")
	var segs []Segment
	for i, line := range lines {
		lineSegs := highlightPythonLine(line)
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
// JSON syntax highlighter
// ============================================================

// HighlightJSON applies syntax highlighting to JSON content
func HighlightJSON(content string) []Segment {
	if content == "" {
		return nil
	}

	var segs []Segment
	remaining := content
	inString := false
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

		if ch == '\\' && inString {
			escaped = true
			buf += string(ch)
			continue
		}

		if ch == '"' {
			if inString {
				buf += string(ch)
				flush(colorString)
				inString = false
			} else {
				flush(colorDefault)
				buf = string(ch)
				inString = true
			}
			continue
		}

		if inString {
			buf += string(ch)
			continue
		}

		// Keywords and values
		if ch == ':' || ch == ',' || ch == '{' || ch == '}' || ch == '[' || ch == ']' {
			flush(colorDefault)
			segs = append(segs, Segment{Text: string(ch), Color: colorKeyword})
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			flush(colorDefault)
			segs = append(segs, Segment{Text: string(ch), Color: colorDefault})
			continue
		}

		// Collect word
		buf += string(ch)
		if i+1 >= len(remaining) || remaining[i+1] == ' ' || remaining[i+1] == '\t' ||
			remaining[i+1] == '\n' || remaining[i+1] == ':' || remaining[i+1] == ',' ||
			remaining[i+1] == '}' || remaining[i+1] == ']' {
			word := strings.TrimSpace(buf)
			if word != "" {
				if word == "true" || word == "false" || word == "null" {
					flush(colorKeyword)
				} else if isNumber(word) {
					flush(colorNumber)
				} else {
					flush(colorDefault)
				}
			}
			buf = ""
		}
	}

	if buf != "" {
		flush(colorDefault)
	}

	return segs
}

// ============================================================
// YAML syntax highlighter
// ============================================================

// HighlightYAML applies syntax highlighting to YAML content
func HighlightYAML(content string) []Segment {
	if content == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	var segs []Segment

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		
		// Check for comment
		if strings.HasPrefix(trimmed, "#") {
			segs = append(segs, Segment{Text: line, Color: colorComment})
		} else if strings.Contains(line, ":") {
			// Key-value line
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				// Key part (including leading spaces for indentation visualization)
				keyWithSpace := parts[0]
				colonPos := strings.Index(keyWithSpace, ":")
				if colonPos > 0 {
					indent := keyWithSpace[:colonPos-len(strings.TrimLeft(keyWithSpace, " \t"))]
					key := strings.TrimSpace(keyWithSpace)
					value := parts[1]
					
					if indent != "" {
						segs = append(segs, Segment{Text: indent, Color: colorDefault})
					}
					segs = append(segs, Segment{Text: key + ":", Color: colorKeyword})
					
					// Value part
					if strings.TrimSpace(value) != "" {
						if strings.HasPrefix(strings.TrimSpace(value), "\"") || 
						   strings.HasPrefix(strings.TrimSpace(value), "'") {
							segs = append(segs, Segment{Text: value, Color: colorString})
						} else if isNumber(strings.TrimSpace(value)) {
							segs = append(segs, Segment{Text: value, Color: colorNumber})
						} else if strings.TrimSpace(value) == "true" || 
								  strings.TrimSpace(value) == "false" ||
								  strings.TrimSpace(value) == "null" ||
								  strings.TrimSpace(value) == "~" {
							segs = append(segs, Segment{Text: value, Color: colorKeyword})
						} else {
							segs = append(segs, Segment{Text: value, Color: colorDefault})
						}
					}
				} else {
					segs = append(segs, Segment{Text: line, Color: colorDefault})
				}
			} else {
				segs = append(segs, Segment{Text: line, Color: colorDefault})
			}
		} else {
			segs = append(segs, Segment{Text: line, Color: colorDefault})
		}
		
		if i < len(lines)-1 {
			segs = append(segs, Segment{Text: "\n", Color: colorDefault})
		}
	}

	return segs
}

// ============================================================
// Code formatting helpers
// ============================================================

func formatJSON(content string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return content // Return original if invalid JSON
	}
	formatted, _ := json.MarshalIndent(obj, "", "  ")
	return string(formatted)
}

func formatShell(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	indentLevel := 0
	baseIndent := "  "

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// Decrease indent for closing keywords
		if strings.HasPrefix(trimmed, "done") || strings.HasPrefix(trimmed, "fi") ||
			strings.HasPrefix(trimmed, "esac") || strings.HasPrefix(trimmed, "}") {
			indentLevel--
			if indentLevel < 0 {
				indentLevel = 0
			}
		}

		result = append(result, strings.Repeat(baseIndent, indentLevel)+trimmed)

		// Increase indent for opening keywords
		if strings.HasSuffix(trimmed, "then") || strings.HasSuffix(trimmed, "do") ||
			strings.HasSuffix(trimmed, "else") || strings.HasSuffix(trimmed, "{") ||
			strings.HasPrefix(trimmed, "case ") {
			indentLevel++
		}
	}
	return strings.Join(result, "\n")
}

func formatPython(content string) string {
	// Basic Python formatting: clean up extra blank lines while preserving indentation
	lines := strings.Split(content, "\n")
	var result []string
	prevEmpty := false

	for _, line := range lines {
		isEmpty := len(strings.TrimSpace(line)) == 0
		if isEmpty {
			if !prevEmpty {
				result = append(result, "")
			}
			prevEmpty = true
		} else {
			result = append(result, line) // Preserve original indentation
			prevEmpty = false
		}
	}
	return strings.Join(result, "\n")
}

func formatYAML(content string) string {
	// YAML formatting is complex; return as-is to preserve indentation
	return content
}
