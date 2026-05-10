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
| `WithSession(id, store)` | Persist conversation history to a `session.Store`; load on start, append at turn boundaries | disabled |
| `WithContextStrategy(strategy, maxTokens)` | Trim, slide, or summarize the message list before each LLM call when it exceeds `maxTokens` | disabled |
| `WithHooks(hooks...)` | Synchronous interception points (mutate / deny / observe) at user-message commit, LLM call, tool use, lifecycle | disabled |
| `WithHandoffs(configs...)` | Register `transfer_to_<Name>` tools that swap the active agent mid-conversation | disabled |

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

## Session persistence

By default the conversation lives only in memory. `WithSession` plugs in any `session.Store` (in-memory, file, or your own) so history survives across reconnects within the same process — and across server restarts when the store is durable.

```go
import "github.com/joakimcarlsson/ai/session"

agent := voice.New(llm, stt, tts,
    voice.WithSystemPrompt("..."),
    voice.WithSession("user-42", session.MemoryStore()),
)
```

It mirrors `agent.WithSession` exactly — same `session.Store` and `session.Session` interfaces. Drop in `session.MemoryStore()`, the file store from `session/file.go`, or any custom implementation.

**Behavior:**

- At conversation start, the runner calls `store.Load(ctx, id)` (or `store.Create` if new) and uses the stored messages as the starting history. If the session is empty and a system prompt is configured, the system prompt is persisted as the first message.
- New messages added during a turn — user, assistant + tool calls per iteration, tool results, final assistant reply — are persisted in a single `session.AddMessages` call once the turn finishes (or once the barge-in branch appends its truncated reply).
- A store error surfaces as `EventError` and ends the conversation, the same as any other unrecoverable error.

**Constraints:**

- One session id per `VoiceAgent`. Concurrent conversations writing to the same id is not supported (last writer wins) — use one agent per id.
- Persistence is batched per turn, not message-by-message. A crash mid-turn loses that turn's tail; the user message survives because it's appended just before the turn opens.

## Context-window management

Voice conversations grow without bound. With `WithSession` enabled the history can re-load from disk at thousands of messages. Without management every LLM call eventually hits the model's context window and fails. `WithContextStrategy` plugs in a `tokens.Strategy` that runs **before every LLM call**: it counts tokens, and if the conversation exceeds the budget it trims, slides, or summarizes.

```go
import (
    "github.com/joakimcarlsson/ai/tokens/sliding"
    "github.com/joakimcarlsson/ai/tokens/truncate"
    "github.com/joakimcarlsson/ai/tokens/summarize"
)

// Drop oldest messages until we fit.
voice.WithContextStrategy(truncate.Strategy(), 8000)

// Keep only the last N messages.
voice.WithContextStrategy(sliding.Strategy(sliding.KeepLast(40)), 8000)

// Replace older messages with an LLM-generated summary.
voice.WithContextStrategy(summarize.Strategy(summaryLLM), 8000)
```

Mirrors `agent.WithContextStrategy` exactly — same `tokens.Strategy` interface, same three strategies in `tokens/sliding`, `tokens/truncate`, `tokens/summarize`.

**Behavior:**

- The strategy is invoked at the top of `streamLLMAndSpeak`, once per LLM iteration (so a turn that fans out into multiple tool calls runs it multiple times). Its result (`Messages`) is what's sent to the LLM; the agent's full in-memory history is left untouched, except for the SessionUpdate folding described below.
- If the strategy returns a `SessionUpdate.AddMessages` (typically a single summary message), those messages are appended to the live history. The runner's per-turn persist step then writes them to the configured `session.Store` together with the rest of the turn's new messages — no double-write.
- When `maxContextTokens` is left at zero, the option falls back to `model.ContextWindow - 4096` (4096 reserved for output). When the model's context window is also unknown, the strategy is silently skipped — same as if it weren't configured.
- A strategy error surfaces as `EventError` and ends the conversation.

**Picking a strategy:**

- `truncate` — cheapest, no LLM cost, but loses early context outright.
- `sliding` — keep last N messages. Good when older context rarely matters; predictable.
- `summarize` — preserves long-range gist via an LLM-generated summary; costs an extra LLM call when it fires. Good for long support / agent-style calls where early context (user identity, ticket id, etc.) keeps mattering.

## Hooks

`Conversation.Events()` is async, fire-and-forget, observe-only. `WithHooks` registers synchronous callbacks that fire at well-defined interception points and can **allow**, **deny**, or **modify** the in-flight values. Use hooks when you need to mutate or veto; use Events when you just need to watch.

```go
voice.WithHooks(voice.Hooks{
    OnUserMessage: func(_ context.Context, uc voice.UserMessageContext) (voice.UserMessageResult, error) {
        if containsPII(uc.Text) {
            return voice.UserMessageResult{
                Action:     voice.HookDeny,
                DenyReason: "user message contained PII",
            }, nil
        }
        return voice.UserMessageResult{Action: voice.HookAllow}, nil
    },
    PreToolUse: func(_ context.Context, tc voice.ToolUseContext) (voice.PreToolUseResult, error) {
        return voice.PreToolUseResult{
            Action: voice.HookModify,
            Input:  redact(tc.Input),
        }, nil
    },
})
```

