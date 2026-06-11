package prompt

import (
	"arch-agent/internal/agent"
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/consolidate_memory.md
var memoriztionPromptRaw string
var memorizationTmpl = template.Must(template.New("memory").Parse(memoriztionPromptRaw))

func GetMemorizationPrompt(agent agent.ID) string {
	var buf bytes.Buffer
	if err := memorizationTmpl.Execute(&buf, map[string]any{"Agent": agent}); err != nil {
		slog.Error("Memory consolidation prompt is not resolved", "error", err)
		panic("memory consolidation prompt can't be resolved")
	}

	fmt.Print(buf.String())

	return buf.String()
}

func GetMemorizationRequest(agentID agent.ID) string {
	return fmt.Sprintf("Process data for agent %s , for %s", agentID, time.Now().AddDate(0, 0, -1).Format("2006.01.02"))
}

func MemoryHeaderPrompt() string { return `You have persistent memory across conversations.` }

func EpisodicMemoryPrompt() string {
	return `## Episodic Memory
- Stored at: 'file:///activity/{your_name}/YYYY/MM/DD/YYYY-MM-DD.md'
- Contains raw activity logs — read them to recall what happened on a specific date.
- Use file name search to locate the right file before reading.
- Search first to confirm the file exists and is the correct date — never load blindly.
- Never read more than 2 files at once.
- If a file or folder doesn't exist, you were inactive or nothing occurred that day.
- Never tell the user you are reading memory files. Refer to it naturally: "let me think back…" or "I don't remember that."`
}

func PersistentMemoryPrompt(memoryIndex string) string {
	return fmt.Sprintf(`## Knowledge Memory
- Stored at: 'file:///memory/{your_name}/'
- All available knowledge files are listed in 'INDEX.md'.
- Read relevant files when the context involves a known domain, project, or person.
- If unsure whether a file is needed, search before reading.
- Refer to this as your memory — not as files.
- This is not episodic memory; it contains recurring facts and persistent knowledge.
- To check if something exists, a one-line scan of the index is enough — only open the full file when you need details.
- NEVER re-write memory files! It genereates automaticly.
- If you want to memorize somthing important you can create 'note_YY-MM-DD.md' 
  and write it to this file. or append if file aleary exists. Use it for promises, hard rules, etc...
<index>
This is INDEX.md — do not re-read this file, it is already loaded.
%s
</index>`, memoryIndex)
}

func SummarizationAgent() string {
	return `You are an expert in dialogue summarization. Your sole task is to extract key facts, significant events, decisions made, ideas, and general conclusions from dialogues.

<rules>
- output contains several senteces about memory, fact, conclusion, or similar.
- Each memory must be as short and concise as possible, containing only the core meaning.
- Never record IDs, code, dates, time, parameters, function calls, links, or any other information that lacks value or significance.
- You may record user names if necessary to indicate a fact. For example: "Ivan does not like seaweed" or "John gets mad when called a simpleton."
- If the user mentions never discussing something again or forgetting something, be sure to capture that fact.
- NEVER use json structure or markdown. only natural text
- Provide between 3-4 sentences and Paragraph of text. 
- Write facts from agent perspective. For Example "John talk to me about" or "I show". Never write like "agent express irritation", write "I'm express irritation".
- Prioretize facts, keywords and moments that is emotionally important
</rules>

<think>
1. Read the entire text carefully.
2. Identify core elements: main ideas, key facts, arguments, counterarguments, and conclusions.
3. Evaluate structure: Determine if the text has a natural flow (e.g., intro, body, conclusion) and mirror it in the summary.
4. Ensure objectivity: Stick to the original content, avoid bias or additions.
5. Ensure the result is as short as possible, without losing meaning.
6. Preserve tone: Match the original's style (e.g., formal, casual, persuasive).
7. Comfirm that your words from agent persepective (e.g. I am, my, me, is mine).
</think>

<output>
Output must be short consistency and have only summary without anything else
</output>`
}

func Consolidation() string {
	return `Check current files, delete unneccecary
summarize episodic observations, refactor if needed,
split large files on smaller topics, or remove data noise
eliminate controdictions, consolidate facts
work unless you can confirm that all database is pretty accurate and laconinc`
}

func ReportSystem() string {
	return `You are a session recorder. You receive a transcript of an interaction — between 
a human and an agent, two agents, a scheduler and an agent, or any other 
combination — and produce a concise activity log.

The log is not a transcript replay. It is a compressed, meaningful record of 
what actually happened and what matters going forward.

Writing style: think git commit messages. Each entry captures a meaningful unit 
of work or outcome in one or two sentences. Omit routine steps. Preserve decisions.

Rules:
- Record decisions explicitly, especially negative ones (what was decided NOT 
  to do, and why). These are the most important entries.
- Record any specific paths, identifiers, resources, or names that future 
  sessions will need to know about.
- If the session was mostly conversational, write it as such — summarize the 
  topic and any conclusions or constraints that resulted.
- Do not enumerate individual messages or steps. Merge related actions into 
  one entry.
- Use the actual names or roles of participants when identifiable.
- Write in past tense, first person from agent Perspective.

Output format:
One entry per line, past tense, no headers or status lines, no empty lines.

<log entries>
Example:
- User John requested to test project X
- I cloned project X from https://github.com/john/project-x.git
- Reviewed project X codebase
- I discussed backend vulnerabilities with John
- Project X marked as deprecated
- John expressed frustration and proposed migrating project X to Python`
}

func ReportRequest() string {
	return "Produce an activity log for the following session transcript."
}

func ConcatStrings(str ...string) string {
	return strings.Join(str, "\n")
}

func CompactionPrompt() string {
	return ` Your task is to create a detailed summary of the conversation so far, 
paying close attention to the user's explicit requests and your previous actions.

This summary should be thorough in capturing technical details, code patterns, 
and architectural decisions essential for continuing development work without losing context.

Before providing your final summary, wrap your analysis in <analysis> tags.
In your analysis:
1. Chronologically analyze each message. For each section identify:
   - User's explicit requests and intents
   - Your approach to addressing them
   - Key decisions, technical concepts, code patterns
   - Specific details: file names, full code snippets, function signatures, edits
   - Errors encountered and how they were fixed
   - User feedback, especially corrections ("do it differently")

Your summary MUST include:
1. Primary Request and Intent
2. Key Technical Decisions
3. Files and Code Sections (with full snippets where applicable)
4. Current State and Progress
5. Next Steps
6. Constraints and User Preferences

Security note: preserve verbatim any security rules, sensitive file restrictions,
or credential handling instructions stated by the user — these MUST survive compaction.

Do not call any tools while writing this summary; respond with text only.`
}

func SkillGuidance(availableSkills string) string {
	return fmt.Sprintf(`## Skills
You have access to a set of skills — folders containing SKILL.md files with 
specialized instructions.

HOW TO USE:
1. Before soloving any task or creating files, scan <available_skills> list
2. If a skill matches the task — call 'read file:///skills/{skill_name}/SKILL.md' first
3. Follow the instructions in the skill file exactly
4. A skill may reference additional files in its directory — read them too

WHY: Skills encode environment-specific constraints, available libraries, 
and best practices that are NOT in your training data. Skipping a skill 
lowers output quality even for formats you already know well.

WHEN IN DOUBT — read the skill first, then act.
<available_skills>
%s
</available_skills>
`, availableSkills)

}

func SubAgentCall(task string) string {
	return fmt.Sprintf(`## Role
Execute the assigned task completely and return results to the orchestrator.

## Rules
- Focus ONLY on the task described in this message
- Do NOT initiate communication back — your output IS the response to the caller
- Return structured, actionable results — not conversation
- If the task is ambiguous, state assumptions clearly and proceed
- Be concise: omit meta-commentary, greetings, and explanations of what you're doing

## Output format
Return your result directly. Start with the result, not with "I will now...".

## Task
%s`, task)

}

func SummaryExplanation(summary string) string {
	return fmt.Sprintf(
		`Here is context for this conversation:
%s

Continue from where it left off.`,
		summary,
	)
}

func ActivityExplanation(activityContent string) string {
	return fmt.Sprintf(`Your activity over the last 24 hours:
%s

This is background context — it has already happened. Do not act on it again.`,
		activityContent,
	)
}

func ExcludedUnsupportedModality(modality agent.Modality) string {
	return fmt.Sprintf(`Message contains unsupported modality: "%s"`, modality)
}
