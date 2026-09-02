package files

import (
	"arch-agent/internal/memory"
	"bytes"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

const memoryConfig = "memory.toml"
const memoryConfigDoc string = `# Memory config

# Agent for **consolidation** memory is invoked once every 24h;
# it analyzes activity logs and maintains consistency of memory files.
# [consolidation]

# Enabled - true, Disabled - false
# enabled = true

# Model for consolidation, must support tool calls.
# Choose a model capable of handling this agentic task.
# model = ""

# Additional instruction for the consolidation request
# instruction="""
# Save all file paths and write user comments on the code in a separate file.
# """

# LLM called periodically to summarize a session segment and log it to activity.
# [activity]

# Enabled - true, Disabled - false
# enabled = true

# Model for logging; tool support is not required, a simple model is fine.
# model = ""

# Interval in seconds for flushing activity to logs. 120-600s is fine.
# interval = 600

# All available models are described in 'models.toml'.
# If a model is not listed in 'models.toml', it is not allowed.

# Do not touch this comment!
# After edit, ensure file consistency and comment integrity`

type MemoryConfigFile struct {
	fs *FileSystem

	mu sync.Mutex
}

func NewMemoryConfigFile(fs *FileSystem) (*MemoryConfigFile, error) {

	if err := ensureFilePlaceholder(fs, memoryConfig, []byte(memoryConfigDoc)); err != nil {
		return nil, err
	}

	return &MemoryConfigFile{
		fs: fs,
	}, nil
}

func (r *MemoryConfigFile) Save(new MemoryConfigDTO) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.fs.ReadFile(memoryConfig)
	if err != nil {
		return err
	}

	var dto MemoryConfigDTO
	if err := toml.Unmarshal(data, &dto); err != nil {
		return err
	}

	if new.Consolidator != nil {
		dto.Consolidator = new.Consolidator
	}

	if new.Activity != nil {
		dto.Activity = new.Activity
	}

	data, err = toml.Marshal(dto)
	if err != nil {
		return err
	}

	// with doc
	data = bytes.Join(
		[][]byte{[]byte(memoryConfigDoc), data},
		[]byte("\n\n"),
	)

	return r.fs.WriteToFile(memoryConfig, data)
}

func (r *MemoryConfigFile) Load() (MemoryConfigDTO, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.fs.ReadFile(memoryConfig)
	if err != nil {
		return MemoryConfigDTO{}, err
	}

	var dto *MemoryConfigDTO
	if err := toml.Unmarshal(data, &dto); err != nil {
		return MemoryConfigDTO{}, err
	}

	return *dto, nil
}

// activity memory repo
var _ memory.ActivityConfigRepo = (*ActivityRepo)(nil)

type ActivityRepo struct {
	*MemoryConfigFile
}

func NewActivityRepo(c *MemoryConfigFile) *ActivityRepo {
	return &ActivityRepo{
		MemoryConfigFile: c,
	}
}

func (r *ActivityRepo) Save(ac memory.ActivityConfig) error {
	return r.MemoryConfigFile.Save(MemoryConfigDTO{
		Activity: &ac,
	})
}

func (r *ActivityRepo) Load() (memory.ActivityConfig, error) {
	cfg, err := r.MemoryConfigFile.Load()
	if err != nil {
		return memory.ActivityConfig{}, err
	}

	return *cfg.Activity, nil
}

// Persistent memory repo
var _ memory.MemoryRepo = (*ConsolidatorRepo)(nil)

type ConsolidatorRepo struct {
	*MemoryConfigFile
}

func NewConsolidatorRepo(c *MemoryConfigFile) *ConsolidatorRepo {
	return &ConsolidatorRepo{
		MemoryConfigFile: c,
	}
}

func (r *ConsolidatorRepo) Save(c memory.ConsolidatorConfig) error {
	return r.MemoryConfigFile.Save(MemoryConfigDTO{
		Consolidator: &c,
	})
}

func (r *ConsolidatorRepo) Load() (memory.ConsolidatorConfig, error) {
	cfg, err := r.MemoryConfigFile.Load()
	if err != nil {
		return memory.ConsolidatorConfig{}, err
	}

	return *cfg.Consolidator, nil
}

type MemoryConfigDTO struct {
	Consolidator *memory.ConsolidatorConfig `toml:"consolidation,omitempty"`
	Activity     *memory.ActivityConfig     `toml:"activity,omitempty"`
}
