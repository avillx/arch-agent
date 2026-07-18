package agent

type SkillFrontmatter struct {
	ID          string
	Description string
	StoreHint   string
}

type SkillRepo interface {
	GetSkills(ID) ([]SkillFrontmatter, error)
}
