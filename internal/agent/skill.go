package agent

type SkillID string

type Skill interface {
	ID() SkillID
	Description() string
	Tools() []ToolName
	Prompt() string
}

type skill struct {
	id          SkillID
	description string
	tools       []ToolName
	prompt      string
}

func NewSkill(
	id SkillID,
	description string,
	tools []ToolName,
	prompt string,
) *skill {
	return &skill{
		id:          id,
		description: description,
		tools:       tools,
		prompt:      prompt,
	}
}

func (s *skill) ID() SkillID         { return s.id }
func (s *skill) Description() string { return s.description }
func (s *skill) Tools() []ToolName   { return s.tools }
func (s *skill) Prompt() string      { return s.prompt }
