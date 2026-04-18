package llm

import (
	"arch-agent/internal/app/reflection"
	"arch-agent/internal/app/usecases/answer"
)

// answer prompt
func NewAnswerPrompt() prompt[answer.AnswerPromptParams] {
	return mustLoadPrompt[answer.AnswerPromptParams]("reasoning.tmpl")
}

// reflection prompt
func NewReflectionPrompt() prompt[reflection.ReflectionParams] {
	return mustLoadPrompt[reflection.ReflectionParams]("reflection.tmpl")
}

// summarizaton prompt
type SummarizationPrompt struct {
	prompt[struct{}]
}

func NewSummaryPrompt() *SummarizationPrompt {
	return &SummarizationPrompt{
		prompt: mustLoadPrompt[struct{}]("summary.tmpl"),
	}
}

func (p *SummarizationPrompt) Render() (string, error) {
	return p.prompt.Render(struct{}{})
}

// dream prompt
type DreamPrompt struct {
	prompt[struct{}]
}

func NewDreamPrompt() DreamPrompt {
	return DreamPrompt{
		prompt: mustLoadPrompt[struct{}]("dreaming.tmpl"),
	}
}

func (p *DreamPrompt) Render() (string, error) {
	return p.prompt.Render(struct{}{})
}
