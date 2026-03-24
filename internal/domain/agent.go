package domain

type Agent struct {
	role        string
	personality string
	preferences string
	keyphrases  string
	bannedSlang string
}

func NewAgent(role, personality, preferences, keyphrases, bannedSlang string) *Agent {
	return &Agent{
		role:        role,
		personality: personality,
		preferences: preferences,
		keyphrases:  keyphrases,
		bannedSlang: bannedSlang,
	}
}
