package tool

import "context"

// Toolset groups multiple tools under a name with optional dynamic filtering.
// Toolsets compose — a toolset can contain individual tools and other toolsets.
type Toolset interface {
	// Name returns the toolset's identifier.
	Name() string
	// Tools returns the tools available in this toolset for the given context.
	// Implementations may filter tools dynamically based on context values.
	Tools(ctx context.Context) []BaseTool
}

// ToolsetPredicate decides whether a tool should be available given the current context.
type ToolsetPredicate func(ctx context.Context, t BaseTool) bool

// NewToolset creates a static toolset that always returns the same tools.
func NewToolset(name string, tools ...BaseTool) Toolset {
	return &staticToolset{name: name, tools: tools}
}

// NewFilterToolset wraps a toolset with a predicate that controls per-tool availability.
// On each call to Tools(), the predicate is evaluated for every tool in the inner set.
func NewFilterToolset(
	name string,
	inner Toolset,
	predicate ToolsetPredicate,
) Toolset {
	return &filterToolset{name: name, inner: inner, predicate: predicate}
}

// NewCompositeToolset creates a toolset that merges tools from multiple child toolsets.
func NewCompositeToolset(name string, children ...Toolset) Toolset {
	return &compositeToolset{name: name, children: children}
}

// MCPToolset creates a toolset from MCP server tools.
// The returned toolset resolves tools by connecting to the configured MCP servers.
func MCPToolset(name string, servers map[string]MCPServer) Toolset {
	return &mcpToolset{name: name, servers: servers}
}

type staticToolset struct {
	name  string
	tools []BaseTool
}

func (s *staticToolset) Name() string                       { return s.name }
func (s *staticToolset) Tools(_ context.Context) []BaseTool { return s.tools }

type filterToolset struct {
	name      string
	inner     Toolset
	predicate ToolsetPredicate
}

func (f *filterToolset) Name() string { return f.name }

func (f *filterToolset) Tools(ctx context.Context) []BaseTool {
	all := f.inner.Tools(ctx)
	filtered := make([]BaseTool, 0, len(all))
	for _, t := range all {
		if f.predicate(ctx, t) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

type compositeToolset struct {
	name     string
	children []Toolset
}

func (c *compositeToolset) Name() string { return c.name }

func (c *compositeToolset) Tools(ctx context.Context) []BaseTool {
	var tools []BaseTool
	for _, child := range c.children {
		tools = append(tools, child.Tools(ctx)...)
	}
	return tools
}

// WithConfirmation wraps a toolset so that every tool it returns has RequireConfirmation set to true.
// The wrapped tools delegate Run() to the originals unchanged.
func WithConfirmation(inner Toolset) Toolset {
	return &confirmationToolset{inner: inner}
}

type confirmationToolset struct {
	inner Toolset
}

func (c *confirmationToolset) Name() string { return c.inner.Name() }

func (c *confirmationToolset) Tools(ctx context.Context) []BaseTool {
	tools := c.inner.Tools(ctx)
	wrapped := make([]BaseTool, len(tools))
	for i, t := range tools {
		wrapped[i] = &confirmationToolWrapper{inner: t}
	}
	return wrapped
}

type confirmationToolWrapper struct {
	inner BaseTool
}

func (w *confirmationToolWrapper) Info() Info {
	info := w.inner.Info()
	info.RequireConfirmation = true
	return info
}

func (w *confirmationToolWrapper) Run(
	ctx context.Context,
	params Call,
) (Response, error) {
	return w.inner.Run(ctx, params)
}

type mcpToolset struct {
	name    string
	servers map[string]MCPServer
}

func (m *mcpToolset) Name() string { return m.name }

func (m *mcpToolset) Tools(ctx context.Context) []BaseTool {
	tools, err := GetMcpTools(ctx, m.servers)
	if err != nil {
		return nil
	}
	return tools
}
