package anthropic

import (
	"reflect"
	"testing"

	"github.com/joakimcarlsson/ai/schema"
)

// TestFactorSharedObjectDefs_HoistsRepeatedTypes confirms a repeated object
// type is hoisted into a single shared $def and referenced by $ref at every
// site, keeping each site's own description.
func TestFactorSharedObjectDefs_HoistsRepeatedTypes(t *testing.T) {
	duration := func(desc string) map[string]any {
		return map[string]any{
			"type":        "object",
			"description": desc,
			"properties": map[string]any{
				"canonical": map[string]any{"type": "number"},
				"display":   map[string]any{"type": "string"},
			},
			"required":             []string{"display"},
			"additionalProperties": false,
		}
	}
	props := map[string]any{
		"title":      map[string]any{"type": "string"},
		"prep_time":  duration("prep"),
		"cook_time":  duration("cook"),
		"total_time": duration("total"),
		"steps": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timer": duration("step timer"),
				},
				"required":             []string{},
				"additionalProperties": false,
			},
		},
	}

	defs, out := factorSharedObjectDefs(props)

	if len(defs) != 1 {
		t.Fatalf("defs = %d, want exactly 1 shared duration def", len(defs))
	}
	for field, wantDesc := range map[string]string{"prep_time": "prep", "cook_time": "cook", "total_time": "total"} {
		ref, ok := out[field].(map[string]any)
		if !ok || ref["$ref"] == nil {
			t.Errorf("%s = %v, want a $ref", field, out[field])
			continue
		}
		if ref["description"] != wantDesc {
			t.Errorf(
				"%s description = %v, want %q preserved at the ref site",
				field,
				ref["description"],
				wantDesc,
			)
		}
	}
	var def map[string]any
	for _, d := range defs {
		def = d.(map[string]any)
	}
	if _, has := def["description"]; has {
		t.Error("shared def body must not carry a per-site description")
	}
	if _, ok := def["properties"].(map[string]any)["canonical"]; !ok {
		t.Error("shared def lost the duration properties")
	}
	stepItems := out["steps"].(map[string]any)["items"].(map[string]any)
	timer, ok := stepItems["properties"].(map[string]any)["timer"].(map[string]any)
	if !ok || timer["$ref"] == nil {
		t.Errorf(
			"step timer = %v, want a $ref to the shared duration",
			stepItems["properties"],
		)
	}
	if out["title"].(map[string]any)["type"] != "string" {
		t.Error("unshared scalar field should stay inline")
	}
}

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

// TestBuildOutputConfigSendsTheFactoredSchema confirms buildOutputConfig
// actually calls factorSharedObjectDefs, not just that the helper works in
// isolation.
func TestBuildOutputConfigSendsTheFactoredSchema(t *testing.T) {
	duration := func(desc string) map[string]any {
		return map[string]any{
			"type":        "object",
			"description": desc,
			"properties": map[string]any{
				"canonical": map[string]any{"type": "number"},
				"display":   map[string]any{"type": "string"},
			},
			"required":             []string{"display"},
			"additionalProperties": false,
		}
	}

	cfg := (&Client{}).buildOutputConfig(&schema.StructuredOutputInfo{
		Parameters: map[string]any{
			"prep_time":  duration("prep"),
			"cook_time":  duration("cook"),
			"total_time": duration("total"),
		},
		Required: []string{"prep_time", "cook_time", "total_time"},
	})

	sent := cfg.Format.Schema
	defs, ok := sent["$defs"].(map[string]any)
	if !ok || len(defs) == 0 {
		t.Fatalf(
			"sent schema carries no $defs -- factorSharedObjectDefs is not wired in:\n%#v",
			sent,
		)
	}
	if len(defs) != 1 {
		t.Errorf(
			"$defs holds %d entries, want 1 -- the three sites share one shape",
			len(defs),
		)
	}

	props, _ := sent["properties"].(map[string]any)
	for _, site := range []string{"prep_time", "cook_time", "total_time"} {
		node, _ := props[site].(map[string]any)
		if node["$ref"] == nil {
			t.Errorf(
				"%s was left inlined rather than referencing the shared def: %#v",
				site,
				node,
			)
		}
	}
}
