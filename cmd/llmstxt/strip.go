package main

import (
	"regexp"
	"strings"
)

var (
	admonitionRe = regexp.MustCompile(
		`^(!{3}|\?{3}\+?) (\w+)\s*(?:"([^"]*)")?`,
	)
	tabRe  = regexp.MustCompile(`^=== "([^"]*)"`)
	attrRe = regexp.MustCompile(`\{[.:][^}]*\}\s*$`)
)

func stripMkDocsSyntax(content string) string {
	lines := strings.Split(content, "\n")
	var out []string

	inCodeBlock := false
	inAdmonition := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			out = append(out, line)
			continue
		}

		if inCodeBlock {
			out = append(out, line)
			continue
		}

		if inAdmonition {
			if strings.HasPrefix(line, "    ") {
				out = append(out, "> "+line[4:])
				continue
			}
			if trimmed == "" {
				out = append(out, ">")
				continue
			}
			inAdmonition = false
		}

		if m := admonitionRe.FindStringSubmatch(line); m != nil {
			title := m[3]
			if title == "" {
				title = strings.Title(m[2]) //nolint:staticcheck
			}
			out = append(out, "> **"+title+":**")
			inAdmonition = true
			continue
		}

		if m := tabRe.FindStringSubmatch(line); m != nil {
			out = append(out, "**"+m[1]+":**")
			continue
		}

		line = attrRe.ReplaceAllString(line, "")
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}
