# Toolsets

Toolsets group multiple tools under a name with optional dynamic filtering. Unlike static tool lists, toolsets are resolved **per-call** — the predicate runs on every `Chat()` turn, so you can enable or disable tools based on runtime context.

## Creating a Toolset

A basic toolset is a named collection of tools:

```go
recon := tool.NewToolset("recon",
    &NmapTool{},
    &DnsLookupTool{},
    &WhoisTool{},
)

a := agent.New(llmClient,
    agent.WithToolsets(recon),
)
```

You can mix toolsets with individual tools:

```go
a := agent.New(llmClient,
    agent.WithTools(&AlwaysAvailableTool{}),
    agent.WithToolsets(recon, exploitation),
)
```

## Filtered Toolsets

`NewFilterToolset` wraps a toolset with a predicate that controls which tools are available. The predicate receives the `context.Context` and each tool, and returns whether that tool should be included.

```go
type phaseKey struct{}

allTools := tool.NewToolset("pentest",
    &NmapTool{},
    &SqlInjectionTool{},
    &BruteForcePasswordTool{},
)

filtered := tool.NewFilterToolset("phase-aware", allTools,
    func(ctx context.Context, t tool.BaseTool) bool {
        phase, _ := ctx.Value(phaseKey{}).(string)
        switch t.Info().Name {
        case "sql_injection", "brute_force_password":
            return phase == "exploitation"
        default:
            return true
        }
    },
)

a := agent.New(llmClient,
    agent.WithToolsets(filtered),
)

// During recon phase, only NmapTool is available
ctx := context.WithValue(ctx, phaseKey{}, "recon")
resp, _ := a.Chat(ctx, "Start scanning the target")

// During exploitation phase, all tools are available
ctx = context.WithValue(ctx, phaseKey{}, "exploitation")
resp, _ = a.Chat(ctx, "Try exploiting the SQL injection")
```

### Filtering by Configuration

Predicates can also read from engagement configuration or any other source:

```go
type EngagementConfig struct {
    AllowBruteForce bool
    AllowExploits   bool
}

configKey := struct{}{}

filtered := tool.NewFilterToolset("engagement", allTools,
    func(ctx context.Context, t tool.BaseTool) bool {
        cfg, _ := ctx.Value(configKey).(*EngagementConfig)
        if cfg == nil {
            return false
        }
        switch t.Info().Name {
        case "brute_force":
            return cfg.AllowBruteForce
        case "sql_injection", "xss_scanner":
            return cfg.AllowExploits
        default:
            return true
        }
    },
)
```

## Composing Toolsets

Toolsets compose — use `NewCompositeToolset` to merge multiple toolsets into one:

```go
recon := tool.NewToolset("recon", &NmapTool{}, &DnsLookupTool{})
exploit := tool.NewToolset("exploit", &SqlInjectionTool{})
reporting := tool.NewToolset("reporting", &ReportTool{})

all := tool.NewCompositeToolset("full-suite", recon, exploit, reporting)
```

Composite toolsets work with filtered toolsets too — you can filter individual groups and then compose them:

```go
filteredExploit := tool.NewFilterToolset("filtered-exploit", exploit, exploitPredicate)
combined := tool.NewCompositeToolset("suite", recon, filteredExploit, reporting)
```

## MCP Toolsets

Wrap MCP server tools as a toolset:

```go
mcpTools := tool.MCPToolset("external", map[string]tool.MCPServer{
    "filesystem": {
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
        Type:    tool.MCPStdio,
    },
})

a := agent.New(llmClient,
    agent.WithToolsets(mcpTools),
)
```

## Toolsets and Hooks

Tool confirmation works at the toolset level through the existing [hook system](hooks.md). Since toolsets resolve to `[]tool.BaseTool`, hooks apply to individual tools regardless of how they were grouped:

```go
a := agent.New(llmClient,
    agent.WithToolsets(exploitToolset),
    agent.WithHooks(agent.Hooks{
        PreToolUse: func(ctx context.Context, tc agent.ToolUseContext) (agent.PreToolUseResult, error) {
            if tc.ToolName == "sql_injection" {
                // Require confirmation for dangerous tools
                return agent.PreToolUseResult{
                    Action:     agent.HookDeny,
                    DenyReason: "SQL injection requires manual approval",
                }, nil
            }
            return agent.PreToolUseResult{Action: agent.HookAllow}, nil
        },
    }),
)
```

## Custom Toolset Implementations

The `Toolset` interface is simple — implement it for custom resolution logic:

```go
type Toolset interface {
    Name() string
    Tools(ctx context.Context) []tool.BaseTool
}
```

For example, a toolset that loads tools from a database:

```go
type DBToolset struct {
    db *sql.DB
}

func (d *DBToolset) Name() string { return "db-tools" }

func (d *DBToolset) Tools(ctx context.Context) []tool.BaseTool {
    // Query available tools from database based on user permissions
    rows, _ := d.db.QueryContext(ctx, "SELECT name, config FROM tools WHERE enabled = true")
    // ... build and return tools
}
```
