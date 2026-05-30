// Package perplexity provides an OpenAI-compatible LLM client targeting
// Perplexity's Sonar API.
//
// This wraps [llm/openai] fixed to Perplexity's chat-completions endpoint.
// Beyond the OpenAI-compatible subset, it surfaces Perplexity's vendor features:
// search-control options (domain/recency filters, related questions, web search
// options, disabling search) and the citations and search_results returned with
// every answer, exposed via [llm.Response].ProviderMetadata under
// [MetadataKeyCitations] and [MetadataKeySearchResults].
package perplexity

import (
	"github.com/joakimcarlsson/ai/llm"
	llmopenai "github.com/joakimcarlsson/ai/llm/openai"
)

// DefaultBaseURL is the canonical Perplexity API endpoint.
const DefaultBaseURL = "https://api.perplexity.ai"

// ProviderMetadata keys under which Perplexity search data is surfaced on
// [llm.Response].ProviderMetadata.
const (
	MetadataKeyCitations     = "perplexity.citations"
	MetadataKeySearchResults = "perplexity.search_results"
)

// Option re-exports [llmopenai.Option] for caller convenience.
type Option = llmopenai.Option

// NewLLM constructs a Perplexity LLM client.
//
// [llmopenai.WithBaseURL] is prepended with [DefaultBaseURL]; pass it again in
// opts to override. The client always surfaces the response citations and
// search_results into [llm.Response].ProviderMetadata.
func NewLLM(opts ...Option) llm.LLM {
	base := []Option{
		llmopenai.WithBaseURL(DefaultBaseURL),
		llmopenai.WithResponseMetadataField("citations", MetadataKeyCitations),
		llmopenai.WithResponseMetadataField(
			"search_results", MetadataKeySearchResults),
	}
	return llmopenai.NewLLM(append(base, opts...)...)
}

// WithSearchDomainFilter restricts (or, with a leading "-", excludes) the
// domains Perplexity searches via search_domain_filter.
func WithSearchDomainFilter(domains ...string) Option {
	return llmopenai.WithRequestJSONField("search_domain_filter", domains)
}

// WithSearchRecencyFilter restricts results to a recency window via
// search_recency_filter (e.g. "day", "week", "month", "year").
func WithSearchRecencyFilter(recency string) Option {
	return llmopenai.WithRequestJSONField("search_recency_filter", recency)
}

// WithReturnRelatedQuestions toggles return_related_questions, asking Perplexity
// to include follow-up question suggestions.
func WithReturnRelatedQuestions(enabled bool) Option {
	return llmopenai.WithRequestJSONField("return_related_questions", enabled)
}

// WithWebSearchOptions sets the web_search_options object (e.g. search context
// size, user location) passed through verbatim.
func WithWebSearchOptions(options map[string]any) Option {
	return llmopenai.WithRequestJSONField("web_search_options", options)
}

// WithDisableSearch sets disable_search, turning off web search so the model
// answers from its own knowledge.
func WithDisableSearch() Option {
	return llmopenai.WithRequestJSONField("disable_search", true)
}
