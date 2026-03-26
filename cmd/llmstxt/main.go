// Package main generates llms.txt and llms-full.txt from MkDocs documentation.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var badgeRe = regexp.MustCompile(`^\[!\[.*?\]\(.*?\)\]\(.*?\)$`)

func main() {
	configPath := flag.String(
		"config",
		"www/mkdocs.yml",
		"path to mkdocs.yml",
	)
	docsDir := flag.String(
		"docs",
		"www/docs",
		"path to docs directory",
	)
	outDir := flag.String(
		"out",
		"www/docs",
		"output directory for generated files",
	)
	baseURL := flag.String(
		"url",
		"",
		"override base URL (default: from mkdocs.yml)",
	)
	flag.Parse()

	siteName, siteURL, pages, err := parseConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	if *baseURL != "" {
		siteURL = *baseURL
	}
	siteURL = strings.TrimRight(siteURL, "/")

	pageContents := make(map[string]string, len(pages))
	for _, p := range pages {
		data, err := os.ReadFile(
			filepath.Join(*docsDir, p.Path),
		)
		if err != nil {
			log.Fatalf("reading %s: %v", p.Path, err)
		}
		pageContents[p.Path] = string(data)
	}

	llmsTxt := generateLlmsTxt(
		siteName,
		siteURL,
		pages,
		pageContents,
	)
	llmsFullTxt := generateLlmsFullTxt(
		siteName,
		pages,
		pageContents,
	)

	if err := os.WriteFile(
		filepath.Join(*outDir, "llms.txt"),
		[]byte(llmsTxt),
		0o644,
	); err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(*outDir, "llms-full.txt"),
		[]byte(llmsFullTxt),
		0o644,
	); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Generated llms.txt and llms-full.txt in %s\n",
		*outDir,
	)
}

func generateLlmsTxt(
	siteName, siteURL string,
	pages []NavPage,
	contents map[string]string,
) string {
	var b strings.Builder

	desc := extractDescription(contents[pages[0].Path])
	fmt.Fprintf(&b, "# %s\n\n", siteName)
	fmt.Fprintf(&b, "> %s\n\n", desc)
	b.WriteString("## Docs\n")

	currentSection := ""
	for _, p := range pages {
		if p.Section != "" && p.Section != currentSection {
			currentSection = p.Section
			fmt.Fprintf(&b, "\n### %s\n", currentSection)
		}

		pageURL := pageToURL(siteURL, p.Path)
		summary := extractFirstSentence(contents[p.Path])
		fmt.Fprintf(
			&b,
			"- [%s](%s): %s\n",
			p.Title,
			pageURL,
			summary,
		)
	}

	return b.String()
}

func generateLlmsFullTxt(
	siteName string,
	pages []NavPage,
	contents map[string]string,
) string {
	var b strings.Builder

	fmt.Fprintf(
		&b,
		"# %s - Complete Documentation\n\n",
		siteName,
	)

	b.WriteString("## Table of Contents\n\n")
	for _, p := range pages {
		label := pageLabel(p)
		anchor := toAnchor(label)
		fmt.Fprintf(&b, "- [%s](#%s)\n", label, anchor)
	}

	for _, p := range pages {
		label := pageLabel(p)
		b.WriteString("\n---\n\n")
		fmt.Fprintf(&b, "## %s\n\n", label)
		fmt.Fprintf(&b, "> Source: %s\n\n", p.Path)
		b.WriteString(stripMkDocsSyntax(contents[p.Path]))
		b.WriteString("\n")
	}

	return b.String()
}

func pageLabel(p NavPage) string {
	if p.Section == "" {
		return p.Title
	}
	return p.Section + " > " + p.Title
}

func pageToURL(baseURL, path string) string {
	path = strings.TrimSuffix(path, ".md")
	if path == "index" {
		return baseURL + "/"
	}
	path = strings.TrimSuffix(path, "/index")
	return baseURL + "/" + path + "/"
}

func toAnchor(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteRune('-')
		case r == '>':
			// skip
		}
	}
	result := b.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return strings.Trim(result, "-")
}

func extractDescription(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if badgeRe.MatchString(trimmed) {
			continue
		}
		return trimmed
	}
	return ""
}

func extractFirstSentence(content string) string {
	inCode := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if badgeRe.MatchString(trimmed) {
			continue
		}
		if strings.HasPrefix(trimmed, "!!! ") ||
			strings.HasPrefix(trimmed, "??? ") {
			continue
		}
		if strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") &&
			!strings.ContainsAny(trimmed, ".") {
			continue
		}

		if idx := strings.Index(trimmed, ". "); idx != -1 {
			return trimmed[:idx+1]
		}
		if strings.HasSuffix(trimmed, ".") {
			return trimmed
		}
		return trimmed
	}
	return ""
}
