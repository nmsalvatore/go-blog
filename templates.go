package main

import (
	"html/template"
	"strings"
)

func linebreaks(s string) template.HTML {
	s = template.HTMLEscapeString(s)

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
	pages := []string{"home.html", "detail.html", "create.html", "edit.html", "delete.html", "settings.html", "login.html"}

	funcs := template.FuncMap{
		"linebreaks": linebreaks,
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
