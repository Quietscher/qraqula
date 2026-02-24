package highlight

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// vampireStyle defines the color theme matching qraqula's aesthetic.
var vampireStyle = styles.Register(chroma.MustNewStyle("vampire", chroma.StyleEntries{
	// Keywords — bold red (query, mutation, subscription, true, false, null)
	chroma.Keyword:            "#ff0000 bold",
	chroma.KeywordDeclaration: "#ff0000 bold",
	chroma.KeywordType:        "#ff0000",
	chroma.KeywordConstant:    "#5f5fd7", // true/false/null → purple

	// Names
	chroma.NameBuiltin:   "#ff0000",
	chroma.NameTag:       "#ff0000", // JSON keys
	chroma.NameAttribute: "#ffffff", // HTML/XML attributes
	chroma.NameProperty:  "#ffffff", // GraphQL field names
	chroma.NameVariable:  "#ff5f5f", // GraphQL $variables → light red
	chroma.NameClass:     "#5f87d7", // GraphQL type names → blue

	// Literals
	chroma.LiteralString:       "#bcbcbc",
	chroma.LiteralStringDouble: "#bcbcbc",
	chroma.LiteralNumber:       "#5f5fd7", // purple

	// Syntax
	chroma.Punctuation: "#8a8a8a",
	chroma.Operator:    "#8a8a8a",

	// Comments
	chroma.Comment:       "#626262",
	chroma.CommentSingle: "#626262",

	chroma.GenericError: "#ff0000 bold",
	chroma.Background:   "bg:",
}))

// Colorize returns the source string with ANSI escape codes for syntax highlighting.
func Colorize(src string, lexerName string) string {
	if src == "" {
		return ""
	}

	lexer := lexers.Get(lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal256")
	style := styles.Get("vampire")

	iterator, err := lexer.Tokenise(nil, src)
	if err != nil {
		return src
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return src
	}

	result := buf.String()
	if !strings.HasSuffix(src, "\n") {
		result = strings.TrimRight(result, "\n")
	}

	return result
}

// ColorizeLines highlights the source and splits into lines.
func ColorizeLines(src string, lexerName string) []string {
	if src == "" {
		return []string{""}
	}

	highlighted := Colorize(src, lexerName)
	return strings.Split(highlighted, "\n")
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes all ANSI escape codes from a string.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
