package schema

import (
	"reflect"
	"strings"
)

// GenerateSchema generates a JSON schema from a Go struct using reflection.
// It inspects struct fields and their tags to build the schema properties and required fields.
//
// Supported struct tags:
//   - json: field name in JSON (e.g., `json:"field_name"`)
//   - desc: field description (e.g., `desc:"The field description"`)
//   - enum: comma-separated enum values (e.g., `enum:"value1,value2"`)
//   - required: explicitly mark as required or not (e.g., `required:"true"` or `required:"false"`)
//
// Returns the properties map and list of required field names.
//
// OpenAI strict structured-output requires every property in `properties` to
// appear in `required` and every nested object to set `additionalProperties:
// false`. Optional fields (pointer types or `,omitempty` JSON tags) are
// represented by making their type nullable (e.g. ["string", "null"]) rather
// than by omission from `required`.
func GenerateSchema(v any) (map[string]any, []string) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	properties := make(map[string]any)
	var required []string

	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Tag.Get("json")
		if name == "" {
			name = strings.ToLower(field.Name)
		} else {
			name = strings.Split(name, ",")[0]
			if name == "-" {
				continue
			}
		}

		optional := field.Type.Kind() == reflect.Pointer ||
			strings.Contains(field.Tag.Get("json"), "omitempty") ||
			field.Tag.Get("required") == "false"

		prop := make(map[string]any)
		baseType := goTypeToJSONType(field.Type)
		if optional {
			prop["type"] = []string{baseType, "null"}
		} else {
			prop["type"] = baseType
		}

		if desc := field.Tag.Get("desc"); desc != "" {
			prop["description"] = desc
		}

		if enum := field.Tag.Get("enum"); enum != "" {
			prop["enum"] = strings.Split(enum, ",")
		}

		structType := field.Type
		if structType.Kind() == reflect.Pointer {
			structType = structType.Elem()
		}
		if structType.Kind() == reflect.Struct &&
			structType != reflect.TypeOf(struct{}{}) {
			nested, nestedReq := GenerateSchema(
				reflect.New(structType).Elem().Interface(),
			)
			if nested != nil {
				if optional {
					prop["type"] = []string{"object", "null"}
				} else {
					prop["type"] = "object"
				}
				prop["properties"] = nested
				prop["additionalProperties"] = false
				prop["required"] = nestedReq
			}
		}

		if field.Type.Kind() == reflect.Slice {
			elemType := field.Type.Elem()
			items := map[string]any{"type": goTypeToJSONType(elemType)}
			if elemType.Kind() == reflect.Struct {
				nested, nestedReq := GenerateSchema(
					reflect.New(elemType).Elem().Interface(),
				)
				if nested != nil {
					items["type"] = "object"
					items["properties"] = nested
					items["additionalProperties"] = false
					items["required"] = nestedReq
				}
			}
			prop["items"] = items
		}

		properties[name] = prop
		required = append(required, name)
	}

	return properties, required
}

func goTypeToJSONType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}
