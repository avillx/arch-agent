package agent

type SkillID string

type Skill interface {
	ID() SkillID
	Description() string
	Tools() []ToolName
	Prompt() string
}

// type SkillRepo interface {
// 	Get(SkillID) (Skill, error)
// }
