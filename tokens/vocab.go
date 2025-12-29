package tokens

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

func loadVocabulary(data []byte) (map[string]int, map[int]string, error) {
	encoder := make(map[string]int)
	decoder := make(map[int]string)

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			continue
		}

		tokenBytes, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			continue
		}

		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		token := string(tokenBytes)
		encoder[token] = rank
		decoder[rank] = token
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to scan vocabulary: %w", err)
	}

	if len(encoder) == 0 {
		return nil, nil, fmt.Errorf("vocabulary is empty")
	}

	return encoder, decoder, nil
}
