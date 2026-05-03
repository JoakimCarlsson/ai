package tts

// DeepgramOption is a function that configures Deepgram-specific TTS options.
type DeepgramOption func(*deepgramOptions)

type deepgramOptions struct {
	baseURL    string
	model      string
	encoding   string
	container  string
	sampleRate int
	bitRate    int
}

// WithDeepgramBaseURL sets a custom base URL for the Deepgram API.
// Default is "https://api.deepgram.com/v1".
func WithDeepgramBaseURL(baseURL string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.baseURL = baseURL
	}
}

// WithDeepgramModel sets the full Deepgram TTS model identifier (e.g.
// "aura-2-thalia-en", "aura-asteria-en"). The model encodes both voice and
// language. Set at construction time. Overrides the APIModel from
// model.AudioModel if both are provided.
func WithDeepgramModel(name string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.model = name
	}
}

// WithDeepgramEncoding sets the audio encoding (e.g. "mp3", "linear16",
// "mulaw", "alaw", "opus", "flac", "aac"). Default is "mp3".
func WithDeepgramEncoding(encoding string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.encoding = encoding
	}
}

// WithDeepgramContainer sets the audio container (e.g. "wav", "ogg", "none").
// For raw PCM use "none" with encoding "linear16".
func WithDeepgramContainer(container string) DeepgramOption {
	return func(options *deepgramOptions) {
		options.container = container
	}
}

// WithDeepgramSampleRate sets the audio sample rate in Hz (e.g. 8000, 16000,
// 24000, 48000). Only valid for uncompressed encodings (linear16, mulaw,
// alaw).
func WithDeepgramSampleRate(rate int) DeepgramOption {
	return func(options *deepgramOptions) {
		options.sampleRate = rate
	}
}

// WithDeepgramBitRate sets the audio bit rate in bits per second for
// compressed encodings (mp3, opus, aac). Ignored for uncompressed encodings.
func WithDeepgramBitRate(rate int) DeepgramOption {
	return func(options *deepgramOptions) {
		options.bitRate = rate
	}
}
