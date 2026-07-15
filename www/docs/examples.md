# Examples

The repository includes runnable examples under [`/examples`](https://github.com/joakimcarlsson/ai/tree/main/examples).
Each example is its own Go module so it shows exactly which packages a downstream
application imports after the multi-module refactor.

Run examples from their own directory:

```bash
cd examples/llm/openai-chat
go run .
```

If you are working inside a local checkout with `go.work` enabled, either add
the example module to the workspace or disable workspace mode for the command:

```bash
GOWORK=off go run .
```

PowerShell:

```powershell
$env:GOWORK = "off"; go run .
```

The example `go.mod` files use local `replace` directives so they build against
the checkout. In your own application, remove those `replace` directives and
install the modules you import with `go get`.

## Package Examples

- `llm/openai-chat` — basic chat completion with `llm/openai`
- `llm/anthropic-stream` — streaming chat with `llm/anthropic`
- `llm/builtin-tools` — server-side built-in tools across `anthropic`, `gemini`, `openai-responses`, `groq-compound`
- `embeddings/voyage` — text embeddings with `embeddings/voyage`
- `image/gemini` — image generation with `image/gemini`
- `image/azure` — image generation against Azure OpenAI with `image/azure`
- `tts/elevenlabs` — text-to-speech with `tts/elevenlabs`
- `stt/deepgram` — speech-to-text with `stt/deepgram`
- `rerankers/cohere` — document reranking with `rerankers/cohere`
- `fim/mistral` — fill-in-the-middle code completion with `fim/mistral`
- `batch/concurrent` — concurrent batch processing around an LLM client
- `agent/basic` — a minimal agent using an LLM client
- `tokens/truncate` — local context truncation without an API key

## Provider Switching

Provider-switch examples show the main point of the modality interfaces: the
business logic stays typed against the shared interface, while only construction
changes per vendor.

- `llm/provider-switch` — `openai`, `anthropic`, `gemini`
- `embeddings/provider-switch` — `openai`, `voyage`, `cohere`
- `image/provider-switch` — `openai`, `gemini`
- `tts/provider-switch` — `openai`, `elevenlabs`
- `stt/provider-switch` — `openai`, `deepgram`
- `rerankers/provider-switch` — `cohere`, `voyage`
- `fim/provider-switch` — `mistral`, `deepseek`
- `batch/provider-switch` — `openai`, `anthropic`, `gemini`
- `agent/provider-switch` — `openai`, `anthropic`, `gemini`

Set `AI_PROVIDER` to choose the implementation:

```bash
AI_PROVIDER=gemini go run .
```

PowerShell:

```powershell
$env:AI_PROVIDER = "gemini"; go run .
```

## Observability And Pricing

- `tracing/otel` — configures OpenTelemetry with `tracing.New`, attaches a stdout
  span exporter, and runs a traced LLM request.
- `model/pricing` — estimates chat, embedding, image, and TTS costs from the
  public model registry fields.

The tracing example prints spans locally by default. For collector-based export,
configure `tracing.New` with OTLP settings such as `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Environment Variables

Set the provider key used by the example you run:

- `OPENAI_API_KEY` for OpenAI LLM, embedding, image, TTS, STT, batch, agent, and tracing examples
- `ANTHROPIC_API_KEY` for Anthropic LLM, batch, and agent examples
- `VOYAGE_API_KEY` for Voyage embedding and reranker examples
- `GEMINI_API_KEY` for Gemini LLM, image, batch, and agent examples
- `GROQ_API_KEY` for the `groq-compound` provider in `llm/builtin-tools`
- `XAI_API_KEY` for the `xai-responses` provider in `llm/builtin-tools`
- `ELEVENLABS_API_KEY` for ElevenLabs TTS examples
- `DEEPGRAM_API_KEY` for Deepgram STT examples
- `COHERE_API_KEY` for Cohere embedding and reranker examples
- `MISTRAL_API_KEY` for Mistral FIM examples
- `DEEPSEEK_API_KEY` for DeepSeek FIM examples

`model/pricing` and `tokens/truncate` run locally and do not require credentials.

## Generated Files

Audio and image examples may write generated artifacts next to the example
program:

- `image/gemini` writes `gemini-image.png`
- `image/azure` writes `azure-image.png`
- `image/provider-switch` writes `<provider>-image.png`
- `tts/elevenlabs` writes `elevenlabs-speech.mp3`
- `tts/provider-switch` writes `<provider>-speech.mp3`
- `stt/deepgram` expects a local audio file path as its only argument
- `stt/provider-switch` expects a local audio file path as its only argument

Do not commit generated audio or image outputs.
