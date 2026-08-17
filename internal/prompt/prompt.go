package prompt

import (
	"arch-agent/internal/agent"
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
	"time"
)

// raw prompts

//go:embed templates/summarization_agent.md
var summarizationAgentRaw string

func SummarizationAgent() string { return summarizationAgentRaw }

//go:embed templates/report_system.md
var reportSystemRaw string

func ReportSystem() string { return reportSystemRaw }

//go:embed templates/report_request.md
var reportRequestRaw string

func ReportRequest() string { return reportRequestRaw }

//go:embed templates/compaction.md
var compactionRaw string

func Compaction() string { return compactionRaw }

//go:embed templates/empty_answer_caution.md
var emptyAnswerCautionRaw string

func EmptyAnswerCaution() string { return emptyAnswerCautionRaw }

//go:embed templates/tool_usage_guidance.md
var toolUsageGuidanceRaw string

func ToolUsageGuide() string { return toolUsageGuidanceRaw }

//go:embed templates/default_agent.md
var defaultAgentRaw string

func DefaultAgent() string { return defaultAgentRaw }

//go:embed templates/sub_agent_callstack_overflow.md
var subAgentCallStackOverflowRaw string

func SubAgentCallStackOverflowCaution() string { return subAgentCallStackOverflowRaw }

//go:embed templates/subagent_call.md
var subagentCallRaw string

func SubAgentGuidance() string { return subagentCallRaw }

//go:embed templates/consolidation_fs_instruction.md
var consolidationFSinstructionRaw string
var consolidationFSinstructionTempl = template.Must(template.New("consolidate_fs_inst").Parse(consolidationFSinstructionRaw))

func ConsolidationFSInstruction(cwd string, agentID agent.ID) string {

	vars := map[string]any{
		"Agent": agentID,
		"Cwd":   cwd,
	}

	return mustExecute(consolidateMemoryTmpl, vars)
}

//go:embed templates/consolidator.md
var consolidateMemoryRaw string
var consolidateMemoryTmpl = template.Must(template.New("consolidate_memory").Parse(consolidateMemoryRaw))

func Memorization(agentID agent.ID) string {

	vars := map[string]any{
		"Agent": agentID,
	}

	return mustExecute(consolidateMemoryTmpl, vars)
}

//go:embed templates/memorization_request.md
var memorizationRequestRaw string
var memorizationRequestTmpl = template.Must(template.New("memorization_request").Parse(memorizationRequestRaw))

func MemorizationRequest(agentID agent.ID) string {

	vars := map[string]any{
		"AgentID": agentID,
		"Date":    time.Now().AddDate(0, 0, -1).Format("2006.01.02"),
	}

	return mustExecute(memorizationRequestTmpl, vars)
}

//go:embed templates/memory_explanation.md
var persistentMemoryRaw string
var persistentMemoryTmpl = template.Must(template.New("persistent_memory").Parse(persistentMemoryRaw))

func PersistentMemory(agentID agent.ID, memoryIndex string, activity string) string {

	vars := map[string]any{
		"Index":    memoryIndex,
		"Agent":    agentID,
		"Activity": activity,
	}

	return mustExecute(persistentMemoryTmpl, vars)
}

//go:embed templates/skill_guidance.md
var skillGuidanceRaw string
var skillGuidanceTmpl = template.Must(template.New("skill_guidance").Parse(skillGuidanceRaw))

func SkillGuidance(availableSkills string) string {

	vars := map[string]any{
		"Skills": availableSkills,
	}

	return mustExecute(skillGuidanceTmpl, vars)
}

//go:embed templates/summary_explanation.md
var summaryExplanationRaw string
var summaryExplanationTmpl = template.Must(template.New("summary_explanation").Parse(summaryExplanationRaw))

func SummaryExplanation(summary string) string {

	vars := map[string]any{
		"Summary": summary,
	}

	return mustExecute(summaryExplanationTmpl, vars)
}

//go:embed templates/excluded_unsupported_modality.md
var excludedModalityRaw string
var excludedModalityTmpl = template.Must(template.New("excluded_unsupported_modality").Parse(excludedModalityRaw))

func ExcludedUnsupportedModality(modality agent.Modality) string {

	vars := map[string]any{
		"Modality": modality,
	}

	return mustExecute(excludedModalityTmpl, vars)
}

//go:embed templates/autonomous_request.md
var autonomousRequestRaw string
var autonomousRequestTmpl = template.Must(template.New("autonomous_request").Parse(autonomousRequestRaw))

func GetAutonomusRequest(request string) string {

	vars := map[string]any{
		"Request": request,
	}

	return mustExecute(autonomousRequestTmpl, vars)
}

//go:embed templates/memory_file_system_instruction.md
var memoryFilesystemInstructionRaw string
var memoryFilesystemInstructionTmpl = template.Must(template.New("mem_fs_inst").Parse(memoryFilesystemInstructionRaw))

func memoryFileSystemInstruction(agentID agent.ID) string {

	vars := map[string]any{
		"Agent": agentID,
	}

	return mustExecute(memoryFilesystemInstructionTmpl, vars)
}

//go:embed templates/file_system_instruction.md
var filesystemInstructionRaw string
var filesystemInstructionTmpl = template.Must(template.New("file_system_instruction").Parse(filesystemInstructionRaw))

func FileSystemInstruction(cwd string, agentID agent.ID, addMemory bool) string {

	additional := ""

	if addMemory {
		additional = memoryFileSystemInstruction(agentID)
	}

	components := map[string]any{
		"Cwd":        cwd,
		"Agent":      agentID,
		"Additional": additional,
	}

	return mustExecute(filesystemInstructionTmpl, components)
}

//go:embed templates/undone_todos_caution.md
var undoneTodoCautionRaw string
var undoneTodoCautionTmpl = template.Must(template.New("undone_todos_caution").Parse(undoneTodoCautionRaw))

func UndoneTodosCaution(todoList string) string {

	vars := map[string]any{
		"Todos": todoList,
	}

	return mustExecute(undoneTodoCautionTmpl, vars)
}

// helpers

func mustExecute(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("prompt template %q execution failed: %v", tmpl.Name(), err))
	}
	return buf.String()
}
