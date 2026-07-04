package anthropic

import (
	"reflect"
	"testing"

	"github.com/joakimcarlsson/ai/schema"
)

// strictSchema returns an OpenAI-strict schema: every field required,
// optionals as nullable unions, including a nested object and
// array-of-object items with their own nullable fields.
func strictSchema() (map[string]any, []string) {
	props := map[string]any{
		"title":    map[string]any{"type": "string"},
		"servings": map[string]any{"type": []string{"integer", "null"}},
		"prep_time": map[string]any{
			"type": []string{"object", "null"},
			"properties": map[string]any{
				"minutes": map[string]any{"type": []string{"number", "null"}},
				"display": map[string]any{"type": "string"},
			},
			"required":             []string{"minutes", "display"},
			"additionalProperties": false,
		},
		"components": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"qty":  map[string]any{"type": []string{"number", "null"}},
				},
				"required":             []string{"name", "qty"},
				"additionalProperties": false,
			},
		},
	}
	return props, []string{"title", "servings", "prep_time", "components"}
}

// hasNullType reports whether any "type" anywhere in the schema tree still lists "null".
func hasNullType(v any) bool {
	switch node := v.(type) {
	case map[string]any:
		if t, ok := node["type"]; ok {
			for _, e := range asAnySlice(t) {
				if s, ok := e.(string); ok && s == "null" {
					return true
				}
			}
		}
		for _, child := range node {
			if hasNullType(child) {
				return true
			}
		}
	case []any:
		for _, e := range node {
			if hasNullType(e) {
				return true
			}
		}
	}
	return false
}

func asAnySlice(t any) []any {
	switch v := t.(type) {
	case []string:
		out := make([]any, len(v))
		for i, s := range v {
			out[i] = s
		}
		return out
	case []any:
		return v
	default:
		return nil
	}
}

// TestRelaxNullableUnions_StripsNullAndOptionalizesEverywhere confirms every
// nullable union in the tree is stripped, the affected fields drop out of
// their required list, and the input is left untouched.
func TestRelaxNullableUnions_StripsNullAndOptionalizesEverywhere(t *testing.T) {
	props, required := strictSchema()

	gotProps, gotReq := relaxNullableUnions(props, required)

	if hasNullType(map[string]any{"properties": gotProps}) {
		t.Errorf(
			"relaxed schema still contains a nullable-union type:\n%#v",
			gotProps,
		)
	}

	if want := []string{
		"title",
		"components",
	}; !reflect.DeepEqual(
		gotReq,
		want,
	) {
		t.Errorf(
			"top-level required = %v, want %v (servings + prep_time are optional now)",
			gotReq,
			want,
		)
	}

	if got := gotProps["title"].(map[string]any)["type"]; got != "string" {
		t.Errorf("title type = %v, want plain \"string\"", got)
	}
	if got := gotProps["servings"].(map[string]any)["type"]; got != "integer" {
		t.Errorf("servings type = %v, want plain \"integer\"", got)
	}
	prep := gotProps["prep_time"].(map[string]any)
	if prep["type"] != "object" {
		t.Errorf("prep_time type = %v, want plain \"object\"", prep["type"])
	}
	if got := prep["required"]; !reflect.DeepEqual(got, []string{"display"}) {
		t.Errorf(
			"prep_time.required = %v, want [display] (minutes is optional now)",
			got,
		)
	}
	items := gotProps["components"].(map[string]any)["items"].(map[string]any)
	if got := items["required"]; !reflect.DeepEqual(got, []string{"name"}) {
		t.Errorf(
			"components.items.required = %v, want [name] (qty is optional now)",
			got,
		)
	}

	if !hasNullType(map[string]any{"properties": props}) {
		t.Error(
			"input schema was mutated: its nullable unions were stripped in place",
		)
	}
	if want := []string{
		"title",
		"servings",
		"prep_time",
		"components",
	}; !reflect.DeepEqual(
		required,
		want,
	) {
		t.Errorf("input required slice was mutated: %v", required)
	}
}

// TestBuildOutputConfigSendsTheRelaxedSchema confirms buildOutputConfig
// actually calls relaxNullableUnions, not just that the helper works in
// isolation.
func TestBuildOutputConfigSendsTheRelaxedSchema(t *testing.T) {
	props, required := strictSchema()

	cfg := (&Client{}).buildOutputConfig(&schema.StructuredOutputInfo{
		Parameters: props,
		Required:   required,
	})

	sent := cfg.Format.Schema
	if sent == nil {
		t.Fatal("buildOutputConfig produced no schema")
	}
	if hasNullType(sent) {
		t.Errorf(
			"the schema actually sent still carries a nullable union -- relaxNullableUnions is not wired in:\n%#v",
			sent,
		)
	}
	gotReq, _ := sent["required"].([]string)
	if want := []string{
		"title",
		"components",
	}; !reflect.DeepEqual(
		gotReq,
		want,
	) {
		t.Errorf("sent required = %v, want %v", gotReq, want)
	}
	if sent["additionalProperties"] != false {
		t.Errorf(
			"additionalProperties = %v, want false",
			sent["additionalProperties"],
		)
	}
}
