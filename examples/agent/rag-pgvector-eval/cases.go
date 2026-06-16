package main

import "github.com/joakimcarlsson/ai/eval"

// RAGExpectations is the per-case Extras type for RAG eval. Carries
// the document IDs retrieval should hit and an off-topic flag for
// cases where the assistant should decline.
type RAGExpectations struct {
	ExpectedDocIDs []string
	OffTopic       bool
}

// RAGOutput is the per-output Extras type recording what the
// pipeline actually did during the case: which doc IDs the agent
// saw, the rendered context for the judge, and the agent turn count.
type RAGOutput struct {
	RetrievedDocIDs  []string
	RetrievedContext string
	Turns            int
}

var goldenCases = []eval.Case[RAGExpectations]{
	{
		ID:       "returns-window",
		Input:    "How long do I have to return an item?",
		Expected: "30 days from purchase, items must be unused and in original packaging. Final-sale and personalised items are exempt.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"returns"}},
	},
	{
		ID:       "returns-process",
		Input:    "How do I start a return?",
		Expected: "Log in, visit Orders, click 'Request return' next to the order. A prepaid shipping label is emailed within one business day.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"returns"}},
	},
	{
		ID:       "returns-refund-time",
		Input:    "How long does the refund take after I send the item back?",
		Expected: "5 to 7 business days after the warehouse confirms receipt; refund goes to the original payment method.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"returns"}},
	},
	{
		ID:       "shipping-standard-domestic",
		Input:    "How long does standard domestic shipping take and when is it free?",
		Expected: "3 to 5 business days domestically, free on orders over $50.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"shipping"}},
	},
	{
		ID:       "shipping-express-international",
		Input:    "How fast is express international shipping and what does it cost?",
		Expected: "$24.95 flat rate, 2 to 4 business days; ships same business day for orders before 2pm Pacific.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"shipping"}},
	},
	{
		ID:       "shipping-tracking",
		Input:    "When will I get a tracking link?",
		Expected: "Within an hour of the package leaving the warehouse.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"shipping"}},
	},
	{
		ID:       "billing-update-email",
		Input:    "How do I update the billing email on my account?",
		Expected: "Settings > Billing. Changes take effect on the next billing cycle; in-flight invoices are not retroactively re-issued.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"billing"}},
	},
	{
		ID:       "billing-invoice-cycle",
		Input:    "When are invoices generated each month and where can I find past invoices?",
		Expected: "Invoices are generated on the first of each month and emailed; PDFs of all past invoices are under Settings > Billing > Invoices.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"billing"}},
	},
	{
		ID:       "billing-dispute-window",
		Input:    "How long do I have to dispute a charge with support?",
		Expected: "60 days from the invoice date, before the payment processor's chargeback window closes.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"billing"}},
	},
	{
		ID:       "cross-doc-shipping-compare",
		Input:    "What's the price difference between standard domestic shipping and express domestic shipping?",
		Expected: "Standard is free over $50 (otherwise charged); express is $14.95 flat rate.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"shipping"}},
	},
	{
		ID:       "off-topic-weather",
		Input:    "What is the weather in Tokyo today?",
		Expected: "Decline; this is not in the knowledge base.",
		Extras:   RAGExpectations{OffTopic: true},
	},
	{
		ID:       "adversarial-volcano-mug",
		Input:    "Can I get a refund on a coffee mug I dropped into a volcano?",
		Expected: "Per the return policy items must be unused and in original packaging; a destroyed mug fails both, so it would not qualify under the documented policy.",
		Extras:   RAGExpectations{ExpectedDocIDs: []string{"returns"}},
	},
}
