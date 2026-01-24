package main

import (
	"html/template"
	"net/url"
	"regexp"
	"strings"
)

var boldRegex = regexp.MustCompile(`\*\*([^*]+)\*\*`)
var italicRegex = regexp.MustCompile(`\*([^*]+)\*`)
var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(((?:[^()]+|\([^()]*\))+)\)`)

func format(s string) template.HTML {
	s = template.HTMLEscapeString(s)
	s = linkRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := linkRegex.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		text, rawURL := parts[1], parts[2]
		// Parse and validate URL scheme
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return match
		}
		scheme := strings.ToLower(parsedURL.Scheme)
		if scheme != "http" && scheme != "https" && scheme != "mailto" {
			return match
		}
		return `<a href="` + rawURL + `" target="_blank" rel="noopener">` + text + `</a>`
	})
	s = boldRegex.ReplaceAllString(s, "<strong>$1</strong>")
	s = italicRegex.ReplaceAllString(s, "<em>$1</em>")

	paragraphs := strings.Split(s, "\n\n")
	var result []string

	for _, p := range paragraphs {
		if p = strings.TrimSpace(p); p != "" {
			p = strings.ReplaceAll(p, "\n", "<br>")
			result = append(result, "<p>"+p+"</p>")
		}
	}

	return template.HTML(strings.Join(result, "\n"))
}

func loadTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	pages := []string{"home.html", "detail.html", "create.html", "edit.html", "delete.html", "settings.html", "admin.html"}

	funcs := template.FuncMap{
		"format": format,
	}

	for _, page := range pages {
		templates[page] = template.Must(
			template.New("").Funcs(funcs).ParseFiles(
				"templates/base.html",
				"templates/"+page,
			))
	}

	return templates
}
