# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

This is a Go project using standard Go tooling:

- **Build**: `go build ./...`
- **Run example**: `go run example/structured_output/main.go`
- **Test**: `go test ./...`
- **Format**: `go fmt ./...`
- **Lint**: `go vet ./...`
- **Mod tidy**: `go mod tidy` (to clean up dependencies)

## Code Architecture

This is a multi-provider LLM client library for Go that provides a unified interface for interacting with various AI providers.

### Core Components

- **`providers/`**: Contains the main LLM interface and provider implementations
  - `llm.go`: Main LLM interface with unified API for all providers
  - Individual provider files (anthropic.go, openai.go, gemini.go, etc.)
  - Support for streaming and non-streaming responses
  - Built-in retry logic with exponential backoff

- **`model/`**: Model definitions and provider configurations
  - `models.go`: Core Model struct with pricing, capabilities, and metadata
  - Provider-specific model definitions (anthropic.go, openai.go, etc.)
  - Each provider has its own model constants and configurations

- **`message/`**: Message handling system
  - `base.go`: Core message types (User, Assistant, System, Tool roles)
  - Support for multimodal content (text, images via attachments)
  - `multimodal.go`: Attachment handling for images and other media

- **`tool/`**: Tool calling infrastructure
  - `tool.go`: BaseTool interface for function calling
  - `mcp-tools.go`: MCP (Model Context Protocol) tool integration
  - Structured parameter validation and response handling

- **`schema/`**: Structured output support
  - JSON schema definitions for constrained model outputs
  - Used with providers that support structured generation

- **`types/`**: Event system for streaming responses
  - Event types for content deltas, tool calls, errors, etc.

### Provider Support

The library supports multiple LLM providers through a unified interface:
- Anthropic (Claude models)
- OpenAI (GPT models)
- Google Gemini
- AWS Bedrock
- Azure OpenAI
- Groq
- OpenRouter
- xAI (Grok)
- Google Vertex AI

Each provider implements the same `LLM` interface but may have provider-specific options and capabilities.

### Key Design Patterns

1. **Provider Abstraction**: All providers implement the same `LLM` interface, allowing easy switching between providers
2. **Builder Pattern**: Client creation uses functional options (WithAPIKey, WithModel, etc.)
3. **Streaming Support**: Both regular and structured output support streaming responses via Go channels
4. **Generic Type Safety**: Uses Go generics for type-safe provider implementations
5. **Tool Integration**: Native support for function calling and MCP tools

- Architecture decisions (consider implications)

Use Git Tools:
- Before modifying files (understand history)
- When tests fail (check recent changes)
- Finding related code (git grep)
- Understanding features (follow evolution)
- Checking workflows (CI/CD issues)

The Ten Universal Commandments

1. Thou shalt ALWAYS use MCP tools before coding
2. Thou shalt NEVER assume; always question
3. Thou shalt write code that's clear and obvious
4. Thou shalt be BRUTALLY HONEST in assessments
5. Thou shalt PRESERVE CONTEXT, not delete it
6. Thou shalt make atomic, descriptive commits
7. Thou shalt document the WHY, not just the WHAT
8. Thou shalt test before declaring done
9. Thou shalt handle errors explicitly
10. Thou shalt treat user data as sacred
11. you are NOT ALLOWED TO FUCKING PUT COMMENTS INSIDE FUCKING FUNCTIONS OPR STRCUTS.

Final Reminders
- Codebase > Documentation > Training data (in order of truth)
- Research current docs, don't trust outdated knowledge
- Ask questions early and often
- Use task commands for consistent workflows
- Derive documentation on-demand
- Extended thinking for complex problems
- Visual inputs for UI/UX debugging
- Test locally before pushing
- Think simple: clear, obvious, no bullshit

---
Remember: Write code as if the person maintaining it is a violent psychopath who knows where you live. Make it that clear.
you are NOT ALLOWED TO FUCKING PUT COMMENTS INSIDE FUCKING FUNCTIONS OPR STRCUTS.