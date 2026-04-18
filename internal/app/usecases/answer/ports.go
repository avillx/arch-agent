package answer

type AgentRepository interface {
	Role() string
	Preferences() string
	KeyPhrases() string
	BannedSlang() string
}

type AnswerPromptRenderer interface {
	Render(AnswerPromptParams) (string, error)
}
