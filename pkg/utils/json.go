package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func FixJSON(input string) string {
	// If the decoded string is still wrapped in extra quotes, handle unquoting
	if strings.HasPrefix(input, "\"") && strings.HasSuffix(input, "\"") {
		unquotedData, err := strconv.Unquote(input)
		if err == nil {
			input = unquotedData
		}
	}

	// Clean up any remaining escape sequences (like \n or \") if present
	// https://stackoverflow.com/questions/53776683/regex-find-newline-between-double-quotes-and-replace-with-space/53777149#53777149
	matchNewlines := regexp.MustCompile(`[\r\n]`)
	escapeNewlines := func(s string) string {
		return matchNewlines.ReplaceAllString(s, "\\n")
	}
	re := regexp.MustCompile(`"[^"\\]*(?:\\[\s\S][^"\\]*)*"`)
	input = re.ReplaceAllStringFunc(input, escapeNewlines)

	// https://www.reddit.com/r/golang/comments/14lkgw4/repairing_malformed_json_in_go/
	jsonRegexp := regexp.MustCompile(`(?m:^\s*"([^"]*)"\s*:\s*"(.*?)"\s*(,?)\s*$)`)
	fixed := jsonRegexp.ReplaceAllStringFunc(input, func(s string) string {
		submatches := jsonRegexp.FindStringSubmatch(s)
		return fmt.Sprintf(`"%s": "%s"%s`, submatches[1], strings.ReplaceAll(submatches[2], `"`, `\"`), submatches[3])
	})

	return fixed
}
