package llm

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed prompts/*.tmpl
var _promptTemplatesFS embed.FS

func mustLoadPrompt[T any](templatePath string) prompt[T] {
	templ, err := template.ParseFS(_promptTemplatesFS, "prompts/"+templatePath)
	if err != nil {
		panic(fmt.Errorf("must prompt: %w", err))
	}
	return newPrompt[T](templ)
}

type prompt[PromptParams any] struct {
	template *template.Template
}

func newPrompt[PromptParams any](templ *template.Template) prompt[PromptParams] {
	return prompt[PromptParams]{
		template: templ,
	}
}

func (p prompt[PromptParams]) Render(params PromptParams) (string, error) {
	// tmpl, _ := template.ParseFiles("templates/index.html")
	var buf bytes.Buffer
	if err := p.template.Execute(&buf, params); err != nil {
		return "", err
	}

	return buf.String(), nil
}
