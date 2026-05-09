# Voice Agent

The `voice` package provides a streaming voice agent that runs an STT → LLM → TTS pipeline over a duplex audio transport. STT and TTS are pluggable: pass any `stt.SpeechToText` and any `tts.Generation` implementation when you construct the agent.

When the TTS client implements `tts.StreamingTextProvider` (ElevenLabs does), the pipeline streams LLM tokens directly into TTS for end-to-end concurrent text-to-audio. When it does not (OpenAI, etc.), the pipeline buffers the LLM output at sentence boundaries and falls back to single-shot `tts.Generation.StreamAudio` per sentence.

## Quick start

```go
import (
    llmopenai "github.com/joakimcarlsson/ai/llm/openai"
    "github.com/joakimcarlsson/ai/model"
    "github.com/joakimcarlsson/ai/stt"
    sttelevenlabs "github.com/joakimcarlsson/ai/stt/elevenlabs"
    "github.com/joakimcarlsson/ai/tts"
    ttselevenlabs "github.com/joakimcarlsson/ai/tts/elevenlabs"
    "github.com/joakimcarlsson/ai/voice"
)

llmClient := llmopenai.NewLLM(
    llmopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    llmopenai.WithModel(model.OpenAIModels[model.GPT4oMini]),
)

sttClient := sttelevenlabs.NewSpeechToText(
    sttelevenlabs.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    sttelevenlabs.WithModel(model.ElevenLabsTranscriptionModels[model.ElevenLabsScribeV2]),
    sttelevenlabs.WithStreamCommitStrategy(sttelevenlabs.CommitStrategyVAD),
    sttelevenlabs.WithStreamVADSilenceMs(500),
)

ttsClient := ttselevenlabs.NewGeneration(
    ttselevenlabs.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    ttselevenlabs.WithModel(model.ElevenLabsAudioModels[model.ElevenFlashV2_5]),
    ttselevenlabs.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
    ttselevenlabs.WithOutputFormat("pcm_16000"),
)

agent := voice.New(llmClient, sttClient, ttsClient,
    voice.WithSystemPrompt("You are a concise voice assistant."),
    voice.WithTools(myTool),
)

conv, err := agent.StartConversation(ctx, transport)
if err != nil {
    log.Fatal(err)
}

go func() {
    for evt := range conv.Events() {
        // observe transcripts, deltas, tool calls, etc.
    }
}()

if err := conv.Wait(); err != nil {
    log.Println("conversation ended with error:", err)
}
```

## Configuration options

| Option | Description | Default |
|--------|-------------|---------|
| `WithSystemPrompt(prompt)` | System prompt prepended to every LLM call | none |
| `WithTools(tools...)` | Tools the LLM can call during a conversation | none |
| `WithMaxToolIterations(n)` | Cap on tool-call rounds inside one assistant turn | 4 |
| `WithFiller(cfg)` | Speak a short filler phrase if the LLM is slow to produce its first content delta | disabled |
| `WithToolSound(cfg)` | Loop ambient PCM audio while a tool is executing | disabled |
| `WithBargeIn(policy)` | Cancel the current turn when the user speaks over the agent | `BargeInIgnore` |

Sample rate, channel count, voice, and TTS output format are configured on the STT and TTS clients you pass to `voice.New`. The voice package does not redeclare them.

## Audio transport

`voice.AudioTransport` is the duplex audio channel for a conversation. Implementations adapt WebSockets, telephony streams, in-memory pipes (for tests), files, etc.

```go
type AudioTransport interface {
    Read(ctx context.Context) ([]byte, error)
    Write(ctx context.Context, frame []byte) error
    Close() error
}
```

`Read` returns one mono PCM frame per call. `Write` is called for every TTS audio frame. The PCM encoding (sample rate, channel count) is whatever the consumer configured on the STT and TTS clients; the voice package does not inspect the bytes. The example at `examples/voice/web` adapts a `coder/websocket` connection.

## Events

`Conversation.Events()` returns a channel of typed events. The channel closes when the conversation ends. The consumer must drain it; failing to do so blocks the pipeline.

