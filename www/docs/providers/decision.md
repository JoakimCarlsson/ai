# Decision Models

The `decision` modality. Vendors under `decision/`.

A decision model does not generate text. It reads a state and returns typed
answers — a choice, a score or a probability — in a single pass, which suits
routing, tool-call gating, classification and guardrails inside agent loops.
Several questions are answered over one state in a single call.

## Questions

| Constructor | Answer | Fields |
|---|---|---|
| `decision.Choice(instructions, criteria)` | `decision.ChoiceAnswer` | `Choice`, `Probabilities`, `Confidence` |
| `decision.Score(instructions, levels...)` | `decision.ScoreAnswer` | `Score` (may fall between levels), `Probabilities`, `Legend`, `Confidence` |
| `decision.Noul(instructions)` | `decision.NoulAnswer` | `Probability` of yes, 0 to 1 |

## Jev (TypeSafe AI)

```go
import (
    "github.com/joakimcarlsson/ai/decision"
    "github.com/joakimcarlsson/ai/decision/jev"
)

decider := jev.NewDecider(
    jev.WithAPIKey(os.Getenv("TYPESAFE_API_KEY")), // default: TYPESAFE_API_KEY
    jev.WithModel(jev.Models[jev.JevLatest]),
)

resp, err := decider.Decide(ctx,
    "My invoice is wrong and I need it fixed today.",
    map[string]decision.Question{
        "department": decision.Choice("Route the ticket.", map[string]string{
            "billing":   "Payments and invoices",
            "technical": "Bugs and outages",
        }),
        "frustration": decision.Score(
            "How frustrated is the customer?", "calm", "annoyed", "furious",
        ),
        "urgent": decision.Noul("Is the customer asking for urgent help?"),
    },
)

dept, _ := resp.Choice("department")
fmt.Println(dept.Choice, dept.Confidence)
urgent, _ := resp.Noul("urgent")
fmt.Println(urgent.Probability)
fmt.Println(resp.Usage.Cost(decider.Model()))
```

Options: `WithAPIKey`, `WithModel`, `WithBaseURL`, `WithHTTPClient`,
`WithTimeout`. Input is billed at $0.042 per 1M tokens; output is free.

## OpenRouter

OpenRouter serves Jev over the same System One wire format, billed to an
OpenRouter key. `decision/openrouter` wraps `decision/jev` with that base URL.

```go
import "github.com/joakimcarlsson/ai/decision/openrouter"

decider := openrouter.NewDecider() // default key: OPENROUTER_API_KEY
```

It takes the same options as `jev.NewDecider`. Models are
`openrouter.Models[openrouter.JevLatest]` (default) and
`openrouter.Jev113`; `openrouter.WithModelID("...")` selects any other
OpenRouter id. Pricing is the same as TypeSafe's.

Rate-limit (429) and overload (529) responses are returned as
`*decision.APIError`, which satisfies `llm.RetryableError`, so
`llm.ShouldRetry` treats 429 as retryable.
