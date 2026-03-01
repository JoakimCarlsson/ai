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
