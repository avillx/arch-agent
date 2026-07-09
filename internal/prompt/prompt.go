package prompt

import (
	"arch-agent/internal/agent"
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/consolidate_memory.md
var consolidateMemoryRaw string
var consolidateMemoryTmpl = template.Must(template.New("consolidate_memory").Parse(consolidateMemoryRaw))

//go:embed templates/memorization_request.md
var memorizationRequestRaw string
var memorizationRequestTmpl = template.Must(template.New("memorization_request").Parse(memorizationRequestRaw))

//go:embed templates/memory_header.md
var memoryHeaderRaw string

//go:embed templates/episodic_memory.md
var episodicMemoryRaw string

//go:embed templates/persistent_memory.md
var persistentMemoryRaw string
var persistentMemoryTmpl = template.Must(template.New("persistent_memory").Parse(persistentMemoryRaw))

//go:embed templates/summarization_agent.md
var summarizationAgentRaw string

//go:embed templates/consolidation.md
var consolidationRaw string

//go:embed templates/report_system.md
var reportSystemRaw string

//go:embed templates/report_request.md
var reportRequestRaw string

//go:embed templates/compaction.md
var compactionRaw string

//go:embed templates/skill_guidance.md
var skillGuidanceRaw string
var skillGuidanceTmpl = template.Must(template.New("skill_guidance").Parse(skillGuidanceRaw))

//go:embed templates/subagent_call.md
var subagentCallRaw string
var subagentCallTmpl = template.Must(template.New("subagent_call").Parse(subagentCallRaw))

//go:embed templates/summary_explanation.md
var summaryExplanationRaw string
var summaryExplanationTmpl = template.Must(template.New("summary_explanation").Parse(summaryExplanationRaw))

//go:embed templates/activity_explanation.md
var activityExplanationRaw string
var activityExplanationTmpl = template.Must(template.New("activity_explanation").Parse(activityExplanationRaw))

//go:embed templates/excluded_unsupported_modality.md
var excludedModalityRaw string
var excludedModalityTmpl = template.Must(template.New("excluded_unsupported_modality").Parse(excludedModalityRaw))

//go:embed templates/autonomous_request.md
var autonomousRequestRaw string
var autonomousRequestTmpl = template.Must(template.New("autonomous_request").Parse(autonomousRequestRaw))

func mustExecute(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("prompt template %q execution failed: %v", tmpl.Name(), err))
	}
	return buf.String()
}

func GetMemorizationPrompt(agentID agent.ID) string {
	return mustExecute(consolidateMemoryTmpl, map[string]any{"Agent": agentID})
}

func GetMemorizationRequest(agentID agent.ID) string {
	return mustExecute(memorizationRequestTmpl, map[string]any{
		"AgentID": agentID,
		"Date":    time.Now().AddDate(0, 0, -1).Format("2006.01.02"),
	})
}

func MemoryHeaderPrompt() string { return memoryHeaderRaw }

func EpisodicMemoryPrompt() string { return episodicMemoryRaw }

func PersistentMemoryPrompt(memoryIndex string, agentID agent.ID) string {
	return mustExecute(persistentMemoryTmpl, map[string]any{"Index": memoryIndex, "Agent": agentID})
}

func SummarizationAgent() string { return summarizationAgentRaw }

func Consolidation() string { return consolidationRaw }

func ReportSystem() string { return reportSystemRaw }

func ReportRequest() string { return reportRequestRaw }

func ConcatStrings(str ...string) string { return strings.Join(str, "\n") }

func CompactionPrompt() string { return compactionRaw }

func SkillGuidance(availableSkills string) string {
	return mustExecute(skillGuidanceTmpl, map[string]any{"Skills": availableSkills})
}

func SubAgentCall(task string) string {
	return mustExecute(subagentCallTmpl, map[string]any{"Task": task})
}

func SummaryExplanation(summary string) string {
	return mustExecute(summaryExplanationTmpl, map[string]any{"Summary": summary})
}

func ActivityExplanation(activityContent string) string {
	return mustExecute(activityExplanationTmpl, map[string]any{"Content": activityContent})
}

func ExcludedUnsupportedModality(modality agent.Modality) string {
	return mustExecute(excludedModalityTmpl, map[string]any{"Modality": modality})
}

func GetAutonomusRequest(request string) string {
	return mustExecute(autonomousRequestTmpl, map[string]any{"Request": request})
}
