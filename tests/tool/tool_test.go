package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/joakimcarlsson/ai/tool"
)

type stubTool struct {
	name   string
	output string
}

func (s *stubTool) Info() tool.Info {
	return tool.NewInfo(s.name, "stub tool", struct{}{})
}

func (s *stubTool) Run(
	_ context.Context,
	_ tool.Call,
) (tool.Response, error) {
	return tool.NewTextResponse(s.output), nil
}

func TestRegistry_ExecuteFound(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&stubTool{name: "greet", output: "hello"})

	resp, err := reg.Execute(
		context.Background(),
		tool.Call{ID: "1", Name: "greet", Input: "{}"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Content)
	}
	if resp.IsError {
		t.Error("expected IsError=false")
	}
}

func TestRegistry_ExecuteNotFound(t *testing.T) {
	reg := tool.NewRegistry()

	resp, err := reg.Execute(
		context.Background(),
		tool.Call{ID: "1", Name: "missing"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsError {
		t.Error("expected IsError=true for missing tool")
	}
	if resp.Content == "" {
		t.Error("expected error message in content")
	}
}

func TestRegistry_GetAndList(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(&stubTool{name: "a", output: ""})
	reg.Register(&stubTool{name: "b", output: ""})

	if _, ok := reg.Get("a"); !ok {
		t.Error("expected to find tool 'a'")
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("expected not to find 'nonexistent'")
	}
	if len(reg.List()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(reg.List()))
	}
	if len(reg.Names()) != 2 {
		t.Errorf("expected 2 names, got %d", len(reg.Names()))
	}
}

func TestNewTextResponse(t *testing.T) {
	r := tool.NewTextResponse("ok")
	if r.Type != tool.ResponseTypeText {
		t.Errorf("expected type text, got %s", r.Type)
	}
	if r.Content != "ok" {
		t.Errorf("expected 'ok', got %q", r.Content)
	}
	if r.IsError {
		t.Error("expected IsError=false")
	}
}

func TestNewTextErrorResponse(t *testing.T) {
	r := tool.NewTextErrorResponse("bad")
	if !r.IsError {
		t.Error("expected IsError=true")
	}
	if r.Content != "bad" {
		t.Errorf("expected 'bad', got %q", r.Content)
	}
}

func TestNewJSONResponse_Success(t *testing.T) {
	r := tool.NewJSONResponse(map[string]int{"count": 42})
	if r.Type != tool.ResponseTypeJSON {
		t.Errorf("expected type json, got %s", r.Type)
	}
	if r.IsError {
		t.Error("expected IsError=false")
	}

	var result map[string]int
	if err := json.Unmarshal([]byte(r.Content), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["count"] != 42 {
		t.Errorf("expected count=42, got %d", result["count"])
	}
}

func TestNewJSONResponse_MarshalError(t *testing.T) {
	r := tool.NewJSONResponse(make(chan int))
	if !r.IsError {
		t.Error("expected IsError=true for unmarshalable value")
	}
}

func TestNewImageResponse(t *testing.T) {
	r := tool.NewImageResponse("http://example.com/img.png")
	if r.Type != tool.ResponseTypeImage {
		t.Errorf("expected type image, got %s", r.Type)
	}
	if r.IsError {
		t.Error("expected IsError=false")
	}
}

func TestNewFileResponse(t *testing.T) {
	r := tool.NewFileResponse(
		[]byte("pdf data"),
		"application/pdf",
	)
	if r.Type != tool.ResponseTypeFile {
		t.Errorf("expected type file, got %s", r.Type)
	}
	if r.MimeType != "application/pdf" {
		t.Errorf(
			"expected mime application/pdf, got %q",
			r.MimeType,
		)
	}
}

func TestWithResponseMetadata(t *testing.T) {
	base := tool.NewTextResponse("ok")
	r := tool.WithResponseMetadata(
		base,
		map[string]string{"key": "val"},
	)
	if r.Metadata == "" {
		t.Fatal("expected non-empty metadata")
	}

	var meta map[string]string
	if err := json.Unmarshal(
		[]byte(r.Metadata),
		&meta,
	); err != nil {
		t.Fatalf("metadata not valid JSON: %v", err)
	}
	if meta["key"] != "val" {
		t.Errorf("expected key=val, got %q", meta["key"])
	}
}

func TestWithResponseMetadata_Nil(t *testing.T) {
	base := tool.NewTextResponse("ok")
	r := tool.WithResponseMetadata(base, nil)
	if r.Metadata != "" {
		t.Error("expected empty metadata for nil input")
	}
}

func TestGenerateSchema_BasicStruct(t *testing.T) {
	type Input struct {
		Name string `json:"name" desc:"The name"`
		Age  int    `json:"age"`
	}

	props, required := tool.GenerateSchema(Input{})
	if props == nil {
		t.Fatal("expected non-nil properties")
	}
	if len(required) != 2 {
		t.Errorf("expected 2 required, got %d", len(required))
	}

	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("expected name property")
	}
	if nameProp["type"] != "string" {
		t.Errorf(
			"expected name type string, got %v",
			nameProp["type"],
		)
	}
	if nameProp["description"] != "The name" {
		t.Errorf(
			"expected description 'The name', got %v",
			nameProp["description"],
		)
	}
}

func TestGenerateSchema_PointerInput(t *testing.T) {
	type Input struct {
		Val string `json:"val"`
	}
	props, _ := tool.GenerateSchema(&Input{})
	if props == nil {
		t.Fatal("expected non-nil properties for pointer input")
	}
}

func TestGenerateSchema_NonStruct(t *testing.T) {
	props, required := tool.GenerateSchema("string")
	if props != nil {
		t.Error("expected nil properties for non-struct")
	}
	if required != nil {
		t.Error("expected nil required for non-struct")
	}
}

func TestGenerateSchema_JsonDash(t *testing.T) {
	type Input struct {
		Visible  string `json:"visible"`
		Excluded string `json:"-"`
	}

	props, _ := tool.GenerateSchema(Input{})
	if _, ok := props["-"]; ok {
		t.Error("json:\"-\" field should be excluded")
	}
	if _, ok := props["visible"]; !ok {
		t.Error("expected 'visible' property")
	}
}

func TestGenerateSchema_EnumTag(t *testing.T) {
	type Input struct {
		Color string `json:"color" enum:"red,green,blue"`
	}

	props, _ := tool.GenerateSchema(Input{})
	colorProp := props["color"].(map[string]any)
	enum, ok := colorProp["enum"].([]string)
	if !ok {
		t.Fatal("expected enum slice")
	}
	if len(enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(enum))
	}
}

