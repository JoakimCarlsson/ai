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
| `decision.NoulWithCriteria(instructions, yes, no)` | `decision.NoulAnswer` | as `Noul`, with explicit yes/no definitions |
| `decision.Box(instructions)` | `decision.BoxAnswer` | `Box` (`Top`, `Bottom`, `Left`, `Right` as 0 to 1 fractions of the image), `Confidence` (nil if unreported) |

Box needs an image and a model that supports it (Jev 1.14).

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
`WithTimeout`, `WithLongContext`. Input is billed at $0.042 per 1M tokens;
output is free.

### Images and Box

Wrap image bytes with `decision.NewImage`, write its `Ref` wherever the image
belongs in the state, instructions or criteria, and pass it with
`decision.WithImages`. The images header is set automatically, and images are
sent as base64. Images are PNG or JPEG, 280 to 1568 px per side, and an early
feature of the provider.

```go
png, _ := os.ReadFile("photo.png")
img := decision.NewImage("png", png)

resp, err := decider.Decide(ctx, "photo: "+img.Ref,
    map[string]decision.Question{
        "rect": decision.Box("Locate the red rectangle."),
        "has":  decision.Noul("Is there a red rectangle in the photo?"),
    },
    decision.WithImages(img),
)
box, _ := resp.Box("rect")
fmt.Println(box.Box.Top, box.Box.Left)
```

Box always returns coordinates, even when the target is absent; ask a `Noul`
alongside to check presence. Text over about 32,000 bytes and every image
request use the provider's long-context mode, which is slower and less
accurate; `WithLongContext` opts every request in.

### Listing models

The decider also implements `decision.ModelLister`:

```go
if l, ok := decider.(decision.ModelLister); ok {
    models, err := l.ListModels(ctx)
}
```

## OpenRouter

OpenRouter serves Jev over the same System One wire format, billed to an
OpenRouter key. `decision/openrouter` wraps `decision/jev` with that base URL.

```go
import "github.com/joakimcarlsson/ai/decision/openrouter"

decider := openrouter.NewDecider() // default key: OPENROUTER_API_KEY
```

It takes the same options as `jev.NewDecider`, and Box, images and
`ListModels` work the same way. Models are
`openrouter.Models[openrouter.JevLatest]` (default), `openrouter.Jev114` and
`openrouter.Jev113`; `openrouter.WithModelID("...")` selects any other
OpenRouter id. Pricing is the same as TypeSafe's, and OpenRouter's billed cost
is returned in `Usage.ReportedCost`.

`ListModels` returns every decision model OpenRouter serves, which includes
other vendors' System One-style models besides Jev. Only Jev has been tested
with this package.

Failed requests are returned as `*decision.APIError`, which satisfies
`llm.RetryableError`. `llm.DefaultRetryConfig` retries 429 and 5xx, but not
TypeSafe's 529 (overloaded); add it to `RetryStatusCodes` to retry those too.
