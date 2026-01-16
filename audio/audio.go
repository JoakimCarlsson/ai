// Package audio provides a unified interface for generating audio from text using
// various AI providers.
//
// This package abstracts the differences between audio generation providers like ElevenLabs
// and OpenAI TTS, offering a consistent API for text-to-speech conversion with support for
// voice selection, audio streaming, and usage tracking.
//
// Key features include:
//   - Text-to-speech generation from text
//   - Support for multiple output formats (mp3, pcm, wav)
//   - Streaming audio generation for real-time playback
//   - Voice listing and selection
//   - Voice settings customization (stability, similarity, style)
//   - Character usage tracking and cost calculation
//
// Example usage:
//
//	client, err := audio.NewAudioGeneration(model.ProviderElevenLabs,
//		audio.WithAPIKey("your-api-key"),
//		audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	response, err := client.GenerateAudio(ctx, "Hello, how are you today?",
//		audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	os.WriteFile("output.mp3", response.AudioData, 0644)
package audio

import (
	"context"
	"fmt"
	"time"

	"github.com/joakimcarlsson/ai/model"
)

// AudioUsage tracks the resource consumption for audio generation operations.
type AudioUsage struct {
	// Characters is the number of characters processed.
	Characters int64
}

// AlignmentData contains character-level timing information for generated audio.
type AlignmentData struct {
	Characters                 []string
	CharacterStartTimesSeconds []float64
	CharacterEndTimesSeconds   []float64
}

// CharAlignment represents timing information for a single character.
type CharAlignment struct {
	Text  string
	Start float64
	End   float64
}

// WordAlignment represents timing information for a single word.
type WordAlignment struct {
	Text  string
	Start float64
	End   float64
	Loss  float64
}

// ForcedAlignmentData contains the response from forced alignment operations.
type ForcedAlignmentData struct {
	Characters []CharAlignment
	Words      []WordAlignment
	Loss       float64
}

// AudioResponse contains the generated audio and metadata from an audio generation request.
type AudioResponse struct {
	// AudioData contains the audio file data as bytes.
	AudioData []byte
	// ContentType specifies the MIME type of the audio data (e.g., "audio/mpeg", "audio/pcm").
	ContentType string
	// Usage tracks resource consumption for this request.
	Usage AudioUsage
	// Model identifies which audio generation model was used.
	Model string
	// Alignment contains character-level timing information aligned to the original input text.
	Alignment *AlignmentData
	// NormalizedAlignment contains character-level timing information aligned to normalized text.
	NormalizedAlignment *AlignmentData
}

// AudioChunk represents a piece of streaming audio data.
type AudioChunk struct {
	// Data contains the audio chunk bytes.
	Data []byte
	// Error contains any error that occurred during streaming.
	Error error
	// Done indicates if this is the final chunk.
	Done bool
}

// Voice represents an available voice for audio generation.
type Voice struct {
	// VoiceID is the unique identifier for the voice.
	VoiceID string
	// Name is the human-readable name of the voice.
	Name string
	// Category categorizes the voice (e.g., "premade", "cloned", "professional").
	Category string
	// Description provides additional information about the voice.
	Description string
	// PreviewURL is an optional URL to preview the voice.
	PreviewURL string
	// Labels contains optional metadata tags for the voice.
	Labels map[string]string
}

// AudioGeneration defines the interface for generating audio from text using TTS providers.
// It provides methods for synchronous audio generation, streaming, and voice management.
type AudioGeneration interface {
	// GenerateAudio creates audio from text and returns the complete audio data.
	// The optional GenerationOption parameters can customize the generation (voice, format, settings, etc.).
	GenerateAudio(
		ctx context.Context,
		text string,
		options ...GenerationOption,
	) (*AudioResponse, error)

	// StreamAudio creates audio from text and returns a channel of audio chunks for streaming playback.
	// This is useful for real-time audio playback with lower latency.
	StreamAudio(
		ctx context.Context,
		text string,
		options ...GenerationOption,
	) (<-chan AudioChunk, error)

	// ListVoices retrieves the list of available voices from the provider.
	ListVoices(ctx context.Context) ([]Voice, error)

	// Model returns the audio generation model configuration being used.
	Model() model.AudioModel
}

// ForcedAlignmentProvider defines the interface for providers that support forced alignment.
// Forced alignment matches existing audio with a transcript to produce timing data.
type ForcedAlignmentProvider interface {
	// GenerateForcedAlignment aligns an existing audio file with its transcript.
	// Returns character-level and word-level timing information.
	GenerateForcedAlignment(
		ctx context.Context,
		audioFile []byte,
		transcript string,
	) (*ForcedAlignmentData, error)
}

type audioGenerationClientOptions struct {
	apiKey  string
	model   model.AudioModel
	timeout *time.Duration

	elevenLabsOptions []ElevenLabsOption
}

type AudioGenerationClientOption func(*audioGenerationClientOptions)

type AudioGenerationClient interface {
	generate(
		ctx context.Context,
		text string,
		options ...GenerationOption,
	) (*AudioResponse, error)
	stream(
		ctx context.Context,
		text string,
		options ...GenerationOption,
	) (<-chan AudioChunk, error)
	listVoices(ctx context.Context) ([]Voice, error)
}

type baseAudioGeneration[C AudioGenerationClient] struct {
	options audioGenerationClientOptions
	client  C
}

