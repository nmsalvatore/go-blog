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
			name:  "bold",
			input: "This is **bold** text",
			want:  "<p>This is <strong>bold</strong> text</p>",
		},
		{
			name:  "multiple bold",
			input: "Both **first** and **second** are bold",
			want:  "<p>Both <strong>first</strong> and <strong>second</strong> are bold</p>",
		},
		{
			name:  "bold and italic combined",
			input: "This is **bold** and *italic* text",
			want:  "<p>This is <strong>bold</strong> and <em>italic</em> text</p>",
		},
		{
			name:  "link",
			input: "Check out [my site](https://example.com) for more",
			want:  `<p>Check out <a href="https://example.com" target="_blank" rel="noopener">my site</a> for more</p>`,
		},
		{
			name:  "link with http",
			input: "Visit [here](http://example.com)",
			want:  `<p>Visit <a href="http://example.com" target="_blank" rel="noopener">here</a></p>`,
		},
		{
			name:  "mailto link",
			input: "Email [me](mailto:test@example.com)",
			want:  `<p>Email <a href="mailto:test@example.com" target="_blank" rel="noopener">me</a></p>`,
		},
		{
			name:  "javascript link blocked",
			input: "Click [here](javascript:alert('xss'))",
			want:  "<p>Click [here](javascript:alert(&#39;xss&#39;))</p>",
		},
		{
			name:  "multiple links",
			input: "See [one](https://one.com) and [two](https://two.com)",
			want:  `<p>See <a href="https://one.com" target="_blank" rel="noopener">one</a> and <a href="https://two.com" target="_blank" rel="noopener">two</a></p>`,
		},
		{
			name:  "uppercase HTTP allowed",
			input: "Visit [here](HTTP://example.com)",
			want:  `<p>Visit <a href="HTTP://example.com" target="_blank" rel="noopener">here</a></p>`,
		},
		{
			name:  "data URI blocked",
			input: "[click](data:text/html,<script>alert('xss')</script>)",
			want:  "<p>[click](data:text/html,&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;)</p>",
		},
		{
			name:  "attribute injection blocked via escaping",
			input: `[click](https://example.com" onclick="alert('xss'))`,
			want:  `<p><a href="https://example.com&#34; onclick=&#34;alert(&#39;xss&#39;)" target="_blank" rel="noopener">click</a></p>`,
		},
		{
			name:  "link with bold text",
			input: "Check [**bold link**](https://example.com)",
			want:  `<p>Check <a href="https://example.com" target="_blank" rel="noopener"><strong>bold link</strong></a></p>`,
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
