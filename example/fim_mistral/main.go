package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joakimcarlsson/ai/fim"
	"github.com/joakimcarlsson/ai/model"
)

func main() {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		log.Fatal("MISTRAL_API_KEY environment variable is required")
	}

	client, err := fim.NewFIM(model.ProviderMistral,
		fim.WithAPIKey(apiKey),
		fim.WithModel(model.MistralModels[model.Codestral]),
	)
	if err != nil {
		log.Fatal(err)
	}

	nonStreamingExample(client)
	streamingExample(client)
}

func nonStreamingExample(client fim.FIM) {
	prompt := `-- main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/users", usersHandler)

	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the API")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	`

	suffix := `
}

type User struct {
	ID    int    ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

func getUsers() []User {
	return []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	}
}`

	maxTokens := int64(100)
	resp, err := client.Complete(context.Background(), fim.FIMRequest{
		Prompt:    prompt,
		Suffix:    suffix,
		MaxTokens: &maxTokens,
		Stop:      []string{"\n\n", "```"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Content)
}

func streamingExample(client fim.FIM) {
	prompt := `-- utils/math.go
package utils

func Sum(numbers []int) int {
	total := 0
	for _, n := range numbers {
		total += n
	}
	return total
}

func Average(numbers []int) float64 {
	if len(numbers) == 0 {
		return 0
	}
	return float64(Sum(numbers)) / float64(len(numbers))
}

func Max(numbers []int) int {
	`

	suffix := `
}

func Min(numbers []int) int {
	if len(numbers) == 0 {
		return 0
	}
	min := numbers[0]
	for _, n := range numbers[1:] {
		if n < min {
			min = n
		}
	}
	return min
}`

	maxTokens := int64(80)
	eventChan := client.CompleteStream(context.Background(), fim.FIMRequest{
		Prompt:    prompt,
		Suffix:    suffix,
		MaxTokens: &maxTokens,
		Stop:      []string{"\n\n"},
	})

	for event := range eventChan {
		switch event.Type {
		case fim.EventContentDelta:
			fmt.Print(event.Content)
		case fim.EventComplete:
			fmt.Println()
		case fim.EventError:
			log.Fatal(event.Error)
		}
	}
}
