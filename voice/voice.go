package voice

import (
	"context"

	"github.com/joakimcarlsson/ai/llm"
	"github.com/joakimcarlsson/ai/memory"
	"github.com/joakimcarlsson/ai/session"
	"github.com/joakimcarlsson/ai/stt"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tool"
	"github.com/joakimcarlsson/ai/tts"
)

// Agent is the configured, reusable definition of a voice agent. One
// Agent can power any number of concurrent Conversations.
type Agent struct {
	llm               llm.LLM
	stt               stt.SpeechToText
	tts               tts.Generation
	systemPrompt      string
	tools             []tool.BaseTool
	toolsets          []tool.Toolset
	maxToolIterations int
	filler            FillerConfig
	toolSound         ToolSoundConfig
	bargeIn           BargeInPolicy
	session           session.Session
	contextStrategy   tokens.Strategy
	maxContextTokens  int64
	hooks             []Hooks
	handoffs          []HandoffConfig
	memory            memory.Store
	memoryID          string
	autoExtract       bool
	autoDedup         bool
	memoryLLM         llm.LLM
}

// toolsForContext returns the union of static tools, toolset-resolved
// tools, and (when memory is configured without auto-extract) the memory
// management tools. Toolsets are evaluated on each call so implementations
// can return different tools depending on per-call ctx values.
func (v *Agent) toolsForContext(ctx context.Context) []tool.BaseTool {
	hasToolsets := len(v.toolsets) > 0
	hasMemoryTools := v.memory != nil && !v.autoExtract && v.memoryID != ""
	if !hasToolsets && !hasMemoryTools {
		return v.tools
	}
	out := make([]tool.BaseTool, 0, len(v.tools))
	out = append(out, v.tools...)
	for _, ts := range v.toolsets {
		out = append(out, ts.Tools(ctx)...)
	}
	if hasMemoryTools {
		out = append(out, memory.Tools(v.memory, v.memoryID)...)
	}
	return out
}

const defaultMaxToolIterations = 4

// New constructs a Agent from the given clients and options. STT and TTS
// are passed as interfaces and are fully pluggable. The TTS client is type-
// asserted for tts.StreamingTextProvider when a conversation opens; if absent,
// the pipeline uses a sentence-buffered single-shot fallback.
func New(
	llmClient llm.LLM,
	sttClient stt.SpeechToText,
	ttsClient tts.Generation,
	opts ...Option,
) *Agent {
	v := &Agent{
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
