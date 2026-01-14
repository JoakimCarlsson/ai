package audio

// ElevenLabsOption is a function that configures ElevenLabs-specific options.
type ElevenLabsOption func(*elevenLabsOptions)

type elevenLabsOptions struct {
	baseURL string
}

// WithElevenLabsBaseURL sets a custom base URL for the ElevenLabs API.
// This is useful for testing or using alternative endpoints.
// Default is "https://api.elevenlabs.io/v1".
func WithElevenLabsBaseURL(baseURL string) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.baseURL = baseURL
	}
}
