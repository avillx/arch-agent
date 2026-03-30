package llm

import (
	"embed"
	"fmt"
	"text/template"
)

//go:embed prompts/*.tmpl
var _promptTemplatesFS embed.FS // mebedded fs for prompt templates

func mustLoadPrompt[T any](templatePath string) prompt[T] {
	templ, err := template.ParseFS(_promptTemplatesFS, "prompts/"+templatePath)
	if err != nil {
		panic(fmt.Errorf("must prompt: %w", err))
	}
	return newPrompt[T](templ)
}

// reasoning
type ReasoningPromptParams struct {
	Role                 string
	Feeling              string
	Trigger              string
	Desire               string
	Trait                string
	Thoughts             string
	CommunicationContext string
	Preferences          string
	Tone                 string
	KeyPhrases           string
	BannedSentences      string
}

type ReasoningPrompt = prompt[ReasoningPromptParams]

func NewReasoningPrompt() ReasoningPrompt {
	return mustLoadPrompt[ReasoningPromptParams]("reasoning.tmpl")
}

// reflection
type ReflectionParams struct {
	Personality string
}

type ReflectionPrompt = prompt[ReflectionParams]

func NewReflectionPrompt() ReflectionPrompt {
	return mustLoadPrompt[ReflectionParams]("reflection.tmpl")
}