func TestGenerateSchema_OptionalPointerField(t *testing.T) {
	type Input struct {
		Required string  `json:"required"`
		Optional *string `json:"optional"`
	}

	_, required := tool.GenerateSchema(Input{})

	hasRequired := false
	hasOptional := false
	for _, r := range required {
		if r == "required" {
			hasRequired = true
		}
		if r == "optional" {
			hasOptional = true
		}
	}
	if !hasRequired {
		t.Error("expected 'required' in required list")
	}
	if hasOptional {
		t.Error("pointer field should not be required")
	}
}

func TestGenerateSchema_OmitemptyNotRequired(t *testing.T) {
	type Input struct {
		Always   string `json:"always"`
		Optional string `json:"optional,omitempty"`
	}

	_, required := tool.GenerateSchema(Input{})
	for _, r := range required {
		if r == "optional" {
			t.Error("omitempty field should not be required")
		}
	}
}

func TestGenerateSchema_RequiredFalseTag(t *testing.T) {
	type Input struct {
		Field string `json:"field" required:"false"`
	}

	_, required := tool.GenerateSchema(Input{})
	if len(required) != 0 {
		t.Errorf(
			"required:\"false\" should exclude, got %v",
			required,
		)
	}
}

func TestGenerateSchema_NestedStruct(t *testing.T) {
	type Address struct {
		City string `json:"city"`
	}
	type Input struct {
		Home Address `json:"home"`
	}

	props, _ := tool.GenerateSchema(Input{})
	homeProp := props["home"].(map[string]any)
	if homeProp["type"] != "object" {
		t.Errorf(
			"expected nested type object, got %v",
			homeProp["type"],
		)
	}
	nested, ok := homeProp["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected nested properties")
	}
	if _, ok := nested["city"]; !ok {
		t.Error("expected city in nested properties")
	}
}

func TestGenerateSchema_SliceOfStructs(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type Input struct {
		Items []Item `json:"items"`
	}

	props, _ := tool.GenerateSchema(Input{})
	itemsProp := props["items"].(map[string]any)
	if itemsProp["type"] != "array" {
		t.Errorf("expected array type, got %v", itemsProp["type"])
	}
	items := itemsProp["items"].(map[string]any)
	if items["type"] != "object" {
		t.Errorf(
			"expected items type object, got %v",
			items["type"],
		)
	}
}

func TestGenerateSchema_SliceOfPrimitives(t *testing.T) {
	type Input struct {
		Tags []string `json:"tags"`
	}

	props, _ := tool.GenerateSchema(Input{})
	tagsProp := props["tags"].(map[string]any)
	items := tagsProp["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf(
			"expected items type string, got %v",
			items["type"],
		)
	}
}

func TestGenerateSchema_UnexportedFieldSkipped(t *testing.T) {
	type Input struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}

	props, _ := tool.GenerateSchema(Input{})
	if len(props) != 1 {
		t.Errorf(
			"expected 1 property (only public), got %d",
			len(props),
		)
	}
}

func TestGenerateSchema_NoJsonTagUsesLowercase(t *testing.T) {
	type Input struct {
		MyField string
	}

	props, _ := tool.GenerateSchema(Input{})
	if _, ok := props["myfield"]; !ok {
		t.Error(
			"expected lowercase field name when no json tag",
		)
	}
}

func TestNewInfo_SchemaIntegration(t *testing.T) {
	type Params struct {
		Query string `json:"query" desc:"Search query"`
		Limit int    `json:"limit"`
	}

	info := tool.NewInfo("search", "Search things", Params{})
	if info.Name != "search" {
		t.Errorf("expected name 'search', got %q", info.Name)
	}
	if info.Description != "Search things" {
		t.Errorf("wrong description: %q", info.Description)
	}
	if len(info.Required) != 2 {
		t.Errorf("expected 2 required, got %d", len(info.Required))
	}
	if info.Parameters == nil {
		t.Fatal("expected non-nil parameters")
	}
	if _, ok := info.Parameters["query"]; !ok {
		t.Error("expected 'query' in parameters")
	}
}
