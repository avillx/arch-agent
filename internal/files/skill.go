package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/types"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"sync"
)

const skillsFolder = "skills"
const skillFile = "SKILL.md"

var _ chat.SkillsRepo = (*SkillFiles)(nil)

type SkillFiles struct {
	storage FileStorage
	logger  *slog.Logger

	mu sync.RWMutex
}

func NewSkillFiles(
	storage FileStorage,
	logger *slog.Logger,
) *SkillFiles {
	sf := &SkillFiles{
		storage: storage,
		logger:  logger.WithGroup("skill_files"),
	}

	return sf
}

func (f *SkillFiles) Skills(agentID agent.ID) (map[string]string, error) {
	skillsIndex := map[string]string{}

	skills, err := f.loadSkills(skillsFolder)
	if err != nil {
		return nil, err
	}

	if skills != nil {
		maps.Copy(skillsIndex, skills)
	}

	return skillsIndex, nil
}

func (f *SkillFiles) loadSkills(p string) (map[string]string, error) {

	type skillFrontmatterDTO struct {
		ID          string `yaml:"name" toml:"name"`
		Description string `yaml:"description,omitempty" toml:"description,omitempty"`
	}

	skillIndex := map[string]string{}

	// missing folder means no skills
	if _, err := fs.Stat(f.storage.FS(), p); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return skillIndex, nil
		}
		return nil, err
	}

	walkDirFunc := func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			f.logger.Error("walking dir", "path", p, "error", err)
			return nil
		}

		// skip when entry is empty
		if d == nil {
			return nil
		}

		// skip dirs
		if d.IsDir() {
			return nil
		}

		// skip non skill files
		if d.Name() != skillFile {
			return nil
		}

		// read file
		data, err := f.storage.ReadFile(p)
		if err != nil {
			return err
		}

		// extract frontmatter
		dto, err := resolveFrontmatter[skillFrontmatterDTO](data)
		if err != nil {
			f.logger.Error("frontmatter parsing", "path", p, "error", err)
			return nil
		}

		// add to index
		skillIndex[p] = dto.Description

		return nil
	}

	if err := fs.WalkDir(f.storage.FS(), p, walkDirFunc); err != nil {
		return nil, err
	}

	return skillIndex, nil
}
