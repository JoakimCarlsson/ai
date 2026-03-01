# Audio Generation (Text-to-Speech)

## Basic Usage

```go
import (
    "github.com/joakimcarlsson/ai/audio"
    "github.com/joakimcarlsson/ai/model"
)

client, err := audio.NewAudioGeneration(
    model.ProviderElevenLabs,
    audio.WithAPIKey("your-api-key"),
    audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
)
if err != nil {
    log.Fatal(err)
}

response, err := client.GenerateAudio(
    context.Background(),
    "Hello! This is a demonstration of text-to-speech.",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
)
if err != nil {
    log.Fatal(err)
}

os.WriteFile("output.mp3", response.AudioData, 0644)
fmt.Printf("Characters used: %d\n", response.Usage.Characters)
```

## Custom Voice Settings

```go
response, err := client.GenerateAudio(
    context.Background(),
    "This uses custom voice settings for enhanced expressiveness.",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
    audio.WithStability(0.75),              // 0.0-1.0, higher = more consistent
    audio.WithSimilarityBoost(0.85),        // 0.0-1.0, higher = more similar to original
    audio.WithStyle(0.5),                   // 0.0-1.0, higher = more expressive
    audio.WithSpeakerBoost(true),           // Enhanced speaker similarity
)
```

## Streaming Audio

```go
chunkChan, err := client.StreamAudio(
    context.Background(),
    "This is a streaming audio example.",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
    audio.WithOptimizeStreamingLatency(3), // 0-4, higher = lower latency
)
if err != nil {
    log.Fatal(err)
}

file, _ := os.Create("output_stream.mp3")
defer file.Close()

for chunk := range chunkChan {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    if chunk.Done {
        break
    }
    file.Write(chunk.Data)
}
```

## List Available Voices

```go
voices, err := client.ListVoices(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, voice := range voices {
    fmt.Printf("%s (%s) - %s\n", voice.Name, voice.VoiceID, voice.Category)
}
```

## Alignment Data

Enable character-level timing information for subtitles, word highlighting, or lip sync:

```go
response, err := client.GenerateAudio(
    context.Background(),
    "Hello, world!",
    audio.WithVoiceID("EXAVITQu4vr4xnSDxMaL"),
    audio.WithAlignmentEnabled(true),
)

// response.Alignment contains character-level timing
for i, char := range response.Alignment.Characters {
    fmt.Printf("%s: %.2fs - %.2fs\n", char,
        response.Alignment.CharacterStartTimesSeconds[i],
        response.Alignment.CharacterEndTimesSeconds[i],
    )
}
```

Alignment is also available per-chunk during streaming via `chunk.Alignment`.

## Forced Alignment

Match existing audio with a transcript to produce word-level timing data. The provider must implement the `ForcedAlignmentProvider` interface:

```go
if aligner, ok := client.(audio.ForcedAlignmentProvider); ok {
    audioData, _ := os.ReadFile("speech.mp3")
    result, err := aligner.GenerateForcedAlignment(ctx, audioData, "Hello, world!")

    for _, word := range result.Words {
        fmt.Printf("%s: %.2fs - %.2fs\n", word.Text, word.Start, word.End)
    }
}
```

## Generation Options

| Option | Description |
|--------|-------------|
| `WithVoiceID(id)` | Voice to use for generation |
| `WithOutputFormat(fmt)` | Audio format (`mp3_44100_128`, `pcm_16000`, etc.) |
| `WithStability(f)` | Voice consistency, 0.0–1.0 |
| `WithSimilarityBoost(f)` | Match to original voice, 0.0–1.0 |
| `WithStyle(f)` | Style exaggeration, 0.0–1.0 |
| `WithSpeakerBoost(bool)` | Enhanced speaker similarity |
| `WithOptimizeStreamingLatency(n)` | Latency optimization level, 0–4 |
| `WithAlignmentEnabled(bool)` | Enable character-level timing data |

## Client Options

```go
client, err := audio.NewAudioGeneration(
    model.ProviderElevenLabs,
    audio.WithAPIKey("your-key"),
    audio.WithModel(model.ElevenLabsAudioModels[model.ElevenTurboV2_5]),
    audio.WithTimeout(30*time.Second),
    audio.WithElevenLabsOptions(
        audio.WithElevenLabsBaseURL("custom-endpoint"),
    ),
)
```