| Type | Fired when | Populated fields |
|------|-----------|-------------------|
| `EventReady` | Conversation is ready to receive audio | (none beyond `Type`, `Timestamp`) |
| `EventUserTranscriptPartial` | STT emits an interim result | `Text` |
| `EventUserTranscriptFinal` | STT commits a final transcript | `Text` |
| `EventAssistantDelta` | LLM streams a content token | `Text` |
| `EventAssistantDone` | The current assistant turn ends | `Text` (full reply) |
| `EventToolCallStart` | A tool is about to run | `ToolCall` |
| `EventToolCallEnd` | A tool finished running | `ToolCall`, `ToolResult` |
| `EventTTSStarted` | A TTS stream opens for this turn | (none) |
| `EventTTSEnded` | The TTS stream for this turn drains | (none) |
| `EventFiller` | A filler phrase has been queued for TTS | `Text` (the spoken filler) |
| `EventToolSoundStart` | Ambient tool-sound looper has started for a tool batch | (none) |
| `EventToolSoundEnd` | Ambient tool-sound looper has stopped | (none) |
| `EventUserSpeechStart` | First non-empty STT partial of a new user utterance | (none) |
| `EventAgentInterrupted` | Barge-in fired and the current turn was canceled | `Text` (spoken-so-far approximation) |
| `EventConversationEnd` | The pipeline goroutines have all returned | (none) |
| `EventError` | An unrecoverable error terminated the conversation | `Error` |

## Filler audio

If the LLM takes a long time to produce its first content delta (slow first token, slow tool-call resolution before any visible text, etc.), the user hears silence. `WithFiller` mitigates that by speaking a short phrase after a configurable timeout. It's modeled on the `soft_timeout_config` from ElevenAgents.

```go
voice.WithFiller(voice.FillerConfig{
    Timeout: 1500 * time.Millisecond,
    Message: "One moment.",
})
```

`FillerConfig` fields:

| Field | Description |
|---|---|
| `Timeout` | How long to wait before speaking the filler. A non-positive value disables the feature. |
| `Message` | Static phrase spoken when `Timeout` fires. Required when `Source` is nil. Also serves as the fallback when `Source` errors or returns an empty string. |
| `Source` | Optional `FillerSource` callback that generates the filler dynamically from the conversation history. |
| `SourceDeadline` | Caps how long `Source` may run before falling back to `Message`. Defaults to 800ms. Ignored when `Source` is nil. |

**Dynamic filler example** — generate context-aware fillers via a small fast LLM:

```go
voice.WithFiller(voice.FillerConfig{
    Timeout:        1500 * time.Millisecond,
    Message:        "One moment.",        // fallback
    SourceDeadline: 600 * time.Millisecond,
    Source: func(ctx context.Context, history []message.Message) (string, error) {
        resp, err := fastLLM.SendMessages(ctx, append(history,
            message.NewUserMessage(
                "In one short spoken phrase (under six words), say something to fill silence while you think. No greetings.",
            ),
        ), nil)
        if err != nil {
            return "", err
        }
        return resp.Content, nil
    },
})
```

The filler is only spoken when the TTS client implements `tts.StreamingTextProvider`. Single-shot TTS providers can't speak a filler during the LLM wait — they need the full LLM output buffered first. ElevenLabs and Deepgram TTS both implement the streaming-text-input interface; OpenAI TTS does not.

The filler bypasses sentence-boundary chunking and is sent to TTS as a single phrase so it's spoken immediately. After it plays, the real assistant response continues in the same TTS stream.

## Tool call sounds

While a tool is executing, the conversation goes silent: the LLM has emitted its tool-call decision and is waiting for the result before producing more audio. `WithToolSound` fills that gap with ambient pre-recorded audio that loops until the tool returns. Modeled on ElevenAgents' `tool_call_sound`.

```go
voice.WithToolSound(voice.ToolSoundConfig{
    Audio:    pcmClipBytes,            // PCM 16-bit LE mono at the TTS sample rate
    Behavior: voice.ToolSoundAlways,
})
```

`ToolSoundConfig` fields:

| Field | Description |
|---|---|
| `Audio` | The PCM clip to loop. Format must match what the configured `tts.Generation` client emits (typically signed 16-bit little-endian PCM mono at 16 kHz). Empty disables tool sound entirely. |
| `Behavior` | `ToolSoundAuto` (default) plays only if the agent emitted spoken content in the same iteration before the tool call. `ToolSoundAlways` plays on every tool invocation. |

The `Auto` behavior maps to the natural case where the agent says something like "let me check that" before invoking a tool — the looped sound feels like a continuation. With `Always`, the sound plays even when the LLM goes straight to a tool with no preamble.

