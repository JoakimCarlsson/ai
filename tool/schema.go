package tool

import (
	"reflect"
	"strings"
)

func GenerateSchema(v any) (map[string]any, []string) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	properties := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
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

		prop := make(map[string]any)
		prop["type"] = goTypeToJSONType(field.Type)

		if desc := field.Tag.Get("desc"); desc != "" {
			prop["description"] = desc
		}

		if enum := field.Tag.Get("enum"); enum != "" {
			prop["enum"] = strings.Split(enum, ",")
		}

		if field.Type.Kind() == reflect.Struct && field.Type != reflect.TypeOf(struct{}{}) {
			nested, nestedReq := GenerateSchema(reflect.New(field.Type).Elem().Interface())
			if nested != nil {
				prop["type"] = "object"
				prop["properties"] = nested
				if len(nestedReq) > 0 {
					prop["required"] = nestedReq
				}
			}
		}

		if field.Type.Kind() == reflect.Slice {
			elemType := field.Type.Elem()
			items := map[string]any{"type": goTypeToJSONType(elemType)}
			if elemType.Kind() == reflect.Struct {
				nested, nestedReq := GenerateSchema(reflect.New(elemType).Elem().Interface())
				if nested != nil {
					items["type"] = "object"
					items["properties"] = nested
					if len(nestedReq) > 0 {
						items["required"] = nestedReq
					}
				}
			}
			prop["items"] = items
		}

		properties[name] = prop

		if field.Tag.Get("required") == "true" {
			required = append(required, name)
		} else if field.Type.Kind() != reflect.Ptr && !strings.Contains(field.Tag.Get("json"), "omitempty") {
			if field.Tag.Get("required") != "false" {
				required = append(required, name)
			}
		}
	}

	return properties, required
}

func goTypeToJSONType(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
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

