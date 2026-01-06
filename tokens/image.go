package tokens

import "github.com/joakimcarlsson/ai/message"

// DefaultImageTokens is the default token estimate for images.
// This is a rough approximation; actual tokens vary by image size and detail level.
const DefaultImageTokens int64 = 512

// EstimateImageTokens returns an estimated token count for binary content (images).
// This is a rough approximation since actual token counts depend on image dimensions
// and the detail level requested by the provider.
func EstimateImageTokens(img message.BinaryContent) int64 {
	return DefaultImageTokens
}
