package executioncontext

type AgentRepository interface {
	Get() AgentConfig
}

type AgentConfig struct {
	Role        string
	Personality string
	Preferences string
	Keyphrases  string
	BannedSlang string
}

func NewAgent(role, personality, preferences, keyphrases, bannedSlang string) *AgentConfig {
	return &AgentConfig{
		Role:        role,
		Personality: personality,
		Preferences: preferences,
		Keyphrases:  keyphrases,
		BannedSlang: bannedSlang,
	}
}
