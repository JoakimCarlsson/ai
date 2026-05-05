# User Prompt

Write a {{.format}} about {{quote .topic}}.

Context:
{{.context | nindent 2}}

Requirements:
{{range .requirements}}- {{.}}
{{end}}
