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
	Reflection           string
	CommunicationContext string
	Preferences          string
	KeyPhrases           string
	BannedSentences      string
	Memory               string
	Strategy             string
	Time                 string
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

// summary
type SummaryParams struct{}

type SummaryPrompt = prompt[SummaryParams]

func NewSummaryPrompt() SummaryPrompt {
	return mustLoadPrompt[SummaryParams]("summary.tmpl")
}
