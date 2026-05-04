package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/tokens"
	"github.com/joakimcarlsson/ai/tokens/truncate"
)

func main() {
	ctx := context.Background()

	counter, err := tokens.NewCounter()
	if err != nil {
		log.Fatal(err)
	}

	firstAnswer := message.NewAssistantMessage()
	firstAnswer.AppendContent("A module is a versioned unit of Go code.")
	secondAnswer := message.NewAssistantMessage()
	secondAnswer.AppendContent("A package is compiled from files in one directory.")

	messages := []message.Message{
		message.NewUserMessage("First question about Go modules."),
		firstAnswer,
		message.NewUserMessage("Second question about packages."),
		secondAnswer,
		message.NewUserMessage("Final question that should remain after truncation."),
	}

	before, err := counter.CountTokens(ctx, tokens.CountOptions{
		Messages: messages,
	})
	if err != nil {
		log.Fatal(err)
	}

	strategy := truncate.Strategy(
		truncate.PreservePairs(),
		truncate.MinMessages(2),
	)
	result, err := strategy.Fit(ctx, tokens.StrategyInput{
		Messages:  messages,
		Counter:   counter,
		MaxTokens: 35,
	})
	if err != nil {
		log.Fatal(err)
	}

	after, err := counter.CountTokens(ctx, tokens.CountOptions{
		Messages: result.Messages,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("messages: %d -> %d\n", len(messages), len(result.Messages))
	fmt.Printf("tokens: %d -> %d\n", before.TotalTokens, after.TotalTokens)
	for _, msg := range result.Messages {
		fmt.Printf("%s: %s\n", msg.Role, msg.Content().String())
	}
}
