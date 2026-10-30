// Triage routes support tickets with a decision model.
//
// One Decide call per ticket answers three questions over the same text: which
// department owns it (Choice), how frustrated the customer is (Score) and
// whether it is urgent (Noul). A decision model returns probabilities instead
// of generated text, so low-confidence answers can be sent to a human.
//
// Set AI_PROVIDER to typesafe (default) or openrouter, with
// TYPESAFE_API_KEY or OPENROUTER_API_KEY. Set LIST_MODELS=1 to print the
// models the provider offers first.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joakimcarlsson/ai/decision"
	"github.com/joakimcarlsson/ai/decision/jev"
	"github.com/joakimcarlsson/ai/decision/openrouter"
)

const minConfidence = 0.6

var tickets = []string{
	"My invoice is wrong and I need it fixed today. This is the third time I have asked!",
	"The export button throws a 500 error whenever I click it. No rush, just letting you know.",
	"Thanks for the quick help yesterday, everything works now.",
}

func main() {
	ctx := context.Background()
	decider, provider := newDecider()

	if os.Getenv("LIST_MODELS") != "" {
		listModels(ctx, decider)
	}

	questions := map[string]decision.Question{
		"department": decision.Choice(
			"Which team should handle this ticket?",
			map[string]string{
				"billing":   "Payments, invoices and refunds",
				"technical": "Bugs, errors and outages",
				"other":     "Anything else, including thanks and feedback",
			},
		),
		"frustration": decision.Score(
			"How frustrated is the customer?",
			"calm", "annoyed", "furious",
		),
		"urgent": decision.NoulWithCriteria(
			"Does the customer need action right away?",
			"The customer asks for something to happen now or today.",
			"The customer is fine waiting.",
		),
	}

	for _, ticket := range tickets {
		resp, err := decider.Decide(ctx, ticket, questions)
		if err != nil {
			log.Fatal(err)
		}

		dept, _ := resp.Choice("department")
		mood, _ := resp.Score("frustration")
		urgent, _ := resp.Noul("urgent")

		fmt.Printf("\n%q\n", ticket)
		fmt.Printf("  [%s/%s]\n", provider, resp.Model)
		fmt.Printf(
			"  department:  %s (confidence %.2f)\n",
			dept.Choice, dept.Confidence,
		)
		fmt.Printf(
			"  frustration: %.2f of %d (%s)\n",
			mood.Score, len(mood.Legend)-1, nearest(mood),
		)
		fmt.Printf("  urgent:      %.0f%%\n", urgent.Probability*100)

		if dept.Confidence < minConfidence {
			fmt.Println("  -> low confidence, send to a human")
		} else if urgent.Probability > 0.8 {
			fmt.Printf("  -> escalate to %s now\n", dept.Choice)
		} else {
			fmt.Printf("  -> queue for %s\n", dept.Choice)
		}

		cost := resp.Usage.Cost(decider.Model())
		if resp.Usage.ReportedCost > 0 {
			cost = resp.Usage.ReportedCost
		}
		fmt.Printf(
			"  usage: %d in / %d out tokens, $%.6f\n",
			resp.Usage.InputTokens, resp.Usage.OutputTokens, cost,
		)
	}
}

// nearest returns the legend text of the score level closest to the score.
func nearest(a decision.ScoreAnswer) string {
	i := int(a.Score + 0.5)
	if i < 0 || i >= len(a.Legend) {
		return "?"
	}
	return a.Legend[i]
}

func listModels(ctx context.Context, d decision.Decider) {
	lister, ok := d.(decision.ModelLister)
	if !ok {
		return
	}
	models, err := lister.ListModels(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d models available:\n", len(models))
	for i, m := range models {
		if i == 10 {
			fmt.Printf("  ... and %d more\n", len(models)-i)
			break
		}
		fmt.Printf("  %s\n", m.Name)
	}
}

func newDecider() (decision.Decider, string) {
	switch provider := providerName(); provider {
	case "typesafe":
		return jev.NewDecider(
			jev.WithAPIKey(requiredEnv("TYPESAFE_API_KEY")),
			jev.WithModel(jev.Models[jev.JevLatest]),
		), provider
	case "openrouter":
		return openrouter.NewDecider(
			jev.WithAPIKey(requiredEnv("OPENROUTER_API_KEY")),
			jev.WithModel(openrouter.Models[openrouter.JevLatest]),
		), provider
	default:
		log.Fatalf(
			"unsupported AI_PROVIDER %q (use typesafe or openrouter)",
			provider,
		)
		return nil, ""
	}
}

func providerName() string {
	if p := strings.ToLower(os.Getenv("AI_PROVIDER")); p != "" {
		return p
	}
	return "typesafe"
}

func requiredEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is required", name)
	}
	return v
}