// NewAudioGeneration creates a new audio generation client for the specified provider.
// Supported providers include ElevenLabs. Use WithModel() to specify the audio generation model
// and WithAPIKey() for authentication.
func NewAudioGeneration(
	provider model.ModelProvider,
	opts ...AudioGenerationClientOption,
) (AudioGeneration, error) {
	clientOptions := audioGenerationClientOptions{}
	for _, o := range opts {
		o(&clientOptions)
	}

	switch provider {
	case model.ProviderElevenLabs:
		return &baseAudioGeneration[ElevenLabsClient]{
			options: clientOptions,
			client:  newElevenLabsClient(clientOptions),
		}, nil
	}

	return nil, fmt.Errorf(
		"audio generation provider not supported: %s",
		provider,
	)
}

func (a *baseAudioGeneration[C]) GenerateAudio(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (*AudioResponse, error) {
	return a.client.generate(ctx, text, options...)
}

func (a *baseAudioGeneration[C]) StreamAudio(
	ctx context.Context,
	text string,
	options ...GenerationOption,
) (<-chan AudioChunk, error) {
	return a.client.stream(ctx, text, options...)
}

func (a *baseAudioGeneration[C]) ListVoices(ctx context.Context) ([]Voice, error) {
	return a.client.listVoices(ctx)
}

func (a *baseAudioGeneration[C]) Model() model.AudioModel {
	return a.options.model
}

// WithAPIKey sets the API key for authentication with the audio generation provider.
func WithAPIKey(apiKey string) AudioGenerationClientOption {
	return func(options *audioGenerationClientOptions) {
		options.apiKey = apiKey
	}
}

// WithModel specifies which audio generation model to use for creating audio.
func WithModel(model model.AudioModel) AudioGenerationClientOption {
	return func(options *audioGenerationClientOptions) {
		options.model = model
	}
}

// WithTimeout sets the maximum duration to wait for audio generation requests to complete.
func WithTimeout(timeout time.Duration) AudioGenerationClientOption {
	return func(options *audioGenerationClientOptions) {
		options.timeout = &timeout
	}
}

// WithElevenLabsOptions applies ElevenLabs-specific configuration options.
func WithElevenLabsOptions(
	elevenLabsOptions ...ElevenLabsOption,
) AudioGenerationClientOption {
	return func(options *audioGenerationClientOptions) {
		options.elevenLabsOptions = elevenLabsOptions
	}
}

// GenerationOptions contains parameters for customizing audio generation requests.
type GenerationOptions struct {
	// VoiceID specifies which voice to use for audio generation.
	VoiceID string
	// OutputFormat specifies the audio format (e.g., "mp3_44100_128", "pcm_16000").
	OutputFormat string
	// Stability controls voice consistency (0.0 to 1.0).
	Stability *float64
	// SimilarityBoost controls how much the voice matches the original (0.0 to 1.0).
	SimilarityBoost *float64
	// Style controls the style exaggeration (0.0 to 1.0).
	Style *float64
	// SpeakerBoost enhances speaker similarity when enabled.
	SpeakerBoost *bool
	// OptimizeStreamingLatency optimizes for lower latency (0-4).
	OptimizeStreamingLatency *int
	// EnableAlignment enables character-level timing data in the response.
	EnableAlignment bool
}

// GenerationOption is a function that configures GenerationOptions.
type GenerationOption func(*GenerationOptions)

// WithVoiceID sets the voice to use for audio generation.
func WithVoiceID(voiceID string) GenerationOption {
	return func(options *GenerationOptions) {
		options.VoiceID = voiceID
	}
}

// WithOutputFormat sets the audio format for the generated audio.
// Common formats: "mp3_44100_128", "mp3_44100_192", "pcm_16000", "pcm_22050", "pcm_24000", "pcm_44100".
func WithOutputFormat(format string) GenerationOption {
	return func(options *GenerationOptions) {
		options.OutputFormat = format
	}
}

// WithStability sets the voice stability (0.0 to 1.0).
// Lower values make the voice more variable and expressive, higher values make it more consistent.
func WithStability(stability float64) GenerationOption {
	return func(options *GenerationOptions) {
		options.Stability = &stability
	}
}

// WithSimilarityBoost sets how much the voice should match the original voice (0.0 to 1.0).
// Higher values increase similarity but may reduce stability.
func WithSimilarityBoost(boost float64) GenerationOption {
	return func(options *GenerationOptions) {
		options.SimilarityBoost = &boost
	}
}

// WithStyle sets the style exaggeration (0.0 to 1.0).
// Higher values make the voice more expressive and exaggerated.
func WithStyle(style float64) GenerationOption {
	return func(options *GenerationOptions) {
		options.Style = &style
	}
}

// WithSpeakerBoost enables or disables speaker boost for enhanced similarity.
func WithSpeakerBoost(enabled bool) GenerationOption {
	return func(options *GenerationOptions) {
		options.SpeakerBoost = &enabled
	}
}

// WithOptimizeStreamingLatency sets the streaming latency optimization level (0-4).
// Higher values reduce latency but may decrease quality. 0 = no optimization, 4 = maximum optimization.
func WithOptimizeStreamingLatency(level int) GenerationOption {
	return func(options *GenerationOptions) {
		options.OptimizeStreamingLatency = &level
	}
}

// WithAlignmentEnabled enables or disables character-level timing data in the response.
// When enabled, the audio generation will use the /with-timestamps endpoint and populate
// the Alignment and NormalizedAlignment fields in the AudioResponse.
// This is useful for subtitles, word highlighting, lip sync, and other timing-dependent features.
func WithAlignmentEnabled(enabled bool) GenerationOption {
	return func(options *GenerationOptions) {
		options.EnableAlignment = enabled
	}
}
