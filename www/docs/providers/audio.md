# Text-to-Speech (TTS)

The `tts` modality (formerly "audio generation"). Each native vendor lives
under `tts/`.

## Basic usage

ElevenLabs:

```go
import (
    "github.com/joakimcarlsson/ai/model"
    "github.com/joakimcarlsson/ai/tts"
    ttselevenlabs "github.com/joakimcarlsson/ai/tts/elevenlabs"
)

client := ttselevenlabs.NewGeneration(
    ttselevenlabs.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    ttselevenlabs.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
    ttselevenlabs.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),  // Rachel
)

resp, err := client.GenerateAudio(ctx, "Hello, how are you today?")
os.WriteFile("output.mp3", resp.AudioData, 0644)
```

OpenAI:

```go
import ttsopenai "github.com/joakimcarlsson/ai/tts/openai"

client := ttsopenai.NewGeneration(
    ttsopenai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    ttsopenai.WithModel(model.OpenAIAudioModels[model.TTS1HD]),
    ttsopenai.WithVoice("nova"),
    ttsopenai.WithOutputFormat("mp3"),
)
```

Google Cloud, Azure Speech, Deepgram Aura follow the same shape.

OpenRouter is a thin wrapper over `tts/openai` that fixes the base URL, so one
key reaches OpenAI, Google, Mistral, Microsoft and Deepgram voices:

```go
import ttsopenrouter "github.com/joakimcarlsson/ai/tts/openrouter"

client := ttsopenrouter.NewGeneration(
    ttsopenai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    ttsopenai.WithModel(model.OpenRouterAudioModels[model.OpenRouterMAIVoice2]),
    ttsopenai.WithVoice("en-US-Harper:MAI-Voice-2"),
    ttsopenai.WithOutputFormat("mp3"),
)
```

There is no model-fallback option: OpenRouter documents its `models` fallback
array for chat completions only, and the audio endpoints ignore request fields
they do not recognise, so sending it would look like it worked while doing
nothing. `WithProviderRouting` is documented for `/audio/speech` and is wired up.

Two OpenRouter specifics: `response_format` defaults to `pcm` rather than
OpenAI's `mp3` and the only documented values are `mp3` and `pcm`; and `speed`
is honored only by upstreams that support it and silently ignored by the rest.
`model.OpenRouterAudioModels` carries 18 known-good defaults. OpenRouter routes
more than that, so for anything it does not define, `WithModelID` takes a raw
id:

```go
client := ttsopenrouter.NewGeneration(
    ttsopenai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    ttsopenrouter.WithModelID("minimax/speech-2.8-hd"),
    ttsopenai.WithOutputFormat("mp3"),
)
```

`Model()` then reports only the id and provider — no per-character cost, no
format list. Pass a hand-built `model.AudioModel` to `ttsopenai.WithModel`
instead when something downstream reads those fields.

## Streaming

ElevenLabs and Deepgram stream chunked audio:

```go
chunks, err := client.StreamAudio(ctx, "Hello world",
    tts.WithOptimizeStreamingLatency(3),
)

for chunk := range chunks {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    if chunk.Done {
        break
    }
    output.Write(chunk.Data)
}
```

The other vendors (`tts/openai`, `tts/google`, `tts/azure`) buffer the
non-streaming response into a single chunk for API parity.

## Voice listing

```go
voices, err := client.ListVoices(ctx)
for _, v := range voices {
    fmt.Printf("%s — %s (%s)\n", v.VoiceID, v.Name, v.Category)
}
```

## ElevenLabs voice settings

```go
resp, err := client.GenerateAudio(ctx, "Expressive line",
    tts.WithStability(0.75),
    tts.WithSimilarityBoost(0.85),
    tts.WithStyle(0.5),
    tts.WithSpeakerBoost(true),
)
```

## ElevenLabs alignment

`tts/elevenlabs.Client` also implements `tts.ForcedAlignmentProvider`. The
canonical alignment-enabled call:

```go
resp, err := client.GenerateAudio(ctx, "Hello world",
    tts.WithAlignmentEnabled(true),
)

for i, ch := range resp.Alignment.Characters {
    fmt.Printf("%s: %.2fs - %.2fs\n",
        ch,
        resp.Alignment.CharacterStartTimesSeconds[i],
        resp.Alignment.CharacterEndTimesSeconds[i],
    )
}
```

For aligning existing audio against a transcript:

```go
if fap, ok := client.(tts.ForcedAlignmentProvider); ok {
    audio, _ := os.ReadFile("recording.mp3")
    align, err := fap.GenerateForcedAlignment(ctx, audio,
        "the spoken transcript")

    for _, w := range align.Words {
        fmt.Printf("%s: %.2fs - %.2fs (loss=%.4f)\n",
            w.Text, w.Start, w.End, w.Loss)
    }
}
```

The type assertion succeeds against the wrapper returned from
`ttselevenlabs.NewGeneration` because the wrapper preserves the optional
sub-interface when the inner concrete client implements it.

## Common per-call options

```go
tts.WithOutputFormat("mp3_44100_128")   // ElevenLabs
tts.WithOutputFormat("LINEAR16")        // Google Cloud
tts.WithStability(0.75)
tts.WithSimilarityBoost(0.85)
tts.WithStyle(0.5)
tts.WithSpeakerBoost(true)
tts.WithOptimizeStreamingLatency(3)
tts.WithAlignmentEnabled(true)
```