Multiple `Hooks` structs can be passed; they run in registration order and `HookModify` mutations chain (later hooks see earlier ones' edits).

| Callback | Fires | Capability |
|---|---|---|
| `OnConversationStart` | once when the runner begins | observe |
| `OnConversationEnd` | once when the runner returns | observe |
| `OnUserMessage` | after STT commits a final transcript, before history append | allow / deny / modify |
| `PreModelCall` | before every LLM call (after context-window strategy) | allow / modify |
| `PostModelCall` | after every LLM call returns or errors | observe |
| `PreToolUse` | before every tool invocation | allow / deny / modify |
| `PostToolUse` | after every successful tool invocation | allow / modify |
| `OnToolError` | when a tool run errors | allow / modify (recover) |

**Deny semantics:** `OnUserMessage` deny drops the user turn entirely and surfaces an `EventError` wrapping `voice.ErrUserMessageDenied`. `PreToolUse` deny skips the tool execution and writes a tool-result message carrying the deny reason as content with `IsError=true`, so the LLM sees a structured refusal it can react to.

**Modify semantics:** the modified values flow forward — modified user text reaches history and the LLM; modified `Messages` / `Tools` reach the LLM; modified tool input reaches the tool; modified tool output replaces what lands in history. `OnToolError` modify additionally clears the error flag so downstream sees a successful tool result.

Hooks have no `OnEvent` mass-fanout — that's what `Conversation.Events()` is for.

## Handoffs

`WithHandoffs` lets the LLM transfer control to another `VoiceAgent` mid-conversation. Each `HandoffConfig` registers a `transfer_to_<Name>` tool on the source agent. When the LLM calls it, the runner swaps the active agent for the rest of the conversation: target's system prompt, tools, LLM, hooks, context strategy, and (chained) handoffs all take over. Subsequent user turns continue with the new agent. Mirrors `agent.WithHandoffs`.

```go
specialist := voice.New(llm, stt, tts,
    voice.WithSystemPrompt("You are a billing specialist."),
    voice.WithTools(issueRefundTool{}),
)

triage := voice.New(llm, stt, tts,
    voice.WithSystemPrompt("You answer general questions and transfer billing questions."),
    voice.WithHandoffs(voice.HandoffConfig{
        Name:        "billing",
        Description: "Use this when the user asks about charges, refunds, or invoices.",
        Agent:       specialist,
    }),
)
```

The triage agent's tool list will include `transfer_to_billing` — described to the LLM via the `Description` field. When the user says something money-related, the LLM calls the tool; the runner detects it, rebuilds history with the specialist's system prompt prepended (old system messages stripped, all non-system messages preserved), and continues. The "Transferring to billing" tool result lands in history so the specialist's first reply has full context.

**Constraints (v1):**

- **STT/TTS stay bound to the original agent.** The target's STT/TTS clients are ignored — the audio path doesn't blink at the transfer boundary, but the agent voice is the same. Different voices per agent require restarting the audio streams; future slice.
- **Session is untouched.** Handoff modifies the runner's in-memory history only. The session retains the original system prompt at position 0. On reconnect to a session that experienced a handoff, the user resumes with the original agent (carrying the post-handoff history).
- **Pre/post-handoff hooks** aren't dedicated callbacks. The handoff is a regular tool, so it goes through `PreToolUse` / `PostToolUse` like any other tool — observe and deny work today.

Chained handoffs are supported: A → B → C if B has its own `WithHandoffs(C)`. The runner walks the chain in a single conversation.

See [`examples/voice/handoff`](https://github.com/JoakimCarlsson/ai/tree/main/examples/voice/handoff) for an end-to-end demo.

## How it works

The conversation runs four goroutines coordinated by an `errgroup`:

1. **Audio in** reads frames from `AudioTransport.Read` and feeds them into the STT stream.
2. **STT consumer** drains `<-chan stt.StreamResult`. Partial results emit `EventUserTranscriptPartial`. On `IsFinal=true`, it emits `EventUserTranscriptFinal` and triggers the turn driver.
3. **Turn driver** runs `runAssistantTurn`: calls `llm.LLM.StreamResponse`, type-asserts the TTS client for `tts.StreamingTextProvider`, opens TTS lazily on the first content delta. Tool calls are executed sequentially up to `WithMaxToolIterations`.
4. **Audio out** drains TTS audio frames and writes them to `AudioTransport.Write`.

Sentence chunking (in the streaming-text path) splits text on `.`, `!`, `?`, `\n` followed by whitespace, with a 12-rune minimum length floor so fragments like `"Mr."` or `"1."` keep accumulating.

## Termination

Cancel the context passed to `StartConversation` to terminate the conversation. The runner closes its goroutines, drains the STT stream, calls `AudioTransport.Close`, and emits `EventConversationEnd` followed by closing `Events()`. `Wait()` then returns nil (or any unrecoverable error encountered).

## Not yet covered

Memory injection (`agent.WithMemory` parity), agent transfer (handoffs), context-window strategies, idle / max-duration timeouts, and voice-specific lifecycle hooks. None of those are present yet.

## Example

See [`examples/voice/web`](https://github.com/JoakimCarlsson/ai/tree/main/examples/voice/web) for an end-to-end browser demo: a Vite + TypeScript frontend captures mic audio, streams it to a Go server over WebSocket, and plays back the agent's audio response.
