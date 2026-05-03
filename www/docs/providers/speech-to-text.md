# Speech-to-Text (Transcription)

## Basic Transcription

```go
import (
    "github.com/joakimcarlsson/ai/transcription"
    "github.com/joakimcarlsson/ai/model"
)

client, err := transcription.NewSpeechToText(
    model.ProviderOpenAI,
    transcription.WithAPIKey("your-api-key"),
    transcription.WithModel(model.OpenAITranscriptionModels[model.Whisper1]),
)
if err != nil {
    log.Fatal(err)
}

audioData, err := os.ReadFile("audio.mp3")
if err != nil {
    log.Fatal(err)
}

response, err := client.Transcribe(context.Background(), audioData)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Text)
```

## Transcription with Options

```go
response, err := client.Transcribe(ctx, audioData,
    transcription.WithLanguage("en"),
    transcription.WithResponseFormat("verbose_json"),
    transcription.WithTimestampGranularities("word", "segment"),
    transcription.WithTemperature(0.2),
)

for _, segment := range response.Segments {
    fmt.Printf("[%.2fs - %.2fs] %s\n", segment.Start, segment.End, segment.Text)
}

for _, word := range response.Words {
    fmt.Printf("%s (%.2fs) ", word.Word, word.Start)
}
```

## Translation (to English)

```go
response, err := client.Translate(ctx, audioData,
    transcription.WithPrompt("Translate this Swedish audio to English"),
)

fmt.Println(response.Text)
```

## Client Options

```go
client, err := transcription.NewSpeechToText(
    model.ProviderOpenAI,
    transcription.WithAPIKey("your-key"),
    transcription.WithModel(model.OpenAITranscriptionModels[model.GPT4oTranscribe]),
    transcription.WithTimeout(30*time.Second),
)
```

## Deepgram

```go
client, err := transcription.NewSpeechToText(
    model.ProviderDeepgram,
    transcription.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
    transcription.WithModel(model.DeepgramTranscriptionModels[model.DeepgramNova3]),
    transcription.WithDeepgramOptions(
        transcription.WithDeepgramPunctuate(true),
        transcription.WithDeepgramSmartFormat(true),
    ),
)

response, err := client.Transcribe(ctx, audioData,
    transcription.WithLanguage("en"),
)
```

### Deepgram Options

| Option | Description |
|--------|-------------|
| `WithDeepgramPunctuate(bool)` | Enable automatic punctuation |
| `WithDeepgramDiarize(bool)` | Enable speaker diarization |
| `WithDeepgramSmartFormat(bool)` | Enable smart formatting (dates, numbers, etc.) |
| `WithDeepgramLanguage(string)` | Default language code |

**Models:** `DeepgramNova3`, `DeepgramNova2`

## AssemblyAI

AssemblyAI uses an async upload-and-poll model rather than streaming.

```go
client, err := transcription.NewSpeechToText(
    model.ProviderAssemblyAI,
    transcription.WithAPIKey(os.Getenv("ASSEMBLYAI_API_KEY")),
    transcription.WithModel(model.AssemblyAITranscriptionModels[model.AssemblyAIBest]),
    transcription.WithAssemblyAIOptions(
        transcription.WithAssemblyAISpeakerLabels(true),
    ),
)

response, err := client.Transcribe(ctx, audioData)
```

### AssemblyAI Options

| Option | Description |
|--------|-------------|
| `WithAssemblyAISpeakerLabels(bool)` | Enable speaker diarization |
| `WithAssemblyAIPollInterval(duration)` | Polling interval (default: 3s) |
| `WithAssemblyAIMaxPollDuration(duration)` | Max wait time (default: 5min) |

**Models:** `AssemblyAIBest`, `AssemblyAINano`

## Google Cloud

```go
client, err := transcription.NewSpeechToText(
    model.ProviderGoogleCloud,
    transcription.WithAPIKey(os.Getenv("GOOGLE_CLOUD_API_KEY")),
    transcription.WithModel(model.GoogleCloudTranscriptionModels[model.GoogleCloudSTTDefault]),
    transcription.WithGoogleCloudSTTOptions(
        transcription.WithGoogleCloudEncoding("LINEAR16"),
        transcription.WithGoogleCloudSampleRate(16000),
    ),
)

response, err := client.Transcribe(ctx, audioData,
    transcription.WithLanguage("en-US"),
)
```

### Google Cloud Options

