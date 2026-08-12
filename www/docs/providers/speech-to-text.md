# Speech-to-Text (STT)

The `stt` modality. Each native vendor lives under `stt/`.

## Basic transcription

OpenAI Whisper:

```go
import (
    sttopenai "github.com/joakimcarlsson/ai/stt/openai"
    "github.com/joakimcarlsson/ai/stt"
)

client := sttopenai.NewSpeechToText(
    sttopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    sttopenai.WithModel(sttopenai.Models[sttopenai.Whisper1]),
)

audio, _ := os.ReadFile("audio.mp3")
resp, err := client.Transcribe(ctx, audio,
    stt.WithLanguage("en"),
)
fmt.Println(resp.Text)
```

Deepgram, AssemblyAI, Google Cloud Speech, ElevenLabs Scribe follow the same
shape via their respective vendor packages.

Berget AI (EU-hosted, OpenAI-compatible Whisper; pricing in EUR) is the same
shape via `stt/berget`, with Swedish (`BergetKBWhisperLarge`) and Norwegian
(`BergetNBWhisperLarge`) fine-tunes alongside `BergetFasterWhisperLargeV3`:

```go
import sttberget "github.com/joakimcarlsson/ai/stt/berget"

client := sttberget.NewSpeechToText(
    sttberget.WithAPIKey(os.Getenv("BERGET_API_KEY")),
    sttberget.WithModel(berget.Models[berget.KBWhisperLarge]),
)
```

OpenRouter is the same shape via `stt/openrouter`, reaching Whisper, GPT-4o
Transcribe, Voxtral, Fish Audio and Grok STT on one key:

```go
import sttopenrouter "github.com/joakimcarlsson/ai/stt/openrouter"

client := sttopenrouter.NewSpeechToText(
    sttopenai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    sttopenai.WithModel(openrouter.Models[openrouter.Whisper1]),
)
```

Two OpenRouter specifics. `response_format="verbose_json"` — and with it the
`Segments` and `Words` arrays — is only accepted by the OpenAI-compatible
upstreams (OpenAI, Groq, Together); the rest answer HTTP 400, so pass
`stt.WithResponseFormat("json")` when routing elsewhere. And OpenRouter
publishes no `/audio/translations` route, so `Translate` returns
`sttopenrouter.ErrTranslationNotSupported` instead of issuing a doomed request.
`openrouter.Models` carries 13 known-good defaults. OpenRouter
routes more than that, so for anything it does not define, `WithModelID` takes a
raw id:

```go
client := sttopenrouter.NewSpeechToText(
    sttopenai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    sttopenrouter.WithModelID("nvidia/parakeet-tdt-0.6b-v3"),
)

resp, err := client.Transcribe(ctx, audio, stt.WithResponseFormat("json"))
```

`Model()` then reports only the id and provider — no cost, no capability flags.
Pass a hand-built `stt.TranscriptionModel` to `sttopenai.WithModel` instead
when something downstream reads those fields.

## Translation (OpenAI only)

```go
resp, err := client.Translate(ctx, audio)  // returns English translation
```

## Streaming transcription

Deepgram, AssemblyAI, and ElevenLabs support real-time streaming over
WebSocket. The `stt.SpeechToText` interface exposes `StreamTranscribe`:

```go
import sttdeepgram "github.com/joakimcarlsson/ai/stt/deepgram"

client := sttdeepgram.NewSpeechToText(
    sttdeepgram.WithAPIKey(os.Getenv("DEEPGRAM_API_KEY")),
    sttdeepgram.WithModel(deepgram.Models[deepgram.Nova3]),
    sttdeepgram.WithSmartFormat(true),
    sttdeepgram.WithStreamInterimResults(true),
)

audioCh := make(chan []byte, 16)
go feedAudio(audioCh)  // your PCM frame source

results, err := client.StreamTranscribe(ctx, audioCh,
    stt.WithSampleRate(16000),
    stt.WithChannels(1),
)
if err != nil {
    log.Fatal(err)
}

for r := range results {
    if r.Error != nil {
        log.Fatal(r.Error)
    }
    if r.IsFinal {
        fmt.Println("FINAL:", r.Text)
    } else {
        fmt.Println("interim:", r.Text)
    }
}
```

## SupportsStreaming

For vendors that don't stream:

```go
if !client.SupportsStreaming() {
    // fall back to batch Transcribe
    resp, _ := client.Transcribe(ctx, audio)
}
```

`stt/openai` and `stt/google` return `false`. The streaming-capable vendors
(`stt/deepgram`, `stt/assemblyai`, `stt/elevenlabs`) return `true`.

## Per-call options

```go
stt.WithLanguage("en")
stt.WithPrompt("Domain-specific words: Claude, Anthropic, ...")
stt.WithResponseFormat("verbose_json")  // OpenAI
stt.WithTimestampGranularities("word", "segment")
stt.WithFilename("audio.wav")           // for format detection
stt.WithSampleRate(16000)               // streaming only
stt.WithChannels(1)                     // streaming only
```

## Vendor-specific options

Deepgram:

```go
sttdeepgram.WithPunctuate(true)
sttdeepgram.WithDiarize(true)
sttdeepgram.WithSmartFormat(true)
sttdeepgram.WithKeyterms("Claude", "Anthropic")
sttdeepgram.WithStreamEndpointingMs(300)
```

AssemblyAI:

```go
sttassemblyai.WithSpeakerLabels(true)
sttassemblyai.WithStreamEndOfTurnSilenceMs(700)
sttassemblyai.WithStreamFormatTurns(true)
```

ElevenLabs Scribe:

```go
sttelevenlabs.WithDiarize(true)
sttelevenlabs.WithNumSpeakers(2)
sttelevenlabs.WithStreamLanguageCode("en")
sttelevenlabs.WithStreamIncludeTimestamps(true)
```