The looper sends 100ms PCM chunks into the same audio sink the TTS pipeline uses, so the sound rides through `AudioTransport.Write` exactly like spoken audio. When the tool returns, the looper is canceled; the next iteration's TTS audio plays normally after a brief tail (a single buffered chunk may trail out before the looper goroutine drains).

The package does not bundle audio assets. Convert your own clip with e.g.:

```bash
ffmpeg -i input.wav -f s16le -ar 16000 -ac 1 output.pcm
```

then `//go:embed output.pcm` it into your binary. The `examples/voice/web` example synthesizes a short typing-like loop programmatically so it works without any external assets.

## Barge-in

When the user starts speaking while the agent is mid-reply, the agent should stop talking, throw away the in-flight LLM/TTS work, and listen. `WithBargeIn` controls this. Modeled on ElevenAgents' interruption toggle.

```go
voice.WithBargeIn(voice.BargeInInterrupt)
```

`BargeInPolicy` values:

| Value | Description |
|---|---|
| `BargeInIgnore` (default) | Agent finishes whatever it's saying; STT partials during agent speech are observed but cause no action. |
| `BargeInInterrupt` | Cancels the current LLM/TTS turn the moment STT emits a non-empty partial during agent speech. |

**Detection:** the package leans on the STT's own VAD. The first non-empty partial of a new user utterance fires `EventUserSpeechStart`. If `BargeInInterrupt` is set and the agent is currently speaking (between `EventTTSStarted` and `EventTTSEnded`), the package fires `EventAgentInterrupted` with the spoken-so-far text and cancels the per-turn context.

**What gets canceled:** the in-flight `llm.LLM.StreamResponse`, the sentence chunker, the TTS WebSocket. Any tool execution running for the canceled iteration also unwinds via the per-turn context.

**Server-side audio drop:** queued PCM frames in the runner's `ttsAudio` channel are discarded (not written to `AudioTransport`) until the next turn's TTS opens. This prevents a multi-second tail of already-synthesized audio from leaking into the next turn.

**Browser-side audio flush:** the consumer is responsible for cutting the playback queue as soon as `EventAgentInterrupted` arrives. The example handles this by closing and recreating the `AudioContext`:

```ts
case "agent_interrupted":
  void player?.flush();
  // ...
```

**History truncation:** when barge-in fires, the runner appends a truncated assistant message to the in-memory history of the form `<spoken-so-far> [interrupted]`. The next LLM call then knows what the user actually heard versus what was planned.

The detection is binary: any non-empty partial during agent speech triggers the interrupt. There's no minimum-words or confidence threshold knob yet — STT vendors with sensitive VAD may produce false positives on small noises. If that becomes a problem, raise it in an issue and we'll add a sensitivity option.

## How it works

The conversation runs four goroutines coordinated by an `errgroup`:

1. **Audio in** reads frames from `AudioTransport.Read` and feeds them into the STT stream.
2. **STT consumer** drains `<-chan stt.StreamResult`. Partial results emit `EventUserTranscriptPartial`. On `IsFinal=true`, it emits `EventUserTranscriptFinal` and triggers the turn driver.
3. **Turn driver** runs `runAssistantTurn`: calls `llm.LLM.StreamResponse`, type-asserts the TTS client for `tts.StreamingTextProvider`, opens TTS lazily on the first content delta. Tool calls are executed sequentially up to `WithMaxToolIterations`.
4. **Audio out** drains TTS audio frames and writes them to `AudioTransport.Write`.

Sentence chunking (in the streaming-text path) splits text on `.`, `!`, `?`, `\n` followed by whitespace, with a 12-rune minimum length floor so fragments like `"Mr."` or `"1."` keep accumulating.

## Termination

Cancel the context passed to `StartConversation` to terminate the conversation. The runner closes its goroutines, drains the STT stream, calls `AudioTransport.Close`, and emits `EventConversationEnd` followed by closing `Events()`. `Wait()` then returns nil (or any unrecoverable error encountered).

## Not in this slice

The first slice ships the streaming pipeline with tool calls. Future work covers barge-in / interruption, memory injection, agent transfer (handoffs), filler audio while slow tools run, idle and max-duration timeouts, and voice-specific hooks. None of those are present yet.

## Example

See [`examples/voice/web`](https://github.com/JoakimCarlsson/ai/tree/main/examples/voice/web) for an end-to-end browser demo: a Vite + TypeScript frontend captures mic audio, streams it to a Go server over WebSocket, and plays back the agent's audio response.