| Option | Description |
|--------|-------------|
| `WithGoogleCloudEncoding(string)` | Audio encoding: `"LINEAR16"`, `"FLAC"`, `"OGG_OPUS"` |
| `WithGoogleCloudSampleRate(int)` | Sample rate in Hz (e.g., 16000) |
| `WithGoogleCloudLanguageCode(string)` | BCP-47 language code (default: `"en-US"`) |

**Models:** `GoogleCloudSTTDefault`, `GoogleCloudSTTLong`

## ElevenLabs

```go
client, err := transcription.NewSpeechToText(
    model.ProviderElevenLabs,
    transcription.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    transcription.WithModel(model.ElevenLabsTranscriptionModels[model.ElevenLabsScribeV2]),
    transcription.WithElevenLabsSTTOptions(
        transcription.WithElevenLabsDiarize(true),
    ),
)

response, err := client.Transcribe(ctx, audioData,
    transcription.WithLanguage("eng"),
)
```

### ElevenLabs Options

| Option | Description |
|--------|-------------|
| `WithElevenLabsDiarize(bool)` | Enable speaker diarization |
| `WithElevenLabsNumSpeakers(int)` | Expected speaker count (0-32) |
| `WithElevenLabsTimestampGranularity(string)` | Granularity: `"none"`, `"word"`, `"character"` |
| `WithElevenLabsTagAudioEvents(bool)` | Detect audio events (laughter, music, etc.) |

**Models:** `ElevenLabsScribeV1`, `ElevenLabsScribeV2`

## Streaming Transcription

`SpeechToText.StreamTranscribe` consumes a channel of PCM frames and emits a channel of `StreamResult` events as they arrive — interim transcripts as `IsFinal=false`, settled transcripts as `IsFinal=true`. Errors are sent as a final `StreamResult{Error: ...}` value before the channel closes.

Check `client.SupportsStreaming()` before calling `StreamTranscribe`. Providers that don't support streaming (OpenAI Whisper, Google Cloud) return `transcription.ErrStreamingNotSupported`.

```go
client, err := transcription.NewSpeechToText(
    model.ProviderDeepgram,
    transcription.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
    transcription.WithModel(model.DeepgramTranscriptionModels[model.DeepgramNova3]),
    transcription.WithDeepgramOptions(
        transcription.WithDeepgramStreamEndpointingMs(300),
        transcription.WithDeepgramPunctuate(true),
        transcription.WithDeepgramSmartFormat(true),
        transcription.WithDeepgramNumerals(true),
    ),
)
if err != nil {
    log.Fatal(err)
}
if !client.SupportsStreaming() {
    log.Fatal("provider does not support streaming")
}

audio := make(chan []byte, 64)
results, err := client.StreamTranscribe(ctx, audio,
    transcription.WithStreamSampleRate(16000),
    transcription.WithStreamChannels(1),
    transcription.WithLanguage("en-US"),
)
if err != nil {
    log.Fatal(err)
}

// Feed PCM16-LE frames into `audio`; close when done.
go func() {
    defer close(audio)
    for frame := range mic.Frames() {
        audio <- frame
    }
}()

for r := range results {
    if r.Error != nil {
        log.Printf("stream error: %v", r.Error)
        return
    }
    fmt.Printf("[final=%v conf=%.2f] %s\n", r.IsFinal, r.Confidence, r.Text)
}
```

### Generic Streaming Options

These describe the audio source and apply to every provider:

| Option | Description |
|--------|-------------|
| `WithStreamSampleRate(hz int)` | PCM sample rate of the audio fed in (default 16000) |
| `WithStreamChannels(n int)` | Channel count (default 1) |
| `WithLanguage(code string)` | Language hint for the transcription model |

Provider-specific streaming knobs go through the provider's own `WithXxxOptions(...)` wrapper at construction time, listed below.

### Deepgram Streaming Options

Pass via `WithDeepgramOptions(...)`:

| Option | Description |
|--------|-------------|
| `WithDeepgramStreamEndpointingMs(ms int)` | Silence window (ms) before `is_final` is emitted |
| `WithDeepgramStreamInterimResults(bool)` | Toggle interim transcripts (defaults to true) |
| `WithDeepgramVADEvents(bool)` | Emit SpeechStarted / UtteranceEnd events |
| `WithDeepgramNumerals(bool)` | Convert spoken numbers to digits ("nine hundred" → "900") |
| `WithDeepgramProfanityFilter(bool)` | Strip profanity from transcripts |
| `WithDeepgramDictation(bool)` | Format spoken punctuation commands |
| `WithDeepgramKeyterms(...string)` | Boost recognition of specific terms (Nova-3+) |
| `WithDeepgramKeywords(...string)` | Boost/suppress terms for older models; format `"term"` or `"term:intensifier"` |
| `WithDeepgramRedact(...string)` | Redact categories (e.g. `"pci"`, `"numbers"`, `"ssn"`) |
| `WithDeepgramSearch(...string)` | Acoustic pattern match for given terms |
| `WithDeepgramReplace(...string)` | Find/replace pairs in transcripts (`"find:replace"`) |

