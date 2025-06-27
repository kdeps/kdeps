package utils

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/kaptinlin/jsonrepair"
)

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func FixJSON(input string) string {
	// 1. Remove surrounding quotes if present (common when reading from CLI args)
	if strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\"") {
		if unquoted, err := strconv.Unquote(input); err == nil {
			input = unquoted
		}
	}

	// 2. Fast-path: if the string is already valid JSON leave it untouched.
	if IsJSON(input) {
		return input
	}

	// 3. Stream through the bytes and repair string tokens.
	//    Rules applied while we are INSIDE a string literal:
	//      • unescaped `"`  → `\"`
	//      • raw newlines    → `\n`
	//      • raw carriage return → `\r`
	//    All other bytes are copied verbatim.

	var b strings.Builder
	inString := false   // Currently inside a JSON string literal
	escapeNext := false // The previous byte was a backslash

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if inString {
			if escapeNext {
				// Previous character was a backslash – copy current byte verbatim.
				b.WriteByte(ch)
				escapeNext = false
				continue
			}

			switch ch {
			case '\\':
				// If the next character is a quote, duplicate the backslash to ensure it is escaped once.
				if i+1 < len(input) && input[i+1] == '"' {
					b.WriteString("\\\\") // write two backslashes
				} else {
					b.WriteByte(ch)
				}
				escapeNext = true
			case '"':
				// Determine if this quote should terminate the string or be escaped.
				// If the following rune indicates end of value (comma, brace, bracket, whitespace, newline),
				// treat as terminator; otherwise treat as interior quote and escape it.
				isEnd := false
				if j := i + 1; j >= len(input) {
					isEnd = true
				} else {
					next := input[i+1]
					switch next {
					case ' ', '\t', '\r', '\n', ',', ':', '}', ']':
						isEnd = true
					}
				}
				if isEnd {
					b.WriteByte(ch)
					inString = false
				} else {
					// interior quote – escape it
					b.WriteByte('\\')
					b.WriteByte('"')
				}
			case '\n':
				b.WriteString("\\n")
			case '\r':
				b.WriteString("\\r")
			default:
				b.WriteByte(ch)
			}
			continue
		}

		// Currently NOT in a string literal.
		if ch == '"' {
			inString = true
		}
		b.WriteByte(ch)
	}

	fixed := b.String()

	// 4. Strip leading indentation spaces to normalise formatting.
	fixed = regexp.MustCompile(`(?m)^\s+`).ReplaceAllString(fixed, "")

	if repairedStr, err := jsonrepair.JSONRepair(fixed); err == nil {
		return repairedStr
	}

	return fixed
}
