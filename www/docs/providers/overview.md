# Supported Providers

Each modality has its own Go module. Vendor implementations are sub-modules.
Pull only the ones you use.

## LLM Providers

Each native LLM vendor is its own sub-module under `llm/`:

| Module | Provider | Streaming | Tools | Structured Output | Attachments |
|---|---|---|---|---|---|
| `llm/anthropic` | Anthropic (Claude) | ✅ | ✅ | ✅ | ✅ |
| `llm/openai` | OpenAI (GPT) | ✅ | ✅ | ✅ | ✅ |
| `llm/gemini` | Google Gemini | ✅ | ✅ | ✅ | ✅ |
| `llm/bedrock` | AWS Bedrock (wraps `llm/anthropic` for Claude on Bedrock) | ✅ | ✅ | ✅ | ✅ |
| `llm/azure` | Azure OpenAI (wraps `llm/openai`) | ✅ | ✅ | ✅ | ✅ |
| `llm/vertexai` | Google Vertex AI (wraps `llm/gemini`) | ✅ | ✅ | ✅ | ✅ |

### OpenAI-compatible vendors

These are thin wrappers over `llm/openai` that hardcode the vendor's base URL.
They expose only the OpenAI-compatible subset; vendor-unique features (xAI Live
Search, Perplexity citations, etc.) are not wired up. Pass any vendor-supported
model id via `openai.WithModel` even without an entry in the `model` package.

| Module | Provider | Default base URL |
|---|---|---|
| `llm/xai` | xAI (Grok) | `https://api.x.ai/v1` |
| `llm/openrouter` | OpenRouter | `https://openrouter.ai/api/v1` |
| `llm/groq` | Groq | `https://api.groq.com/openai/v1` |
| `llm/deepseek` | DeepSeek | `https://api.deepseek.com/v1` |
| `llm/perplexity` | Perplexity Sonar | `https://api.perplexity.ai` |
| `llm/mistral` | Mistral AI | `https://api.mistral.ai/v1` |
| `llm/cerebras` | Cerebras Inference | `https://api.cerebras.ai/v1` |
| `llm/fireworks` | Fireworks AI | `https://api.fireworks.ai/inference/v1` |
| `llm/together` | Together AI | `https://api.together.xyz/v1` |
| `llm/ollama` | Ollama (local) | `http://localhost:11434/v1` |

For any other OpenAI-compatible endpoint, use `llm/openai` directly with
`WithBaseURL(...)`. See [BYOM](../advanced/byom.md).

## Embedding Providers

Each native embedding vendor is its own sub-module under `embeddings/`:

| Module | Provider | Text | Multimodal | Contextualized |
|---|---|---|---|---|
| `embeddings/voyage` | Voyage AI | ✅ | ✅ | ✅ |
| `embeddings/openai` | OpenAI | ✅ | ❌ | ❌ |
| `embeddings/cohere` | Cohere | ✅ | ❌ | ❌ |
| `embeddings/gemini` | Google Gemini | ✅ | ❌ | ❌ |
| `embeddings/mistral` | Mistral | ✅ | ❌ | ❌ |
| `embeddings/bedrock` | AWS Bedrock (Titan + Cohere) | ✅ | ❌ | ❌ |

## Reranker Providers

Each native reranker vendor under `rerankers/`:

| Module | Provider |
|---|---|
| `rerankers/voyage` | Voyage AI |
| `rerankers/cohere` | Cohere |

## Image Generation Providers

Under `image/`:

| Module | Provider | Models | Streaming |
|---|---|---|---|
| `image/openai` | OpenAI | DALL-E 2, DALL-E 3, GPT Image 1 / 1-mini / 1.5 / 2 | ✅ (gpt-image-*) |
| `image/gemini` | Google Gemini | Gemini 2.5 Flash Image, Gemini 3 Pro Image, Imagen 4 / 4 Ultra / 4 Fast | ❌ |
| `image/xai` | xAI | Grok 2 Image, Grok Imagine, Grok Imagine Pro | ❌ |

## TTS (Text-to-Speech) Providers

Under `tts/`:

| Module | Provider | Streaming | Forced Alignment |
|---|---|---|---|
| `tts/openai` | OpenAI | (buffered) | ❌ |
| `tts/elevenlabs` | ElevenLabs | ✅ | ✅ |
| `tts/google` | Google Cloud | (buffered) | ❌ |
| `tts/azure` | Azure Speech | (buffered) | ❌ |
| `tts/deepgram` | Deepgram Aura | ✅ | ❌ |

## STT (Speech-to-Text) Providers

Under `stt/`:

| Module | Provider | Streaming | Translation | Timestamps |
|---|---|---|---|---|
| `stt/openai` | OpenAI Whisper / GPT-4o Transcribe | ❌ | ✅ | ✅ |
| `stt/deepgram` | Deepgram | ✅ | ❌ | ✅ |
| `stt/assemblyai` | AssemblyAI v3 Universal-Streaming | ✅ | ❌ | ✅ |
| `stt/google` | Google Cloud Speech | ❌ | ❌ | ✅ |
| `stt/elevenlabs` | ElevenLabs Scribe v2 Realtime | ✅ | ❌ | ✅ |

## Fill-in-the-Middle (FIM) Providers

Under `fim/`:

| Module | Provider | Streaming |
|---|---|---|
| `fim/mistral` | Mistral Codestral | ✅ |
| `fim/deepseek` | DeepSeek FIM | ✅ |

## Modality interfaces

Each modality also publishes a thin interface module so consumers can write
generic code:

- `llm` — `llm.LLM` interface, request/response types, retry helpers
- `embeddings` — `embeddings.Embedding` interface
- `rerankers` — `rerankers.Reranker` interface
- `image` — `image.ImageGeneration` interface
- `tts` — `tts.Generation` interface (+ optional `tts.ForcedAlignmentProvider`)
- `stt` — `stt.SpeechToText` interface
- `fim` — `fim.FIM` interface

These interface modules carry no vendor SDKs.
