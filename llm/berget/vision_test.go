package berget

import (
	"context"
	"os"
	"strings"
	"testing"

	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
)

func TestLiveVision(t *testing.T) {
	k := os.Getenv("BERGET_API_KEY")
	img := os.Getenv("BERGET_TEST_IMAGE")
	if k == "" || img == "" {
		t.Skip("set BERGET_API_KEY and BERGET_TEST_IMAGE")
	}
	data, err := os.ReadFile(img)
	if err != nil {
		t.Fatal(err)
	}
	c := NewLLM(
		llmopenai.WithAPIKey(k),
		llmopenai.WithModel(model.BergetModels[model.BergetGemma431B]),
		llmopenai.WithMaxTokens(32),
	)
	msg := message.NewUserMessage("What number is shown in this image? Reply with just the number.")
	msg.AddBinary("image/png", data)

	resp, err := c.SendMessages(context.Background(), []message.Message{msg}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("vision response=%q", resp.Content)
	if !strings.Contains(resp.Content, "42") {
		t.Fatalf("model did not read the number: %q", resp.Content)
	}
}
