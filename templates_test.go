package main

import (
	"html/template"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  template.HTML
	}{
		{
			name:  "single paragraph",
			input: "Hello world",
			want:  "<p>Hello world</p>",
		},
		{
			name:  "two paragraphs",
			input: "First paragraph\n\nSecond paragraph",
			want:  "<p>First paragraph</p>\n<p>Second paragraph</p>",
		},
		{
			name:  "line break within paragraph",
			input: "Line one\nLine two",
			want:  "<p>Line one<br>Line two</p>",
		},
		{
			name:  "italics",
			input: "This is *italic* text",
			want:  "<p>This is <em>italic</em> text</p>",
		},
		{
			name:  "multiple italics",
			input: "Both *first* and *second* are italic",
			want:  "<p>Both <em>first</em> and <em>second</em> are italic</p>",
		},
		{
			name:  "italics across paragraphs",
			input: "First *italic*\n\nSecond *italic*",
			want:  "<p>First <em>italic</em></p>\n<p>Second <em>italic</em></p>",
		},
		{
			name:  "html escaped",
			input: "<script>alert('xss')</script>",
			want:  "<p>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;</p>",
		},
		{
			name:  "unmatched asterisk preserved",
			input: "This has a * single asterisk",
			want:  "<p>This has a * single asterisk</p>",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format(tt.input)
			if got != tt.want {
				t.Errorf("format() = %q, want %q", got, tt.want)
			}
		})
	}
}
