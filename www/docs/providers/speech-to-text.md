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
