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
