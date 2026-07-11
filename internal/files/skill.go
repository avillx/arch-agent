package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/runtime"
	"arch-agent/internal/types"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const skillsFolder = "/skills"
const skillFile = "SKILL.md"

var _ runtime.SkillRepo = (*SkillFiles)(nil)

type SkillFiles struct {
	fs *FileSystem

	mu sync.RWMutex
}

func NewSkillFiles(fs *FileSystem) *SkillFiles {
	sf := &SkillFiles{
		fs: fs,
	}

	return sf
}

func (f *SkillFiles) GetSkills(agentID agent.ID) ([]agent.SkillFrontmatter, error) {
	skills := []agent.SkillFrontmatter{}

	// private skills
	privateSkillsPath := path.Join(string(filepath.Separator), string(agentID), skillsFolder)
	privateSkills, err := f.loadSkills(privateSkillsPath)
	if err != nil {
		return nil, err
	}

	if privateSkills != nil {
		skills = append(skills, privateSkills...)
	}

	// shared skills
	sharedSkills, err := f.loadSkills(skillsFolder)
	if err != nil {
		return nil, err
	}

	if sharedSkills != nil {
		skills = append(skills, sharedSkills...)
	}

	return skills, nil
}

func (f *SkillFiles) loadSkills(p string) ([]agent.SkillFrontmatter, error) {

	entries, err := f.fs.ReadDir(p)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return nil, nil
		}
		return nil, err
	}

	skills := []agent.SkillFrontmatter{}
	for _, entry := range entries {
		data, err := f.fs.ReadFile(path.Join(p, entry.Name(), skillFile))
		if err != nil {
			return nil, err
		}

		skill, err := resolveSkillFrontmatter(path.Join(p, entry.Name(), skillFile), data)
		if err != nil {
			slog.Error("loading skills", "error", err, "skill folder", entry)
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

type skillFrontmatterDTO struct {
	ID          string           `yaml:"name"`
	Description string           `yaml:"description,omitempty"`
	Tools       []agent.ToolName `yaml:"allowed-tools,omitempty"`
}

func resolveSkillFrontmatter(p string, data []byte) (agent.SkillFrontmatter, error) {
	const delim = "---"
	s := strings.ReplaceAll(string(data), "\r\n", "\n")

	after, ok := strings.CutPrefix(s, delim+"\n")
	if !ok {
		return agent.SkillFrontmatter{}, fmt.Errorf("skill file must start with ---")
	}

	fmEnd := strings.Index(after, "\n"+delim)
	if fmEnd == -1 {
		return agent.SkillFrontmatter{}, fmt.Errorf("unclosed frontmatter")
	}

	var dto skillFrontmatterDTO
	if err := yaml.Unmarshal([]byte(after[:fmEnd]), &dto); err != nil {
		return agent.SkillFrontmatter{}, err
	}

	return agent.SkillFrontmatter{
		ID:          dto.ID,
		Description: dto.Description,
		StoreHint:   p,
	}, nil
}
