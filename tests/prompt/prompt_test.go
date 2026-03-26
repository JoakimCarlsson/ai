package prompt

import (
	"strings"
	"testing"
	"text/template"

	"github.com/joakimcarlsson/ai/prompt"
)

func TestProcess_BasicTemplate(t *testing.T) {
	result, err := prompt.Process(
		"Hello, {{.name}}!",
		map[string]any{"name": "World"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}
}

func TestProcess_NoVariables(t *testing.T) {
	result, err := prompt.Process("Static text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Static text" {
		t.Errorf("expected 'Static text', got %q", result)
	}
}

func TestProcess_InvalidTemplate(t *testing.T) {
	_, err := prompt.Process("{{.unclosed", nil)
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

func TestNew_WithRequired_Missing(t *testing.T) {
	tmpl, err := prompt.New(
		"Hello {{.name}}",
		prompt.WithRequired("name"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tmpl.Process(map[string]any{})
	if err == nil {
		t.Error("expected validation error for missing required")
	}

	var ve *prompt.ValidationError
	ok := false
	if e, matches := err.(*prompt.ValidationError); matches {
		ve = e
		ok = true
	}
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Missing) != 1 || ve.Missing[0] != "name" {
		t.Errorf("expected [name] missing, got %v", ve.Missing)
	}
}

func TestNew_WithRequired_Present(t *testing.T) {
	tmpl, err := prompt.New(
		"Hello {{.name}}",
		prompt.WithRequired("name"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := tmpl.Process(
		map[string]any{"name": "Alice"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello Alice" {
		t.Errorf("expected 'Hello Alice', got %q", result)
	}
}

func TestNew_WithStrictMode_MissingKey(t *testing.T) {
	tmpl, err := prompt.New(
		"Hello {{.name}}",
		prompt.WithStrictMode(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = tmpl.Process(map[string]any{})
	if err == nil {
		t.Error("expected error in strict mode for missing key")
	}
}

func TestNew_WithCache(t *testing.T) {
	cache := prompt.NewCache()
	source := "Hello {{.name}}"

	t1, err := prompt.New(
		source,
		prompt.WithCache(cache),
		prompt.WithName("greeting"),
	)
	if err != nil {
		t.Fatalf("first create error: %v", err)
	}

	t2, err := prompt.New(
		source,
		prompt.WithCache(cache),
		prompt.WithName("greeting"),
	)
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}

	r1, _ := t1.Process(map[string]any{"name": "A"})
	r2, _ := t2.Process(map[string]any{"name": "B"})

	if r1 != "Hello A" {
		t.Errorf("expected 'Hello A', got %q", r1)
	}
	if r2 != "Hello B" {
		t.Errorf("expected 'Hello B', got %q", r2)
	}
}

func TestNew_WithCache_NoName_UsesHash(t *testing.T) {
	cache := prompt.NewCache()
	source := "Template {{.x}}"

	_, err := prompt.New(source, prompt.WithCache(cache))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = prompt.New(source, prompt.WithCache(cache))
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}
}

func TestCache_Clear(t *testing.T) {
	cache := prompt.NewCache()

	_, _ = prompt.New(
		"template A",
		prompt.WithCache(cache),
		prompt.WithName("a"),
	)

	cache.Clear()

	if cache.Get("a") != nil {
		t.Error("expected nil after cache clear")
	}
}

func TestNew_WithCustomFuncs(t *testing.T) {
	result, err := prompt.Process(
		`{{exclaim .text}}`,
		map[string]any{"text": "wow"},
		prompt.WithFuncs(template.FuncMap{
			"exclaim": func(s string) string {
				return s + "!!!"
			},
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "wow!!!" {
		t.Errorf("expected 'wow!!!', got %q", result)
	}
}

func TestBuiltinFunc_Upper(t *testing.T) {
	result, _ := prompt.Process(
		`{{upper .text}}`,
		map[string]any{"text": "hello"},
	)
	if result != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", result)
	}
}

func TestBuiltinFunc_Lower(t *testing.T) {
	result, _ := prompt.Process(
		`{{lower .text}}`,
		map[string]any{"text": "HELLO"},
	)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestBuiltinFunc_Trim(t *testing.T) {
	result, _ := prompt.Process(
		`{{trim .text}}`,
		map[string]any{"text": "  spaced  "},
	)
	if result != "spaced" {
		t.Errorf("expected 'spaced', got %q", result)
	}
}

func TestBuiltinFunc_Join(t *testing.T) {
	result, _ := prompt.Process(
		`{{join ", " .items}}`,
		map[string]any{"items": []string{"a", "b", "c"}},
	)
	if result != "a, b, c" {
		t.Errorf("expected 'a, b, c', got %q", result)
	}
}

func TestBuiltinFunc_First(t *testing.T) {
	result, _ := prompt.Process(
		`{{first .items}}`,
		map[string]any{"items": []string{"x", "y", "z"}},
	)
	if result != "x" {
		t.Errorf("expected 'x', got %q", result)
	}
}

func TestBuiltinFunc_Last(t *testing.T) {
	result, _ := prompt.Process(
		`{{last .items}}`,
		map[string]any{"items": []string{"x", "y", "z"}},
	)
	if result != "z" {
		t.Errorf("expected 'z', got %q", result)
	}
}

func TestBuiltinFunc_First_EmptySlice(t *testing.T) {
	result, _ := prompt.Process(
		`{{first .items}}`,
		map[string]any{"items": []string{}},
	)
	if result != "<no value>" {
		t.Errorf("expected '<no value>', got %q", result)
	}
}

func TestBuiltinFunc_Default(t *testing.T) {
	result, _ := prompt.Process(
		`{{default "fallback" .val}}`,
		map[string]any{"val": ""},
	)
	if result != "fallback" {
		t.Errorf("expected 'fallback', got %q", result)
	}

	result2, _ := prompt.Process(
		`{{default "fallback" .val}}`,
		map[string]any{"val": "present"},
	)
	if result2 != "present" {
		t.Errorf("expected 'present', got %q", result2)
	}
}

func TestBuiltinFunc_Coalesce(t *testing.T) {
	result, _ := prompt.Process(
		`{{coalesce .a .b .c}}`,
		map[string]any{"a": "", "b": nil, "c": "found"},
	)
	if result != "found" {
		t.Errorf("expected 'found', got %q", result)
	}
}

func TestBuiltinFunc_Ternary(t *testing.T) {
	result, _ := prompt.Process(
		`{{ternary .cond "yes" "no"}}`,
		map[string]any{"cond": true},
	)
	if result != "yes" {
		t.Errorf("expected 'yes', got %q", result)
	}

	result2, _ := prompt.Process(
		`{{ternary .cond "yes" "no"}}`,
		map[string]any{"cond": false},
	)
	if result2 != "no" {
		t.Errorf("expected 'no', got %q", result2)
	}
}

func TestBuiltinFunc_Indent(t *testing.T) {
	result, _ := prompt.Process(
		`{{indent 4 .text}}`,
		map[string]any{"text": "line1\nline2"},
	)
	if result != "    line1\n    line2" {
		t.Errorf("expected indented text, got %q", result)
	}
}

func TestBuiltinFunc_Nindent(t *testing.T) {
	result, _ := prompt.Process(
		`before{{nindent 2 .text}}`,
		map[string]any{"text": "line1"},
	)
	if result != "before\n  line1" {
		t.Errorf("expected newline+indent, got %q", result)
	}
}

func TestBuiltinFunc_Quote(t *testing.T) {
	result, _ := prompt.Process(
		`{{quote .text}}`,
		map[string]any{"text": "hello"},
	)
	if result != `"hello"` {
		t.Errorf("expected '\"hello\"', got %q", result)
	}
}

func TestBuiltinFunc_Squote(t *testing.T) {
	result, _ := prompt.Process(
		`{{squote .text}}`,
		map[string]any{"text": "hello"},
	)
	if result != "'hello'" {
		t.Errorf("expected \"'hello'\", got %q", result)
	}
}

func TestBuiltinFunc_Comparisons(t *testing.T) {
	tests := []struct {
		tmpl     string
		expected string
	}{
		{`{{if eq .a .b}}yes{{else}}no{{end}}`, "yes"},
		{`{{if ne .a .c}}yes{{else}}no{{end}}`, "yes"},
		{`{{if lt .d .e}}yes{{else}}no{{end}}`, "yes"},
		{`{{if le .d .d}}yes{{else}}no{{end}}`, "yes"},
		{`{{if gt .e .d}}yes{{else}}no{{end}}`, "yes"},
		{`{{if ge .e .e}}yes{{else}}no{{end}}`, "yes"},
	}

	data := map[string]any{
		"a": 1, "b": 1, "c": 2, "d": 3, "e": 5,
	}

	for _, tt := range tests {
		result, err := prompt.Process(tt.tmpl, data)
		if err != nil {
			t.Fatalf("template %q error: %v", tt.tmpl, err)
		}
		if result != tt.expected {
			t.Errorf(
				"template %q: expected %q, got %q",
				tt.tmpl,
				tt.expected,
				result,
			)
		}
	}
}

func TestBuiltinFunc_Replace(t *testing.T) {
	result, _ := prompt.Process(
		`{{replace "world" "Go" .text}}`,
		map[string]any{"text": "hello world"},
	)
	if result != "hello Go" {
		t.Errorf("expected 'hello Go', got %q", result)
	}
}

func TestBuiltinFunc_Contains(t *testing.T) {
	result, _ := prompt.Process(
		`{{if contains .text "ell"}}yes{{else}}no{{end}}`,
		map[string]any{"text": "hello"},
	)
	if result != "yes" {
		t.Errorf("expected 'yes', got %q", result)
	}
}

func TestBuiltinFunc_HasPrefix(t *testing.T) {
	result, _ := prompt.Process(
		`{{if hasPrefix .text "hel"}}yes{{else}}no{{end}}`,
		map[string]any{"text": "hello"},
	)
	if result != "yes" {
		t.Errorf("expected 'yes', got %q", result)
	}
}

func TestBuiltinFunc_List(t *testing.T) {
	result, _ := prompt.Process(
		`{{$l := list "a" "b" "c"}}{{join "-" $l}}`,
		nil,
	)
	if result != "a-b-c" {
		t.Errorf("expected 'a-b-c', got %q", result)
	}
}

func TestBuiltinFunc_Empty(t *testing.T) {
	tests := []struct {
		val      any
		expected string
	}{
		{nil, "yes"},
		{"", "yes"},
		{0, "yes"},
		{false, "yes"},
		{[]string{}, "yes"},
		{"text", "no"},
		{1, "no"},
		{true, "no"},
	}

	for _, tt := range tests {
		result, err := prompt.Process(
			`{{if empty .val}}yes{{else}}no{{end}}`,
			map[string]any{"val": tt.val},
		)
		if err != nil {
			t.Fatalf("error for val=%v: %v", tt.val, err)
		}
		if result != tt.expected {
			t.Errorf(
				"empty(%v): expected %q, got %q",
				tt.val,
				tt.expected,
				result,
			)
		}
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := &prompt.ValidationError{
		Missing: []string{"name", "age"},
	}
	msg := ve.Error()
	if !strings.Contains(msg, "name") ||
		!strings.Contains(msg, "age") {
		t.Errorf("expected missing fields in error, got %q", msg)
	}
}
