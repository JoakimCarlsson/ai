package agent

import (
	"strings"
	"text/template"
)

var defaultFuncMap = template.FuncMap{
	"eq":  func(a, b any) bool { return a == b },
	"ne":  func(a, b any) bool { return a != b },
	"neq": func(a, b any) bool { return a != b },
}

func processTemplate(tmplStr string, state map[string]any) (string, error) {
	if state == nil {
		state = make(map[string]any)
	}

	tmpl, err := template.New("prompt").Funcs(defaultFuncMap).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, state); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ProcessTemplate(tmplStr string, state map[string]any) (string, error) {
	return processTemplate(tmplStr, state)
}
