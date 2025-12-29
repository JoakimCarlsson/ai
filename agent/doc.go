// Package agent provides an AI assistant framework with tool execution, memory, and session persistence.
//
// The agent package allows you to create conversational AI assistants that can:
//   - Chat with users using any LLM provider
//   - Execute tools automatically during conversations
//   - Maintain long-term memory about users across sessions
//   - Persist conversation history with pluggable session stores
//
// # Basic Usage
//
//	llmClient, _ := providers.NewLLM(model.ProviderOpenAI,
//	    providers.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
//	    providers.WithModel(model.OpenAIModels[model.GPT4oMini]),
//	)
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a helpful assistant."),
//	)
//
//	response, _ := myAgent.Chat(ctx, "Hello!")
//	fmt.Println(response.Content)
//
// # With Tools
//
// Agents can use tools to perform actions:
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a weather assistant."),
//	    agent.WithTools(&WeatherTool{}),
//	)
//
// # With Session Persistence
//
// Conversation history can be persisted using session stores:
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a helpful assistant."),
//	    agent.WithSession("user-123", session.FileStore("./sessions")),
//	)
//
// # With Memory
//
// Long-term memory allows the agent to remember facts about users:
//
//	myAgent := agent.New(llmClient,
//	    agent.WithSystemPrompt("You are a personal assistant."),
//	    agent.WithMemory("user-123", memoryStore,
//	        memory.AutoExtract(),
//	        memory.AutoDedup(),
//	    ),
//	)
//
// # Streaming Responses
//
// For real-time responses, use ChatStream:
//
//	for event := range myAgent.ChatStream(ctx, "Tell me a story") {
//	    switch event.Type {
//	    case types.EventContentDelta:
//	        fmt.Print(event.Content)
//	    case types.EventError:
//	        log.Fatal(event.Error)
//	    }
//	}
package agent


