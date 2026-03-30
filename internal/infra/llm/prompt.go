package llm

import (
	"bytes"
	"text/template"
)

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
