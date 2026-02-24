package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// JSON formats JSON input with standard 2-space indentation.
// Returns the original string and an error if the input is invalid JSON.
func JSON(src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return src, nil
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(src), "", "  "); err != nil {
		return src, err
	}
	return buf.String(), nil
}

// ValidateJSON checks whether the input is valid JSON.
func ValidateJSON(src string) error {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	var v any
	return json.Unmarshal([]byte(src), &v)
}

// GraphQL formats a GraphQL query/mutation with standard 2-space indentation.
func GraphQL(src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return src
	}

	tokens := tokenizeGQL(src)
	if len(tokens) == 0 {
		return src
	}

	var buf strings.Builder
	indent := 0
	parenDepth := 0

	for i, tok := range tokens {
		next := ""
		if i+1 < len(tokens) {
			next = tokens[i+1]
		}
		prev := ""
		if i > 0 {
			prev = tokens[i-1]
		}

		switch {
		case tok == "{":
			if buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString("{\n")
			indent++
			writeIndent(&buf, indent)

		case tok == "}":
			indent--
			if indent < 0 {
				indent = 0
			}
			buf.WriteByte('\n')
			writeIndent(&buf, indent)
			buf.WriteByte('}')
			if next != "" && next != "}" {
				buf.WriteByte('\n')
				writeIndent(&buf, indent)
			}

		case tok == "(":
			parenDepth++
			buf.WriteByte('(')

		case tok == ")":
			if parenDepth > 0 {
				parenDepth--
			}
			buf.WriteByte(')')

		case tok == ":":
			buf.WriteString(": ")

		case tok == "!" || tok == "[" || tok == "]" || tok == "=":
			buf.WriteString(tok)

		default: // words, $, @, ..., strings, comments
			if parenDepth > 0 {
				// Inside arguments — inline with spaces
				if prev != "(" && prev != ":" && prev != "$" && prev != "@" && prev != "[" {
					buf.WriteByte(' ')
				}
				buf.WriteString(tok)
			} else {
				// Field level
				buf.WriteString(tok)
				if next == "{" || next == "(" || next == ":" || next == "!" || next == "[" || next == "]" {
					// Next token attaches to this one
				} else if next == "}" || next == "" {
					// End of block or input
				} else if keepsNextInline(tok) {
					buf.WriteByte(' ')
				} else {
					buf.WriteByte('\n')
					writeIndent(&buf, indent)
				}
			}
		}
	}

	return strings.TrimRight(buf.String(), "\n ")
}

// ValidateGraphQL checks for balanced braces and parentheses.
func ValidateGraphQL(src string) error {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}

	braces := 0
	parens := 0
	inString := false

	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			if c == '"' && (i == 0 || src[i-1] != '\\') {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			braces++
		case '}':
			braces--
			if braces < 0 {
				return fmt.Errorf("unexpected '}' at position %d", i)
			}
		case '(':
			parens++
		case ')':
			parens--
			if parens < 0 {
				return fmt.Errorf("unexpected ')' at position %d", i)
			}
		}
	}

	if braces > 0 {
		return fmt.Errorf("unclosed '{' (%d open)", braces)
	}
	if parens > 0 {
		return fmt.Errorf("unclosed '(' (%d open)", parens)
	}
	if inString {
		return fmt.Errorf("unclosed string")
	}
	return nil
}

func writeIndent(buf *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		buf.WriteString("  ")
	}
}

func keepsNextInline(tok string) bool {
	switch tok {
	case "query", "mutation", "subscription", "fragment", "on", "...", "@":
		return true
	}
	return false
}

func tokenizeGQL(src string) []string {
	var tokens []string
	i := 0
	for i < len(src) {
		c := src[i]

		// Skip whitespace and commas
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',' {
			i++
			continue
		}

		// Single-char tokens
		if c == '{' || c == '}' || c == '(' || c == ')' || c == ':' ||
			c == '!' || c == '$' || c == '@' || c == '[' || c == ']' || c == '=' {
			tokens = append(tokens, string(c))
			i++
			continue
		}

		// Spread
		if c == '.' && i+2 < len(src) && src[i+1] == '.' && src[i+2] == '.' {
			tokens = append(tokens, "...")
			i += 3
			continue
		}

		// String
		if c == '"' {
			j := i + 1
			// Block string """
			if j+1 < len(src) && src[j] == '"' && src[j+1] == '"' {
				j += 2
				for j < len(src) {
					if j+2 < len(src) && src[j] == '"' && src[j+1] == '"' && src[j+2] == '"' {
						j += 3
						break
					}
					j++
				}
			} else {
				for j < len(src) && src[j] != '"' {
					if src[j] == '\\' {
						j++
					}
					j++
				}
				if j < len(src) {
					j++
				}
			}
			tokens = append(tokens, src[i:j])
			i = j
			continue
		}

		// Comment
		if c == '#' {
			j := i
			for j < len(src) && src[j] != '\n' {
				j++
			}
			tokens = append(tokens, src[i:j])
			i = j
			continue
		}

		// Word (identifier, number, etc.)
		j := i
		for j < len(src) && !isGQLSep(src[j]) {
			j++
		}
		if j > i {
			tokens = append(tokens, src[i:j])
			i = j
		} else {
			i++
		}
	}
	return tokens
}

func isGQLSep(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', ',',
		'{', '}', '(', ')', '[', ']',
		':', '!', '$', '@', '=', '"', '#', '.':
		return true
	}
	return false
}
