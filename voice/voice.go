package voice

import (
	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tts"
)

// VoiceAgent is the configured, reusable definition of a voice agent. One
// VoiceAgent can power any number of concurrent Conversations.
type VoiceAgent struct {
	llm               llm.LLM
	stt               stt.SpeechToText
	tts               tts.Generation
	systemPrompt      string
	tools             []tool.BaseTool
	maxToolIterations int
	filler            FillerConfig
	toolSound         ToolSoundConfig
}

const defaultMaxToolIterations = 4

// New constructs a VoiceAgent from the given clients and options. STT and TTS
// are passed as interfaces and are fully pluggable. The TTS client is type-
// asserted for tts.StreamingTextProvider when a conversation opens; if absent,
// the pipeline uses a sentence-buffered single-shot fallback.
func New(
	llmClient llm.LLM,
	sttClient stt.SpeechToText,
	ttsClient tts.Generation,
	opts ...Option,
) *VoiceAgent {
	v := &VoiceAgent{
		llm:               llmClient,
		stt:               sttClient,
		tts:               ttsClient,
		maxToolIterations: defaultMaxToolIterations,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}
