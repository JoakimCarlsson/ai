package voice

import "strings"

// minSentenceRunes is the floor below which a candidate sentence keeps
// accumulating instead of being emitted. Avoids surfacing fragments like
// "Mr." or "1." in numbered lists as standalone TTS chunks.
const minSentenceRunes = 12

// sentenceChunker accumulates streaming text and emits chunks at sentence
// boundaries. A boundary is one of ".", "!", "?", or "\n" followed by
// whitespace. Short candidates are deferred until the next boundary.
type sentenceChunker struct {
	buf strings.Builder
}

// push appends piece to the buffer and returns any newly completed sentences.
// The trailing partial sentence stays in the buffer for the next call.
func (s *sentenceChunker) push(piece string) []string {
	if piece == "" {
		return nil
	}
	s.buf.WriteString(piece)
	current := s.buf.String()

	var out []string
	start := 0
	for i := 0; i < len(current); i++ {
		c := current[i]
		if c != '.' && c != '!' && c != '?' && c != '\n' {
			continue
		}
		if i+1 >= len(current) {
			break
		}
		next := current[i+1]
		if next != ' ' && next != '\n' && next != '\t' {
			continue
		}
		end := i + 1
		candidate := current[start:end]
		if runeLen(candidate) < minSentenceRunes {
			continue
		}
		out = append(out, candidate)
		start = end
	}

	if start > 0 {
		remainder := current[start:]
		s.buf.Reset()
		s.buf.WriteString(remainder)
	}
	return out
}

// flushRemainder returns any text left in the buffer (trimmed) and clears it.
// Call at end-of-stream to ensure trailing text without terminal punctuation
// is still spoken.
func (s *sentenceChunker) flushRemainder() string {
	rem := strings.TrimSpace(s.buf.String())
	s.buf.Reset()
	return rem
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
