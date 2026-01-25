package prompt

import (
	"fmt"
	"strings"
)

// ValidationError is returned when required template variables are missing.
type ValidationError struct {
	Missing []string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing required variables: %s", strings.Join(e.Missing, ", "))
}

func validateRequired(data map[string]any, required []string) error {
	if len(required) == 0 {
		return nil
	}

	var missing []string
	for _, key := range required {
		if _, ok := data[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return &ValidationError{Missing: missing}
	}
	return nil
}
