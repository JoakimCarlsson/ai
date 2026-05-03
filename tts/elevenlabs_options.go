package tts

// ElevenLabsOption is a function that configures ElevenLabs-specific options.
type ElevenLabsOption func(*elevenLabsOptions)

type elevenLabsOptions struct {
	baseURL      string
	voiceID      string
	outputFormat string
}

// WithElevenLabsBaseURL sets a custom base URL for the ElevenLabs API.
// This is useful for testing or using alternative endpoints.
// Default is "https://api.elevenlabs.io/v1".
func WithElevenLabsBaseURL(baseURL string) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.baseURL = baseURL
	}
}

// WithElevenLabsVoiceID sets the voice ID used by every GenerateAudio /
// StreamAudio call on this client. Voice is set at client construction
// time, like model — there is no per-call override.
func WithElevenLabsVoiceID(voiceID string) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.voiceID = voiceID
	}
}

// WithElevenLabsOutputFormat sets the audio output format (e.g.
// "mp3_44100_128", "pcm_16000", "pcm_24000") used by every GenerateAudio /
// StreamAudio call on this client. Set at client construction time;
// there is no per-call override. Default is "mp3_44100_128".
func WithElevenLabsOutputFormat(format string) ElevenLabsOption {
	return func(options *elevenLabsOptions) {
		options.outputFormat = format
	}
}
