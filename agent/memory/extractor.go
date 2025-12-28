package memory

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/joakimcarlsson/ai/message"
	llm "github.com/joakimcarlsson/ai/providers"
)

const factExtractionPrompt = `You are a Personal Information Organizer, specialized in accurately storing facts, user memories, and preferences. Your primary role is to extract relevant pieces of information from conversations and organize them into distinct, manageable facts.

Types of Information to Remember:
1. Personal Preferences: likes, dislikes, preferences in food, products, activities, entertainment
2. Important Personal Details: names, relationships, important dates
3. Plans and Intentions: upcoming events, trips, goals
4. Activity and Service Preferences: dining, travel, hobbies
5. Health and Wellness: dietary restrictions, fitness routines, allergies
6. Professional Details: job titles, work habits, career goals
7. Miscellaneous: favorite books, movies, brands, other details

IMPORTANT: Only extract facts from USER messages. Do not include information from assistant messages.

Return a JSON object with a "facts" array containing strings of extracted facts.
If no relevant facts are found, return {"facts": []}.

Examples:
Input: "Hi, my name is John. I am a software engineer."
Output: {"facts": ["Name is John", "Is a software engineer"]}

Input: "I'm allergic to peanuts and I prefer vegetarian food."
Output: {"facts": ["Allergic to peanuts", "Prefers vegetarian food"]}

Input: "What's the weather like?"
Output: {"facts": []}
`

type factExtractionResult struct {
	Facts []string `json:"facts"`
}

// ExtractFacts extracts facts from a conversation using an LLM.
// It only extracts facts from user messages, ignoring system and assistant messages.
func ExtractFacts(ctx context.Context, llmClient llm.LLM, messages []message.Message) ([]string, error) {
	var conversationBuilder strings.Builder
	for _, msg := range messages {
		if msg.Role == message.System {
			continue
		}
		role := string(msg.Role)
		content := msg.Content().Text
		if content != "" {
			conversationBuilder.WriteString(role + ": " + content + "\n")
		}
	}

	conversation := conversationBuilder.String()
	if conversation == "" {
		return nil, nil
	}

	extractionMessages := []message.Message{
		message.NewSystemMessage(factExtractionPrompt),
		message.NewUserMessage("Extract facts from this conversation:\n\n" + conversation),
	}

	resp, err := llmClient.SendMessages(ctx, extractionMessages, nil)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result factExtractionResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, err
	}

	return result.Facts, nil
}

