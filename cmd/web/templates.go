package main

import (
	"path/filepath"
	"snippety/internal/models"
	"text/template"
)

type templateData struct {
	CurrentYear int
	Snippet models.Snippet
	Snippets []models.Snippet
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// Slice of string of all filespaths matching pattern
	pages, err := filepath.Glob("./ui/html/pages/*.tmpl.html")
    if err != nil {
        return nil, err
    }

	// Add each template set for each page to cache
	for _, page := range pages {
		// Extract last segment from full path: a/b/x.html -> x.html
		name := filepath.Base(page)

		// Parse base
		ts, err := template.ParseFiles("./ui/html/base.tmpl.html")
		if err != nil {
			return nil, err
		}

		// Parse partials
		ts, err = ts.ParseGlob("./ui/html/partials/*.tmpl.html")
		if err != nil {
			return nil, err
		}

		// Parse page
		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}