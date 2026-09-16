package schema

import (
	"reflect"
	"testing"
)

type enumHolder struct {
	Status string   `json:"status" enum:"open,closed"`
	Tags   []string `json:"tags"   enum:"red,green,blue"`
	Plain  []string `json:"plain"`
}

func propOf(t *testing.T, props map[string]any, name string) map[string]any {
	t.Helper()

	prop, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("property %q is missing or not an object", name)
	}
	return prop
}

func TestGenerateSchemaEnumOnScalarStaysOnProperty(t *testing.T) {
	props, _ := GenerateSchema(enumHolder{})

	status := propOf(t, props, "status")
	want := []string{"open", "closed"}
	if got, ok := status["enum"].([]string); !ok ||
		!reflect.DeepEqual(got, want) {
		t.Fatalf("status enum = %v, want %v", status["enum"], want)
	}
}

func TestGenerateSchemaEnumOnSliceMovesToItems(t *testing.T) {
	props, _ := GenerateSchema(enumHolder{})

	tags := propOf(t, props, "tags")
	if _, ok := tags["enum"]; ok {
		t.Error("enum stayed on the array, which no array value satisfies")
	}

	items, ok := tags["items"].(map[string]any)
	if !ok {
		t.Fatal("tags has no items")
	}

	want := []string{"red", "green", "blue"}
	if got, ok := items["enum"].([]string); !ok ||
		!reflect.DeepEqual(got, want) {
		t.Fatalf("items enum = %v, want %v", items["enum"], want)
	}
	if items["type"] != "string" {
		t.Errorf("items type = %v, want string", items["type"])
	}
	if tags["type"] != "array" {
		t.Errorf("tags type = %v, want array", tags["type"])
	}
}

func TestGenerateSchemaSliceWithoutEnumIsUnchanged(t *testing.T) {
	props, _ := GenerateSchema(enumHolder{})

	plain := propOf(t, props, "plain")
	items, ok := plain["items"].(map[string]any)
	if !ok {
		t.Fatal("plain has no items")
	}
	if _, ok := items["enum"]; ok {
		t.Error("items gained an enum from a field that declared none")
	}
}
