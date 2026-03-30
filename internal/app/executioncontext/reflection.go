package executioncontext

import (
	"arch-agent/internal/domain/conversation"
	"context"
)

type Reflection struct {
	Trigger        string
	Traits         string
	Feeling        string
	Desire         string
	InnerMonologue string
	Tone           string
}

func NewReflection(
	trigger string,
	traits string,
	feeling string,
	desire string,
	innerMonologue string,
	tone string,
) Reflection {
	return Reflection{
		Trigger:        trigger,
		Traits:         traits,
		Feeling:        feeling,
		Desire:         desire,
		InnerMonologue: innerMonologue,
		Tone:           tone,
	}
}

type Reflector interface {
	Reflect(
		ctx context.Context,
		conversation []conversation.Message,
		personality string,
	) (*Reflection, error)
}
