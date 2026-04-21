package answer

import (
	"arch-agent/internal/app/activity"
	"arch-agent/internal/app/knowledge"
	"arch-agent/internal/app/reflection"
	"arch-agent/internal/app/types"
	"context"
	"errors"
	"strings"
	"time"
)

type AnswerPromptParams struct {
	Role                 string
	Reflection           string
	CommunicationContext string
	Preferences          string
	KeyPhrases           string
	BannedSentences      string
	Memory               string
	Strategy             string
	Time                 string
	Knowledges           string
}

type ContextAssembler struct {
	promptRenderer    AnswerPromptRenderer
	reflectionService *reflection.Service
	agentRepo         AgentRepository
	activityService   *activity.Service
	knowledgesService *knowledge.Service
}

func NewContextAssembler(
	pr AnswerPromptRenderer,
	rs *reflection.Service,
	ar AgentRepository,
	as *activity.Service,
	ks *knowledge.Service,
) *ContextAssembler {
	return &ContextAssembler{
		promptRenderer:    pr,
		reflectionService: rs,
		agentRepo:         ar,
		activityService:   as,
		knowledgesService: ks,
	}
}

func (b *ContextAssembler) BuildPrompt(ctx context.Context, communicationContext string, msgs []types.Message) (string, error) {
	refl, err := b.reflectionService.Feelings(ctx, msgs)
	if err != nil {
		return "", err
	}

	memory, err := b.memoryAugmentation()
	if err != nil {
		return "", err
	}

	knowledges, err := b.knowledgesService.KnowledgesList()
	if err != nil {
		return "", err
	}

	return b.promptRenderer.Render(AnswerPromptParams{
		Role:                 b.agentRepo.Role(),
		Reflection:           refl,
		CommunicationContext: communicationContext,
		Preferences:          b.agentRepo.Preferences(),
		KeyPhrases:           b.agentRepo.KeyPhrases(),
		BannedSentences:      b.agentRepo.BannedSlang(),
		Memory:               memory,
		Knowledges:           knowledges,
		Strategy:             randomStrategy(),
		Time:                 b.timeAugmentation(),
	})
}

func (b *ContextAssembler) memoryAugmentation() (string, error) {
	var sb strings.Builder

	yesterday, err := b.activityService.GetActivity(time.Now().AddDate(0, 0, -1))
	if err != nil && !errors.Is(err, activity.ErrNotFound) {
		return "", err
	}

	if yesterday != "" {
		sb.WriteString("# Yesterday:\n")
		sb.WriteString(yesterday)
	}

	today, err := b.activityService.GetActivity(time.Now())
	if err != nil && !errors.Is(err, activity.ErrNotFound) {
		return "", err
	}
	if today != "" {
		sb.WriteString("# Today:\n")
		sb.WriteString(today)
	}

	return sb.String(), nil
}

func (b *ContextAssembler) timeAugmentation() string {
	return time.Now().Format(time.RFC3339)
}
