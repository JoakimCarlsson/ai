package main

import (
	"strconv"
	"strings"
	"unicode"
)

// catalogID derives the catalog ID for a model the catalog has not seen
// before: the vendor prefix is dropped and variant suffixes are folded into the
// slug, so "anthropic/claude-opus-5:batch" becomes "claude-opus-5-batch".
func catalogID(apiModel string) string {
	slug := apiModel
	if i := strings.LastIndex(slug, "/"); i >= 0 {
		slug = slug[i+1:]
	}
	return strings.ReplaceAll(slug, ":", "-")
}

// uniqueID derives a catalog ID that no other entry in the package already
// claims. Two vendors can ship the same slug, and a catalog can already hold a
// hand-picked ID for one of them, so a clash falls back to a vendor-qualified
// ID.
func uniqueID(apiModel, prefix string, taken map[string]bool) string {
	id := prefix + catalogID(apiModel)
	if !taken[id] {
		return id
	}

	vendor, slug := splitSlug(apiModel)
	if vendor != "" {
		qualified := prefix + strings.ReplaceAll(vendor, "/", "-") + "-" +
			strings.ReplaceAll(slug, ":", "-")
		if !taken[qualified] {
			return qualified
		}
	}

	for i := 2; i < 100; i++ {
		numbered := id + "-" + strconv.Itoa(i)
		if !taken[numbered] {
			return numbered
		}
	}
	return id
}

// uniqueConstName derives an exported Go identifier for a model, falling back
// to a vendor-qualified name when the plain one is already taken by another
// entry in the same package.
func uniqueConstName(apiModel string, taken map[string]bool) string {
	vendor, slug := splitSlug(apiModel)

	name := pascal(slug)
	if name == "" {
		name = pascal(apiModel)
	}
	if !taken[name] {
		return name
	}

	if vendor != "" {
		qualified := pascal(vendor) + name
		if !taken[qualified] {
			return qualified
		}
	}

	for i := 2; ; i++ {
		numbered := name + string(rune('0'+i%10))
		if i < 10 && !taken[numbered] {
			return numbered
		}
		if i >= 10 {
			return name + pascal(apiModel)
		}
	}
}

func splitSlug(apiModel string) (vendor, slug string) {
	slug = apiModel
	if i := strings.LastIndex(apiModel, "/"); i >= 0 {
		vendor, slug = apiModel[:i], apiModel[i+1:]
	}
	return vendor, slug
}

// acronyms are spelled in full caps in the hand-written catalogs, so derived
// names match them instead of reading as Gpt41 next to GPT41.
var acronyms = map[string]string{
	"gpt": "GPT",
	"oss": "OSS",
	"tts": "TTS",
	"stt": "STT",
	"asr": "ASR",
	"ai":  "AI",
	"llm": "LLM",
	"api": "API",
	"hd":  "HD",
	"glm": "GLM",
	"fp":  "FP",
}

// pascal turns a model slug into an exported Go identifier: separators become
// word boundaries, everything else is stripped, and a leading digit is prefixed
// so the result is a valid identifier.
func pascal(s string) string {
	var b strings.Builder
	for _, token := range tokenize(s) {
		if full, ok := acronyms[strings.ToLower(token)]; ok {
			b.WriteString(full)
			continue
		}
		b.WriteString(strings.ToUpper(token[:1]))
		b.WriteString(token[1:])
	}

	name := b.String()
	if name == "" {
		return ""
	}
	if unicode.IsDigit(rune(name[0])) {
		return "M" + name
	}
	return name
}

// tokenize splits a slug on separators and on letter/digit boundaries, so
// "gpt-4.1-mini" yields gpt, 4, 1, mini.
func tokenize(s string) []string {
	var (
		tokens []string
		cur    strings.Builder
		digits bool
	)
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			if digits {
				flush()
				digits = false
			}
			cur.WriteRune(r)
		case unicode.IsDigit(r):
			if !digits && cur.Len() > 0 {
				flush()
			}
			digits = true
			cur.WriteRune(r)
		default:
			flush()
			digits = false
		}
	}
	flush()
	return tokens
}
