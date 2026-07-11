package agent

type ID string

type Repo interface {
	All() ([]Agent, error)
	Get(ID) (Agent, error)
	Save(Agent) error
	// Delete(ID) error
}

var _ Agent = (*agent)(nil)

type Agent interface {
	ID() ID
	Description() string
	SystemPrompt() string
	Model() ModelID
	ToolServers() []string
	Tools() []ToolName
	HasMemory() bool
}

type agent struct {
	id           ID
	description  string
	systemPrompt string
	model        ModelID
	tools        []ToolName
	toolServers  []string
	hasMemory    bool
}

func NewAgent(
	id ID,
	description string,
	systemPrompt string,
	model ModelID,
	tools []ToolName,
	toolServers []string,
	hasMemory bool,
) *agent {
	return &agent{
		id:           id,
		description:  description,
		systemPrompt: systemPrompt,
		model:        model,
		tools:        tools,
		toolServers:  toolServers,
		hasMemory:    hasMemory,
	}
}

func (a *agent) ID() ID                { return a.id }
func (a *agent) Description() string   { return a.description }
func (a *agent) SystemPrompt() string  { return a.systemPrompt }
func (a *agent) Model() ModelID        { return a.model }
func (a *agent) Tools() []ToolName     { return a.tools }
func (a *agent) ToolServers() []string { return a.toolServers }
func (a *agent) HasMemory() bool       { return a.hasMemory }
