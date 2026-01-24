package prompt

import (
	"reflect"
	"strings"
	"text/template"
)

// DefaultFuncMap contains all built-in template functions.
var DefaultFuncMap = template.FuncMap{
	"eq":  eq,
	"ne":  ne,
	"neq": ne,
	"lt":  lt,
	"le":  le,
	"gt":  gt,
	"ge":  ge,

	"upper":      strings.ToUpper,
	"lower":      strings.ToLower,
	"title":      strings.ToTitle,
	"trim":       strings.TrimSpace,
	"trimPrefix": strings.TrimPrefix,
	"trimSuffix": strings.TrimSuffix,
	"replace":    replace,
	"contains":   strings.Contains,
	"hasPrefix":  strings.HasPrefix,
	"hasSuffix":  strings.HasSuffix,

	"join":  join,
	"split": strings.Split,
	"first": first,
	"last":  last,
	"list":  list,

	"default":  defaultVal,
	"coalesce": coalesce,
	"empty":    empty,
	"ternary":  ternary,

	"indent":  indent,
	"nindent": nindent,
	"quote":   quote,
	"squote":  squote,
}

func eq(a, b any) bool  { return a == b }
func ne(a, b any) bool  { return a != b }
func lt(a, b any) bool  { return toFloat(a) < toFloat(b) }
func le(a, b any) bool  { return toFloat(a) <= toFloat(b) }
func gt(a, b any) bool  { return toFloat(a) > toFloat(b) }
func ge(a, b any) bool  { return toFloat(a) >= toFloat(b) }

func toFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

func replace(old, new, s string) string {
	return strings.ReplaceAll(s, old, new)
}

func join(sep string, v any) string {
	switch s := v.(type) {
	case []string:
		return strings.Join(s, sep)
	case []any:
		strs := make([]string, len(s))
		for i, item := range s {
			if str, ok := item.(string); ok {
				strs[i] = str
			}
		}
		return strings.Join(strs, sep)
	default:
		return ""
	}
}

func first(v any) any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice && val.Len() > 0 {
		return val.Index(0).Interface()
	}
	return nil
}

func last(v any) any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Slice && val.Len() > 0 {
		return val.Index(val.Len() - 1).Interface()
	}
	return nil
}

func list(args ...any) []any {
	return args
}

func defaultVal(def, val any) any {
	if empty(val) {
		return def
	}
	return val
}

func coalesce(args ...any) any {
	for _, v := range args {
		if !empty(v) {
			return v
		}
	}
	return nil
}

func empty(v any) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return val.IsNil()
	default:
		return false
	}
}

func ternary(cond bool, trueVal, falseVal any) any {
	if cond {
		return trueVal
	}
	return falseVal
}

func indent(spaces int, s string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

func nindent(spaces int, s string) string {
	return "\n" + indent(spaces, s)
}

func quote(s string) string {
	return `"` + s + `"`
}

func squote(s string) string {
	return "'" + s + "'"
}