Plus the existing batch options (`WithDeepgramPunctuate`, `WithDeepgramDiarize`, `WithDeepgramSmartFormat`, `WithDeepgramLanguage`) which also work in streaming.

### AssemblyAI Streaming Options

Pass via `WithAssemblyAIOptions(...)`:

| Option | Description |
|--------|-------------|
| `WithAssemblyAIStreamSpeechModel(name string)` | Speech model. Valid: `"universal-streaming-english"` (default), `"universal-streaming-multilingual"`, `"whisper-rt"`, `"alpha-english"`, `"u3-rt-pro"`, `"u3-rt-agent"` |
| `WithAssemblyAIEndOfTurnSilenceMs(ms int)` | Silence threshold (ms) before end-of-turn fires |
| `WithAssemblyAIStreamFormatTurns(bool)` | Auto punctuation/casing on turn transcripts (defaults to true) |
| `WithAssemblyAIStreamEndOfTurnConfidenceThreshold(threshold float64)` | Confidence threshold (0.0–1.0) for end-of-turn detection |
| `WithAssemblyAIStreamMaxTurnSilenceMs(ms int)` | Cap longest silence within a turn before forcing end-of-turn |
| `WithAssemblyAIStreamKeyterms(...string)` | Boost recognition of specific terms |
| `WithAssemblyAIStreamPunctuationFilter(bool)` | Toggle punctuation filter |
| `WithAssemblyAIStreamWordFinalizationMaxWaitMs(ms int)` | Cap how long to wait before finalising last word of a turn |
| `WithAssemblyAIStreamExtraSessionInformation(bool)` | Enable additional session metadata events |

**Note**: AssemblyAI v3 requires audio chunks of 50–1000 ms. The streaming impl buffers smaller frames internally before sending.

### ElevenLabs Scribe v2 Streaming Options

Pass via `WithElevenLabsSTTOptions(...)`:

| Option | Description |
|--------|-------------|
| `WithElevenLabsStreamLanguageCode(code string)` | Language hint (e.g. `"eng"`) |
| `WithElevenLabsStreamKeyterms(...string)` | Boost recognition of specific terms |
| `WithElevenLabsStreamNoVerbatim(bool)` | Strip filler words ("um", "uh", …) |
| `WithElevenLabsStreamIncludeTimestamps(bool)` | Emit word-level timing data |
| `WithElevenLabsStreamIncludeLanguageDetection(bool)` | Auto-detect language |
| `WithElevenLabsStreamVADThreshold(threshold float64)` | VAD sensitivity (0.0–1.0) |
| `WithElevenLabsStreamMinSpeechDurationMs(ms int)` | Min duration (ms) of speech before VAD considers it valid |
| `WithElevenLabsStreamMinSilenceDurationMs(ms int)` | Min duration (ms) of silence before VAD declares end-of-speech |
| `WithElevenLabsStreamTimestampsGranularity(g string)` | Timestamp resolution: `"none"`, `"word"`, `"character"` |
| `WithElevenLabsStreamDisableLogging(bool)` | Opt out of server-side logging |

**Note**: ElevenLabs sends audio as base64-encoded PCM inside JSON `input_audio_chunk` messages (per their protocol — not raw binary).

### Provider Streaming Support

| Provider | Streaming | Protocol | Endpoint |
|---|---|---|---|
| Deepgram | ✓ | WebSocket | `wss://api.deepgram.com/v1/listen` |
| AssemblyAI | ✓ | WebSocket (v3 Universal Streaming) | `wss://streaming.assemblyai.com/v3/ws` |
| ElevenLabs Scribe v2 | ✓ | WebSocket (base64 JSON audio) | `wss://api.elevenlabs.io/v1/speech-to-text/realtime` |
| OpenAI Whisper | — | (no real-time-during-speech streaming) | returns `ErrStreamingNotSupported` |
| Google Cloud STT v2 | — | gRPC streaming | not yet implemented; returns `ErrStreamingNotSupported` |

See the runnable examples:

- `example/transcription_deepgram_stream/`
- `example/transcription_assemblyai_stream/`
- `example/transcription_elevenlabs_stream/`
