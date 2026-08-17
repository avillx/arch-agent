package files

import (
	"arch-agent/internal/agent"
	"arch-agent/internal/chat"
	"io/fs"
	"log/slog"
	"maps"
	"path"
	"path/filepath"
	"sync"
)

const skillsFolder = "/skills"
const skillFile = "SKILL.md"

var _ chat.SkillsRepo = (*SkillFiles)(nil)

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

func (f *SkillFiles) Skills(agentID agent.ID) (map[string]string, error) {
	skillsIndex := map[string]string{}

	// private skills
	privateSkillsPath := path.Join(string(filepath.Separator), string(agentID), skillsFolder)
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

	walkDirFunc := func(currentPath string, d fs.DirEntry, err error) error {

		// skip dirs
		if d.IsDir() {
			return nil
		}

		// skip non skill files
		if d.Name() != skillFile {
			return nil
		}

		// read file
		data, err := f.fs.ReadFile(currentPath)
		if err != nil {
			return err
		}

		// extract frontmatter
		dto, err := resolveFrontmatter[skillFrontmatterDTO](data)
		if err != nil {
			slog.Error("loading skills", "error", err, "path", currentPath)
			return nil
		}

		// add to index
		skillIndex[currentPath] = dto.Description

		return nil
	}

	if err := f.fs.WalkDir(p, walkDirFunc); err != nil {
		return nil, err
	}

	return skillIndex, nil
}
