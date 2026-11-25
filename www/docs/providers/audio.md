# Text-to-Speech (TTS)

The `tts` modality (formerly "audio generation"). Each native vendor lives
under `tts/`.

## Basic usage

ElevenLabs:

```go
import (
    ttselevenlabs "github.com/joakimcarlsson/ai/tts/elevenlabs"
    "github.com/joakimcarlsson/ai/tts"
)

client := ttselevenlabs.NewGeneration(
    ttselevenlabs.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    ttselevenlabs.WithModel(ttselevenlabs.Models[ttselevenlabs.TurboV2_5]),
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
    ttsopenai.WithModel(openai.Models[openai.TTS1HD]),
    ttsopenai.WithVoice("nova"),
    ttsopenai.WithOutputFormat("mp3"),
)
```

Google Cloud, Azure Speech, Deepgram Aura follow the same shape. Vendor
specifics:

- `tts/openai`: `WithInstructions` steers tone and delivery (not on tts-1),
  `WithCustomVoiceID` picks a custom voice, and `StreamAudio` streams the
  audio as it is generated. OpenAI shuts down tts-1, tts-1-hd and
  gpt-4o-mini-tts on 2027-01-06.
- `tts/google`: `WithModel` selects Gemini-TTS (sent as `voice.modelName`;
  pair it with a Gemini voice such as `WithVoiceName("Kore")`), `WithPrompt`
  sets its style instructions, and `WithSpeakingRate`, `WithPitch`,
  `WithSampleRate` and `WithBaseURL` (regional endpoints) tune the rest.
- `tts/deepgram`: `WithModel(deepgram.Models[deepgram.Aura2])` plus
  `WithVoice("thalia-en")` makes `aura-2-thalia-en`; `WithSpeed` and
  `WithMIPOptOut` map to Deepgram's query parameters. Flux TTS (`/v2/speak`)
  is not supported yet and returns an error.

OpenRouter is a thin wrapper over `tts/openai` that fixes the base URL, so one
key reaches OpenAI, Google, Mistral, Microsoft and Deepgram voices:

```go
import ttsopenrouter "github.com/joakimcarlsson/ai/tts/openrouter"

client := ttsopenrouter.NewGeneration(
    ttsopenai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
    ttsopenai.WithModel(openrouter.Models[openrouter.MAIVoice2]),
    ttsopenai.WithVoice("en-US-Harper:MAI-Voice-2"),
    ttsopenai.WithOutputFormat("mp3"),
)
```

There is no model-fallback option: OpenRouter documents its `models` fallback
array for chat completions only, and the audio endpoints ignore request fields
they do not recognise, so sending it would look like it worked while doing
nothing. `WithProviderRouting` is documented for `/audio/speech` and is wired up;
`WithProviderPreferences` sets the whole routing object (`only`, `ignore`,
`zdr`, `data_collection`). The voice is sent only when you set one.

Two OpenRouter specifics: `response_format` defaults to `pcm` rather than
OpenAI's `mp3` and the only documented values are `mp3` and `pcm`; and `speed`
is honored only by upstreams that support it and silently ignored by the rest.
`openrouter.Models` carries 18 known-good defaults. OpenRouter routes
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
format list. Pass a hand-built `tts.AudioModel` to `ttsopenai.WithModel`
instead when something downstream reads those fields.

## Streaming

ElevenLabs, Deepgram and OpenAI (also through OpenRouter) stream chunked audio:

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

The other vendors (`tts/google`, `tts/azure`) buffer the
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

## ElevenLabs dialogue

`tts/elevenlabs.Client` also implements `tts.DialogueProvider`, backed by
ElevenLabs' Text to Dialogue API. `GenerateDialogue` renders a conversation
between several voices as one audio file:

```go
client := ttselevenlabs.NewGeneration(
    ttselevenlabs.WithAPIKey(os.Getenv("ELEVENLABS_API_KEY")),
    ttselevenlabs.WithModel(ttselevenlabs.Models[ttselevenlabs.ElevenV4]),
)

if dp, ok := client.(tts.DialogueProvider); ok {
    resp, err := dp.GenerateDialogue(ctx, []tts.DialogueTurn{
        {Text: "Hi there, how are you today?", VoiceID: voiceA},
        {Text: "Doing great, thanks for asking!", VoiceID: voiceB},
    })
}
```

`StreamDialogue` streams over the Text to Dialogue WebSocket. Register every
voice up front, then send turns as they arrive; `NewTurn` starts a new speaker
turn, and a change of voice does too. Closing the channel flushes the rest:

```go
turns := make(chan tts.DialogueTurn)
chunks, err := dp.StreamDialogue(ctx, []string{voiceA, voiceB}, turns)

go func() {
    defer close(turns)
    turns <- tts.DialogueTurn{Text: "First line.", VoiceID: voiceA, NewTurn: true}
    turns <- tts.DialogueTurn{Text: "The reply.", VoiceID: voiceB, NewTurn: true}
}()

for chunk := range chunks {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    output.Write(chunk.Data)
}
```

Eleven v4 models (`eleven_v4`, `eleven_v4_turbo`) are served only by the
dialogue endpoints. With one of them selected, `GenerateAudio`, `StreamAudio`
and `StreamAudioFromText` send the text as a single turn in the configured
voice, so plain TTS code works unchanged.

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
tts.WithLanguageCode("sv")              // ElevenLabs
tts.WithSeed(42)                        // ElevenLabs
tts.WithTextNormalization("on")         // ElevenLabs: auto, on or off
```
