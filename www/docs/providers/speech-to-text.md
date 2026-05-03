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

Check `client.SupportsStreaming()` before calling `StreamTranscribe`. Providers that don't support streaming (currently OpenAI Whisper, Google Cloud) return `transcription.ErrStreamingNotSupported`.

```go
client, err := transcription.NewSpeechToText(
    model.ProviderDeepgram,
    transcription.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
    transcription.WithModel(model.DeepgramTranscriptionModels[model.DeepgramNova3]),
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
    transcription.WithStreamInterimResults(true),
    transcription.WithStreamEndpointing(300),
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

### Streaming Options

| Option | Description |
|--------|-------------|
| `WithStreamSampleRate(hz int)` | PCM sample rate of the audio fed in (default 16000) |
| `WithStreamChannels(n int)` | Channel count (default 1) |
| `WithStreamInterimResults(bool)` | Emit non-final transcripts (default true) |
| `WithStreamEndpointing(ms int)` | Silence window before final emission |
| `WithDeepgramStreamEndpointingMs(ms int)` | Deepgram-specific endpointing |
| `WithAssemblyAIEndOfTurnSilenceMs(ms int)` | AssemblyAI v3 end-of-turn silence threshold |
| `WithElevenLabsStreamVADSilenceMs(ms int)` | ElevenLabs Scribe v2 VAD silence threshold |
| `WithOpenAIRealtimeVADSilenceMs(ms int)` | OpenAI Realtime server VAD silence threshold |

### Provider Streaming Support

| Provider | Streaming | Protocol | Endpoint |
|---|---|---|---|
| Deepgram | ✓ | WebSocket | `wss://api.deepgram.com/v1/listen` |
| AssemblyAI | ✓ | WebSocket (v3 Universal Streaming) | `wss://streaming.assemblyai.com/v3/ws` |
| ElevenLabs Scribe v2 | ✓ | WebSocket (base64 JSON audio) | `wss://api.elevenlabs.io/v1/speech-to-text/realtime` |
| OpenAI (Realtime, transcription mode) | ✓ | WebSocket (base64 JSON audio, **24 kHz only**) | `wss://api.openai.com/v1/realtime?intent=transcription` |
| Google Cloud STT v2 | — | gRPC streaming | not yet implemented; returns `ErrStreamingNotSupported` |

For OpenAI streaming you must select a Realtime-compatible transcription model (`GPT4oTranscribe` or `GPT4oMiniTranscribe`); the legacy Whisper model is batch-only and `StreamTranscribe` will fail at the upstream layer if it's wired against `Whisper1`.

See the runnable examples:

- `example/transcription_deepgram_stream/`
- `example/transcription_assemblyai_stream/`
- `example/transcription_elevenlabs_stream/`
- `example/transcription_openai_stream/`
