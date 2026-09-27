package anthropic

import (
	"reflect"
	"strings"
	"testing"

	"github.com/joakimcarlsson/ai/schema"
)

// durationSchema returns an inline duration object schema carrying desc as
// its per-site description.
func durationSchema(desc string) map[string]any {
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

// refName returns the definition name node references, or "" when node is
// not a map carrying a local $ref.
func refName(node any) string {
	m, _ := node.(map[string]any)
	ref, _ := m["$ref"].(string)
	name, ok := strings.CutPrefix(ref, defsRefPrefix)
	if !ok {
		return ""
	}
	return name
}

// TestFactorSharedObjectDefs_HoistsRepeatedTypes confirms a repeated object
// type is hoisted into a single shared $def and referenced by $ref at every
// site, keeping each site's own description.
func TestFactorSharedObjectDefs_HoistsRepeatedTypes(t *testing.T) {
	props := map[string]any{
		"title":      map[string]any{"type": "string"},
		"prep_time":  durationSchema("prep"),
		"cook_time":  durationSchema("cook"),
		"total_time": durationSchema("total"),
		"steps": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timer": durationSchema("step timer"),
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
	wantDescs := map[string]string{
		"prep_time":  "prep",
		"cook_time":  "cook",
		"total_time": "total",
	}
	for field, wantDesc := range wantDescs {
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
	stepProps := stepItems["properties"].(map[string]any)
	if refName(stepProps["timer"]) == "" {
		t.Errorf(
			"step timer = %v, want a $ref to the shared duration",
			stepProps,
		)
	}
	if out["title"].(map[string]any)["type"] != "string" {
		t.Error("unshared scalar field should stay inline")
	}
}

// TestFactorSharedObjectDefs_RewritesRefsInsideDefs confirms a shared object
// nested inside another shared object is referenced from the outer
// definition rather than inlined into it, and that every emitted definition
// is actually referenced.
func TestFactorSharedObjectDefs_RewritesRefsInsideDefs(t *testing.T) {
	step := func(desc string) map[string]any {
		return map[string]any{
			"type":        "object",
			"description": desc,
			"properties": map[string]any{
				"a": durationSchema("a"),
				"b": durationSchema("b"),
			},
			"additionalProperties": false,
		}
	}
	props := map[string]any{"x": step("x"), "y": step("y")}

	defs, out := factorSharedObjectDefs(props)

	if len(defs) != 2 {
		t.Fatalf("defs = %d, want 2 (step and duration): %#v", len(defs), defs)
	}
	stepName := refName(out["x"])
	if stepName == "" || refName(out["y"]) != stepName {
		t.Fatalf("x and y = %v, %v, want one shared $ref", out["x"], out["y"])
	}
	stepDef := defs[stepName].(map[string]any)
	stepProps := stepDef["properties"].(map[string]any)
	durName := refName(stepProps["a"])
	if durName == "" || refName(stepProps["b"]) != durName {
		t.Fatalf(
			"step def still inlines its duration fields: %#v",
			stepProps,
		)
	}
	durDef, ok := defs[durName].(map[string]any)
	if !ok {
		t.Fatalf("step def references %q, which is not emitted", durName)
	}
	if refName(durDef) != "" {
		t.Error("duration def was replaced with a reference to itself")
	}
	if _, ok := durDef["properties"].(map[string]any)["canonical"]; !ok {
		t.Error("duration def lost its properties")
	}
}

// TestFactorSharedObjectDefs_DropsUnreferencedDefs confirms a definition no
// schema node references after rewriting is not emitted.
func TestFactorSharedObjectDefs_DropsUnreferencedDefs(t *testing.T) {
	defs := map[string]any{
		"used":   map[string]any{"type": "object"},
		"unused": map[string]any{"type": "object"},
	}
	props := map[string]any{
		"x": map[string]any{"$ref": defsRefPrefix + "used"},
	}

	got := referencedDefs(props, defs)

	if len(got) != 1 || got["used"] == nil {
		t.Errorf("referencedDefs = %#v, want only the used def", got)
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

// hasNullType reports whether any "type" anywhere in the schema tree still
// lists "null".
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

// asAnySlice widens a "type" value to []any, or nil when it is not a list.
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

// TestRelaxNullableUnions_RelaxesArraysOfArrays confirms the object items of
// an array nested inside another array are relaxed too.
func TestRelaxNullableUnions_RelaxesArraysOfArrays(t *testing.T) {
	props := map[string]any{
		"grid": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"label": map[string]any{
							"type": []any{"string", "null"},
						},
					},
					"required":             []string{"label"},
					"additionalProperties": false,
				},
			},
		},
	}

	got, _ := relaxNullableUnions(props, []string{"grid"})

	if hasNullType(got) {
		t.Errorf("nested array items still carry a nullable union: %#v", got)
	}
	grid := got["grid"].(map[string]any)
	cell := grid["items"].(map[string]any)["items"].(map[string]any)
	if _, has := cell["required"]; has {
		t.Errorf("nullable cell field should be optional: %#v", cell)
	}
	if !hasNullType(props) {
		t.Error("input schema was mutated")
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
			"the schema actually sent still carries a nullable union -- "+
				"relaxNullableUnions is not wired in:\n%#v",
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
	cfg := (&Client{}).buildOutputConfig(&schema.StructuredOutputInfo{
		Parameters: map[string]any{
			"prep_time":  durationSchema("prep"),
			"cook_time":  durationSchema("cook"),
			"total_time": durationSchema("total"),
		},
		Required: []string{"prep_time", "cook_time", "total_time"},
	})

	sent := cfg.Format.Schema
	defs, ok := sent["$defs"].(map[string]any)
	if !ok || len(defs) == 0 {
		t.Fatalf(
			"sent schema carries no $defs -- "+
				"factorSharedObjectDefs is not wired in:\n%#v",
			sent,
		)
	}
	if len(defs) != 1 {
		t.Errorf(
			"$defs holds %d entries, want 1 -- "+
				"the three sites share one shape",
			len(defs),
		)
	}

	props, _ := sent["properties"].(map[string]any)
	for _, site := range []string{"prep_time", "cook_time", "total_time"} {
		node, _ := props[site].(map[string]any)
		if node["$ref"] == nil {
			t.Errorf(
				"%s was left inlined rather than "+
					"referencing the shared def: %#v",
				site,
				node,
			)
		}
	}
}
