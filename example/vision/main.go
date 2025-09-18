package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
)

const (
	testImageURL = "https://static0.srcdn.com/wordpress/wp-content/uploads/2020/04/Rickroll-Wide.jpg"
)

func main() {
	ctx := context.Background()
	anthropicExample(ctx)
	openaiExample(ctx)
	multiModalExample(ctx)
}

func anthropicExample(ctx context.Context) {
	client, err := llm.NewLLM(
		model.ProviderAnthropic,
		llm.WithAPIKey(""),
		llm.WithModel(model.AnthropicModels[model.Claude35Sonnet]),
		llm.WithMaxTokens(1000),
	)
	if err != nil {
		log.Fatal(err)
	}

	urlMessage := message.NewUserMessage("Describe this image in detail.")
	urlMessage.AddImageURL(testImageURL, "")

	response, err := client.SendMessages(ctx, []message.Message{urlMessage}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	base64Message := message.NewUserMessage("What do you see in this image?")
	imageData, mimeType, err := downloadImage(testImageURL)
	if err != nil {
		log.Fatal(err)
	}
	base64Message.AddBinary(mimeType, imageData)

	response, err = client.SendMessages(ctx, []message.Message{base64Message}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}

func openaiExample(ctx context.Context) {
	client, err := llm.NewLLM(
		model.ProviderOpenAI,
		llm.WithAPIKey(""),
		llm.WithModel(model.OpenAIModels[model.GPT4o]),
		llm.WithMaxTokens(1000),
	)
	if err != nil {
		log.Fatal(err)
	}

	urlMessage := message.NewUserMessage("What is in this image?")
	urlMessage.AddImageURL(testImageURL, "high")

	response, err := client.SendMessages(ctx, []message.Message{urlMessage}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)

	base64Message := message.NewUserMessage("Analyze this image and tell me what you observe.")
	imageData, mimeType, err := downloadImage(testImageURL)
	if err != nil {
		log.Fatal(err)
	}
	base64Message.AddBinary(mimeType, imageData)

	response, err = client.SendMessages(ctx, []message.Message{base64Message}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}

func downloadImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	return data, mimeType, nil
}

func multiModalExample(ctx context.Context) {
	client, err := llm.NewLLM(
		model.ProviderAnthropic,
		llm.WithAPIKey(""),
		llm.WithModel(model.AnthropicModels[model.Claude35Sonnet]),
		llm.WithMaxTokens(1500),
	)
	if err != nil {
		log.Fatal(err)
	}

	multiMessage := message.NewUserMessage("Compare these two images and tell me the differences:")
	multiMessage.AddImageURL(testImageURL, "")

	imageData, mimeType, err := downloadImage("https://upload.wikimedia.org/wikipedia/commons/a/a7/Camponotus_flavomarginatus_ant.jpg")
	if err != nil {
		log.Fatal(err)
	}
	multiMessage.AddBinary(mimeType, imageData)

	response, err := client.SendMessages(ctx, []message.Message{multiMessage}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Content)
}
