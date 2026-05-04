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
