package agent

import (
	"fmt"
	"regexp"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(\?)?\}`)

func processTemplate(template string, state map[string]string) (string, error) {
	if state == nil {
		state = make(map[string]string)
	}

	var result strings.Builder
	lastIndex := 0
	matches := placeholderRegex.FindAllStringSubmatchIndex(template, -1)

	for _, match := range matches {
		fullStart, fullEnd := match[0], match[1]
		nameStart, nameEnd := match[2], match[3]
		optionalStart, optionalEnd := match[4], match[5]

		result.WriteString(template[lastIndex:fullStart])

		varName := template[nameStart:nameEnd]
		optional := optionalStart != -1 && optionalEnd != -1

		value, exists := state[varName]
		if !exists {
			if optional {
				lastIndex = fullEnd
				continue
			}
			return "", fmt.Errorf("missing required template variable: %s", varName)
		}

		result.WriteString(value)
		lastIndex = fullEnd
	}

	result.WriteString(template[lastIndex:])
	return result.String(), nil
}
