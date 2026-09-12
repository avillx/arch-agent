package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"arch-agent/internal/types"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"path"
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

	// private skills
	// NOTE: path over filepath is required
	privateSkillsPath := path.Join(string(agentID), skillsFolder)
	privateSkills, err := f.loadSkills(privateSkillsPath)
	if err != nil {
		return nil, err
	}

	// shared skills
	sharedSkills, err := f.loadSkills(skillsFolder)
	if err != nil {
		return nil, err
	}

	// unite shared and private in one index
	if privateSkills != nil {
		maps.Copy(skillsIndex, privateSkills)
	}
	if sharedSkills != nil {
		maps.Copy(skillsIndex, sharedSkills)
	}

	return skillsIndex, nil
}

func (f *SkillFiles) loadSkills(p string) (map[string]string, error) {

	type skillFrontmatterDTO struct {
		ID          string           `yaml:"name"`
		Description string           `yaml:"description,omitempty"`
		Tools       []agent.ToolName `yaml:"allowed-tools,omitempty"`
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
