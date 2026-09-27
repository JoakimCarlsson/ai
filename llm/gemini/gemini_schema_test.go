package gemini

import (
	"slices"
	"testing"

	"github.com/joakimcarlsson/ai/schema"
	"google.golang.org/genai"
)

// nestedAddress is an embedded object used to exercise nested-property
// recursion in the structured-output schema converter.
type nestedAddress struct {
	City string `json:"city" desc:"City name"`
	Zip  string `json:"zip"  desc:"Postal code"`
}

// lineItem is the element type of a []struct field, exercising array-item
// object recursion.
type lineItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// structuredOutputFixture covers all four shapes the converter must handle:
// (i) a nested object, (ii) a []struct, (iii) an enum field, and (iv) an
// optional pointer field that the generator renders as a ["string","null"]
// union.
type structuredOutputFixture struct {
	Status   string        `json:"status"             enum:"active,inactive,pending" desc:"Account status"`
	Address  nestedAddress `json:"address"                                           desc:"Mailing address"`
	Items    []lineItem    `json:"items"                                             desc:"Line items"`
	Nickname *string       `json:"nickname,omitempty"                                desc:"Optional nickname"`
}

// TestConvertSchemaToGenaiNested verifies the structured-output converter
// recurses into nested objects, array-item objects, enum lists, nested
// required, and nullable unions instead of dropping them (the shallow-copy bug
// it replaces).
func TestConvertSchemaToGenaiNested(t *testing.T) {
	info := schema.NewStructuredOutputFromStruct(
		"fixture",
		"structured output fixture",
		structuredOutputFixture{},
	)

	c := &Client{}
	got := c.convertSchemaToGenai(info.Parameters, info.Required)

	if got.Type != genai.TypeObject {
		t.Fatalf("top-level Type = %v, want %v", got.Type, genai.TypeObject)
	}

	addr, ok := got.Properties["address"]
	if !ok {
		t.Fatal("expected 'address' property")
	}
	if addr.Type != genai.TypeObject {
		t.Errorf("address.Type = %v, want %v", addr.Type, genai.TypeObject)
	}
	if len(addr.Properties) == 0 {
		t.Fatal(
			"expected address.Properties to be populated (nested object dropped)",
		)
	}
	if _, ok := addr.Properties["city"]; !ok {
		t.Errorf("expected address.Properties[city], got %v", addr.Properties)
	}
	if !slices.Contains(addr.Required, "city") ||
		!slices.Contains(addr.Required, "zip") {
		t.Errorf("address.Required = %v, want city and zip", addr.Required)
	}

	items, ok := got.Properties["items"]
	if !ok {
		t.Fatal("expected 'items' property")
	}
	if items.Type != genai.TypeArray {
		t.Errorf("items.Type = %v, want %v", items.Type, genai.TypeArray)
	}
	if items.Items == nil {
		t.Fatal("expected items.Items to be set")
	}
	if items.Items.Type != genai.TypeObject {
		t.Errorf(
			"items.Items.Type = %v, want %v",
			items.Items.Type,
			genai.TypeObject,
		)
	}
	if len(items.Items.Properties) == 0 {
		t.Fatal(
			"expected items.Items.Properties to be populated (array-item object dropped)",
		)
	}
	if _, ok := items.Items.Properties["sku"]; !ok {
		t.Errorf(
			"expected items.Items.Properties[sku], got %v",
			items.Items.Properties,
		)
	}

	status, ok := got.Properties["status"]
	if !ok {
		t.Fatal("expected 'status' property")
	}
	if len(status.Enum) != 3 {
		t.Errorf("status.Enum = %v, want 3 values", status.Enum)
	}
	if !slices.Contains(status.Enum, "active") ||
		!slices.Contains(status.Enum, "pending") {
		t.Errorf("status.Enum = %v, want active/inactive/pending", status.Enum)
	}

	nick, ok := got.Properties["nickname"]
	if !ok {
		t.Fatal("expected 'nickname' property")
	}
	if nick.Type != genai.TypeString {
		t.Errorf("nickname.Type = %v, want %v", nick.Type, genai.TypeString)
	}
	if nick.Nullable == nil || !*nick.Nullable {
		t.Errorf("expected nickname.Nullable = true, got %v", nick.Nullable)
	}

	if status.Nullable != nil {
		t.Errorf("expected status.Nullable to be nil, got %v", *status.Nullable)
	}
}

// TestConvertSchemaToGenaiTopLevelRequired verifies the top-level required list
// passes through unchanged.
func TestConvertSchemaToGenaiTopLevelRequired(t *testing.T) {
	info := schema.NewStructuredOutputFromStruct(
		"fixture",
		"structured output fixture",
		structuredOutputFixture{},
	)
	got := (&Client{}).convertSchemaToGenai(info.Parameters, info.Required)
	for _, name := range []string{"status", "address", "items", "nickname"} {
		if !slices.Contains(got.Required, name) {
			t.Errorf(
				"top-level Required missing %q; got %v",
				name,
				got.Required,
			)
		}
	}
}

// enumTypesFixture puts an enum tag on a string and on an integer field; the
// generator emits both as string values.
type enumTypesFixture struct {
	Color    string `json:"color"    enum:"red,green"`
	Priority int    `json:"priority" enum:"1,2,3"`
}

// TestConvertSchemaToGenaiEnumOnlyOnStrings verifies an enum reaches Gemini
// only on a STRING field: Gemini rejects enum on any other type, and the
// generator renders an integer field's enum as strings.
func TestConvertSchemaToGenaiEnumOnlyOnStrings(t *testing.T) {
	info := schema.NewStructuredOutputFromStruct(
		"fixture",
		"enum types fixture",
		enumTypesFixture{},
	)
	got := (&Client{}).convertSchemaToGenai(info.Parameters, info.Required)

	if color := got.Properties["color"]; len(color.Enum) != 2 {
		t.Errorf("color.Enum = %v, want red and green", color.Enum)
	}
	priority := got.Properties["priority"]
	if priority.Type != genai.TypeInteger {
		t.Errorf(
			"priority.Type = %v, want %v",
			priority.Type,
			genai.TypeInteger,
		)
	}
	if priority.Enum != nil {
		t.Errorf("priority.Enum = %v, want none on an integer", priority.Enum)
	}
}
