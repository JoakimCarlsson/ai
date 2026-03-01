# Supported Providers

## LLM Providers

| Provider | Streaming | Tools | Structured Output | Attachments |
|----------|-----------|-------|-------------------|-------------|
| Anthropic (Claude) | ✅ | ✅ | ❌ | ✅ |
| OpenAI (GPT) | ✅ | ✅ | ✅ | ✅ |
| Google Gemini | ✅ | ✅ | ✅ | ✅ |
| AWS Bedrock | ✅ | ✅ | ❌ | ✅ |
| Azure OpenAI | ✅ | ✅ | ✅ | ✅ |
| Google Vertex AI | ✅ | ✅ | ✅ | ✅ |
| Groq | ✅ | ✅ | ✅ | ✅ |
| OpenRouter | ✅ | ✅ | ✅ | ✅ |
| xAI (Grok) | ✅ | ✅ | ✅ | ✅ |

## Embedding & Reranker Providers

| Provider | Text Embeddings | Multimodal Embeddings | Contextualized Embeddings | Rerankers |
|----------|-----------------|----------------------|---------------------------|-----------|
| Voyage AI | ✅ | ✅ | ✅ | ✅ |
| OpenAI | ✅ | ❌ | ❌ | ❌ |

## Image Generation Providers

| Provider | Models | Quality Options | Size Options |
|----------|--------|-----------------|--------------|
| OpenAI | DALL-E 2, DALL-E 3, GPT Image 1 | standard, hd, low, medium, high | 256x256 to 1792x1024 |
| xAI (Grok) | Grok 2 Image | default | default |
| Google Gemini | Gemini 2.5 Flash Image, Imagen 3, Imagen 4, Imagen 4 Ultra, Imagen 4 Fast | default | Aspect ratios: 1:1, 3:4, 4:3, 9:16, 16:9 |

## Audio Generation Providers (Text-to-Speech)

| Provider | Models | Streaming | Voice Selection | Max Characters |
|----------|--------|-----------|-----------------|----------------|
| ElevenLabs | Multilingual v2, Turbo v2.5, Flash v2.5 | ✅ | ✅ | 10,000 - 40,000 |

## Speech-to-Text Providers (Transcription)

| Provider | Models | Streaming | Translation | Timestamps | Diarization |
|----------|--------|-----------|-------------|------------|-------------|
| OpenAI | Whisper-1, GPT-4o Transcribe, GPT-4o Mini Transcribe | ✅ | ✅ | ✅ | ✅ |
