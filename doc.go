// Package ai provides a unified interface for interacting with AI models from multiple providers.
//
// This library abstracts away provider-specific differences while maintaining access to
// advanced features like streaming responses, tool calling, structured output, and multimodal capabilities.
//
// Supported providers include OpenAI, Anthropic, Google, AWS Bedrock, Azure, Voyage AI, and others.
//
// Basic usage:
//
//	import (
//		"github.com/joakimcarlsson/ai/message"
//		"github.com/joakimcarlsson/ai/model"
//		llm "github.com/joakimcarlsson/ai/providers"
//	)
//
//	client, err := llm.NewLLM(
//		model.ProviderOpenAI,
//		llm.WithAPIKey("your-api-key"),
//		llm.WithModel(model.OpenAIModels[model.GPT4o]),
//	)
//
//	messages := []message.Message{
//		message.NewUserMessage("Hello, how are you?"),
//	}
//
//	response, err := client.SendMessages(ctx, messages, nil)
//
// The library supports LLMs, embedding models, and document rerankers through
// dedicated subpackages. See individual package documentation for detailed usage examples.
package ai
