# Prompt Templates

A template engine for building dynamic prompts with variable substitution, built-in functions, caching, and validation.

## Basic Usage

```go
import "github.com/joakimcarlsson/ai/prompt"

result, err := prompt.Process("Hello, {{.name}}!", map[string]any{
    "name": "World",
})
// "Hello, World!"
```

## Reusable Templates

```go
tmpl, err := prompt.New("You are {{.role}}. Help with {{.task}}.")
if err != nil {
    log.Fatal(err)
}

result, err := tmpl.Process(map[string]any{
    "role": "a coding assistant",
    "task": "debugging",
})
// "You are a coding assistant. Help with debugging."
```

## Caching

Thread-safe template caching avoids re-parsing the same template repeatedly.

```go
cache := prompt.NewCache()

tmpl, err := prompt.New("You are {{.role}}.",
    prompt.WithCache(cache),
    prompt.WithName("system"),  // cache key
)
```

When using a cache without `WithName`, the template source is hashed automatically as the cache key.

## Validation

Require specific variables to be present in the data map:

```go
_, err := prompt.Process("Hello, {{.name}}!", map[string]any{},
    prompt.WithRequired("name"),
)
// error: missing required variables: name
```

## Strict Mode

Error on any missing variable instead of using zero values:

```go
tmpl, err := prompt.New("{{.name}} is {{.age}} years old.",
    prompt.WithStrictMode(),
)

_, err = tmpl.Process(map[string]any{"name": "Alice"})
// error: template execution fails because .age is missing
```

## Built-in Functions

### String

| Function | Description | Example |
|----------|-------------|---------|
| `upper` | Uppercase | `{{upper .name}}` |
| `lower` | Lowercase | `{{lower .name}}` |
| `title` | Title case | `{{title .name}}` |
| `trim` | Trim whitespace | `{{trim .text}}` |
| `trimPrefix` | Remove prefix | `{{trimPrefix "Mr. " .name}}` |
| `trimSuffix` | Remove suffix | `{{trimSuffix "." .text}}` |
| `replace` | Replace all | `{{replace "old" "new" .text}}` |
| `contains` | Check substring | `{{if contains .text "error"}}...{{end}}` |
| `hasPrefix` | Check prefix | `{{if hasPrefix .name "Dr."}}...{{end}}` |
| `hasSuffix` | Check suffix | `{{if hasSuffix .file ".go"}}...{{end}}` |

### Collections

| Function | Description | Example |
|----------|-------------|---------|
| `join` | Join slice | `{{join ", " .items}}` |
| `split` | Split string | `{{split "," .csv}}` |
| `first` | First element | `{{first .items}}` |
| `last` | Last element | `{{last .items}}` |
| `list` | Create slice | `{{list "a" "b" "c"}}` |

### Comparison

| Function | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `{{if eq .role "admin"}}...{{end}}` |
| `ne` / `neq` | Not equal | `{{if ne .status "done"}}...{{end}}` |
| `lt` | Less than | `{{if lt .count 10}}...{{end}}` |
| `le` | Less or equal | `{{if le .count 10}}...{{end}}` |
| `gt` | Greater than | `{{if gt .count 0}}...{{end}}` |
| `ge` | Greater or equal | `{{if ge .count 1}}...{{end}}` |

### Defaults

| Function | Description | Example |
|----------|-------------|---------|
| `default` | Default value | `{{default "anonymous" .name}}` |
| `coalesce` | First non-empty | `{{coalesce .nickname .name "unknown"}}` |
| `empty` | Check if empty | `{{if empty .list}}...{{end}}` |
| `ternary` | Conditional | `{{ternary .admin "admin" "user"}}` |

### Formatting

| Function | Description | Example |
|----------|-------------|---------|
| `indent` | Indent text | `{{indent 4 .code}}` |
| `nindent` | Newline + indent | `{{nindent 4 .code}}` |
| `quote` | Double quote | `{{quote .name}}` |
| `squote` | Single quote | `{{squote .name}}` |

## Custom Functions

Add your own template functions:

```go
import "text/template"

result, err := prompt.Process("{{shout .name}}", data,
    prompt.WithFuncs(template.FuncMap{
        "shout": func(s string) string {
            return strings.ToUpper(s) + "!!!"
        },
    }),
)
```

## Options

| Option | Description |
|--------|-------------|
| `prompt.WithCache(c)` | Enable template caching |
| `prompt.WithName(name)` | Set template name (used as cache key) |
| `prompt.WithRequired(vars...)` | Require specific variables |
| `prompt.WithStrictMode()` | Error on missing variables |
| `prompt.WithFuncs(funcs)` | Add custom template functions |

## With Agent Instruction Templates

The prompt package powers the agent's [instruction templates](../agent/instruction-templates.md) feature:

```go
myAgent := agent.New(llmClient,
    agent.WithSystemPrompt("You are {{.role}}. The user's name is {{.userName}}."),
    agent.WithState(map[string]any{
        "role":     "a helpful assistant",
        "userName": "Alice",
    }),
)
```
