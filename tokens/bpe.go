package tokens

import (
	"regexp"
	"sync"
)

// BPETokenizer implements byte pair encoding tokenization using the cl100k_base vocabulary.
type BPETokenizer struct {
	encoder map[string]int
	decoder map[int]string
	pattern *regexp.Regexp
	cache   map[string][]int
	cacheMu sync.RWMutex
}

// NewBPETokenizer creates a new BPE tokenizer with the cl100k_base vocabulary.
func NewBPETokenizer() (*BPETokenizer, error) {
	encoder, decoder, err := loadVocabulary(cl100kBaseVocab)
	if err != nil {
		return nil, err
	}

	pattern := regexp.MustCompile(`(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+`)

	return &BPETokenizer{
		encoder: encoder,
		decoder: decoder,
		pattern: pattern,
		cache:   make(map[string][]int),
	}, nil
}

// Encode converts text to token IDs.
func (t *BPETokenizer) Encode(text string) []int {
	if text == "" {
		return nil
	}

	var tokens []int
	chunks := t.pattern.FindAllString(text, -1)

	for _, chunk := range chunks {
		t.cacheMu.RLock()
		cached, ok := t.cache[chunk]
		t.cacheMu.RUnlock()

		if ok {
			tokens = append(tokens, cached...)
			continue
		}

		chunkTokens := t.bpeEncode(chunk)

		t.cacheMu.Lock()
		t.cache[chunk] = chunkTokens
		t.cacheMu.Unlock()

		tokens = append(tokens, chunkTokens...)
	}

	return tokens
}

// Count returns the number of tokens in the text.
func (t *BPETokenizer) Count(text string) int {
	return len(t.Encode(text))
}

// Decode converts token IDs back to text.
func (t *BPETokenizer) Decode(tokens []int) string {
	var result []byte
	for _, token := range tokens {
		if s, ok := t.decoder[token]; ok {
			result = append(result, s...)
		}
	}
	return string(result)
}

func (t *BPETokenizer) bpeEncode(text string) []int {
	if len(text) == 0 {
		return nil
	}

	if rank, ok := t.encoder[text]; ok {
		return []int{rank}
	}

	textBytes := []byte(text)
	pieces := make([][]byte, len(textBytes))
	for i, b := range textBytes {
		pieces[i] = []byte{b}
	}

	for len(pieces) > 1 {
		minRank := -1
		minIdx := -1

		for i := 0; i < len(pieces)-1; i++ {
			pair := string(pieces[i]) + string(pieces[i+1])
			if rank, ok := t.encoder[pair]; ok {
				if minRank == -1 || rank < minRank {
					minRank = rank
					minIdx = i
				}
			}
		}

		if minIdx == -1 {
			break
		}

		merged := append([]byte{}, pieces[minIdx]...)
		merged = append(merged, pieces[minIdx+1]...)

		newPieces := make([][]byte, 0, len(pieces)-1)
		newPieces = append(newPieces, pieces[:minIdx]...)
		newPieces = append(newPieces, merged)
		newPieces = append(newPieces, pieces[minIdx+2:]...)
		pieces = newPieces
	}

	result := make([]int, 0, len(pieces))
	for _, piece := range pieces {
		if rank, ok := t.encoder[string(piece)]; ok {
			result = append(result, rank)
		} else {
			for _, b := range piece {
				if rank, ok := t.encoder[string([]byte{b})]; ok {
					result = append(result, rank)
				}
			}
		}
	}

	return result
}
