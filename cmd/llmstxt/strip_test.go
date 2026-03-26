package main

import (
	"testing"
)

func TestStripMkDocsSyntax(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "admonition with title",
			in:   "!!! note \"Important\"\n    This is a note.\n    Second line.\n\nNext paragraph.",
			want: "> **Important:**\n> This is a note.\n> Second line.\n>\nNext paragraph.",
		},
		{
			name: "admonition without title",
			in:   "!!! warning\n    Be careful.",
			want: "> **Warning:**\n> Be careful.",
		},
		{
			name: "collapsible admonition",
			in:   "??? tip \"Hint\"\n    Hidden content.",
			want: "> **Hint:**\n> Hidden content.",
		},
		{
			name: "collapsible open admonition",
			in:   "???+ note \"Open\"\n    Visible content.",
			want: "> **Open:**\n> Visible content.",
		},
		{
			name: "tab syntax",
			in:   "=== \"Python\"\n\n```python\nprint(\"hi\")\n```\n\n=== \"Go\"\n\n```go\nfmt.Println(\"hi\")\n```",
			want: "**Python:**\n\n```python\nprint(\"hi\")\n```\n\n**Go:**\n\n```go\nfmt.Println(\"hi\")\n```",
		},
		{
			name: "attribute list stripped",
			in:   "![image](img.png){: .center }",
			want: "![image](img.png)",
		},
		{
			name: "code block passthrough",
			in:   "```\n!!! note\n=== \"Tab\"\n{: .class }\n```",
			want: "```\n!!! note\n=== \"Tab\"\n{: .class }\n```",
		},
		{
			name: "plain markdown unchanged",
			in:   "# Hello\n\nSome **bold** text.\n\n- item 1\n- item 2",
			want: "# Hello\n\nSome **bold** text.\n\n- item 1\n- item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMkDocsSyntax(tt.in)
			if got != tt.want {
				t.Errorf(
					"stripMkDocsSyntax():\ngot:\n%s\n\nwant:\n%s",
					got,
					tt.want,
				)
			}
		})
	}
}
