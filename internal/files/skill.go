package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const skillsFolder = "/files/skills"
const skillFile = "SKILL.md"

var _ runtime.SkillIndexer = (*SkillFiles)(nil)

type SkillFiles struct {
	fs *FileSystem

	idx map[agent.SkillID]agent.Skill

	mu sync.RWMutex
}

func NewSkillFiles(fs *FileSystem) *SkillFiles {
	sf := &SkillFiles{
		fs:  fs,
		idx: map[agent.SkillID]agent.Skill{},
	}

	sf.loadSkills()

	return sf
}

func (f *SkillFiles) GetIndex() map[agent.SkillID]agent.Skill {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.idx
}

func (f *SkillFiles) loadSkills() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.fs.ReadDir(skillsFolder)
	if err != nil {
		return err
	}

	for _, skillFolder := range entries {
		data, err := f.fs.ReadFile(fmt.Sprintf("%s/%s/%s", skillsFolder, skillFolder, skillFile))
		if err != nil {
			return err
		}

		skill, err := parseSkillFile(data)
		if err != nil {
			slog.Error("loading skills", "error", err, "skill", skillFolder)
			continue
		}

		if !validateSkillName(skillFolder, skill.ID()) {
			slog.Error("skillfolder must named as skill", "skill folder", skillFolder, "skill name", skill.ID())
			continue
		}

		f.idx[skill.ID()] = skill
	}

	return nil
}

type skillFrontmatterDTO struct {
	ID          agent.SkillID    `yaml:"name"`
	Description string           `yaml:"description,omitempty"`
	Tools       []agent.ToolName `yaml:"allowed-tools,omitempty"`
}

func parseSkillFile(data []byte) (agent.Skill, error) {
	const delim = "---"
	s := strings.ReplaceAll(string(data), "\r\n", "\n")

	after, ok := strings.CutPrefix(s, delim+"\n")
	if !ok {
		return nil, fmt.Errorf("skill file must start with ---")
	}

	fmEnd := strings.Index(after, "\n"+delim)
	if fmEnd == -1 {
		return nil, fmt.Errorf("unclosed frontmatter")
	}

	var dto skillFrontmatterDTO
	if err := yaml.Unmarshal([]byte(after[:fmEnd]), &dto); err != nil {
		return nil, err
	}

	prompt := strings.TrimPrefix(after[fmEnd+len("\n"+delim):], "\n")

	return agent.NewSkill(dto.ID, dto.Description, dto.Tools, prompt), nil
}

func validateSkillName(skillFolder string, skillID agent.SkillID) bool {
	return skillFolder != string(skillID)
}
