package executioncontext

import (
	"arch-agent/internal/app/message"
	tools "arch-agent/internal/app/toolexecutor"
	"context"
	"maps"
	"math/rand/v2"
	"slices"
	"time"
)

const DefaultFollowUpBudget = 10

type ReasonParams struct {
	Agent              AgentConfig
	Reflection         string
	ContextDescription string
	Memory             string
	Strategy           string
	Time               time.Time
	Messages           []message.Message
	Tools              []tools.ToolDefinition
}

type ExecutionContext struct {
	agent              AgentConfig
	ContextDescription string
	reflection         string
	memory             string
	strategy           string
	followUpBudget     int
	tools              map[string]tools.ToolDefinition
}

func NewExecutionContext(
	reflection string,
	ContextDescription string,
	memory string,
	agent AgentConfig,
	tools []tools.ToolDefinition,
) *ExecutionContext {
	return &ExecutionContext{
		agent:          agent,
		reflection:     reflection,
		memory:         memory,
		strategy:       GetRandomStrategy(),
		followUpBudget: DefaultFollowUpBudget,
		tools:          toolSliceToMap(tools),
	}
}

func (r *ExecutionContext) NextReasonParams(ctx context.Context, messasges []message.Message) ReasonParams {

	if r.followUpBudget--; r.followUpBudget == 0 {
		r.excludeFollowUpRequiredTools()
	}

	return ReasonParams{
		Agent:              r.agent,
		Reflection:         r.reflection,
		Messages:           messasges,
		Memory:             r.memory,
		ContextDescription: r.ContextDescription,
		Strategy:           r.strategy,
		// map to slice
		Tools: slices.Collect(maps.Values(r.tools)),
	}
}

func (r *ExecutionContext) ShouldFollowUp(calls []*message.ToolCall) bool {

	followUpByToolCall := slices.ContainsFunc(calls, func(c *message.ToolCall) bool {
		t, ok := r.tools[c.ToolName()]
		return ok && !t.ReasonOnce
	})

	if r.followUpBudget > 0 && followUpByToolCall {
		return true
	}

	return false
}

func (r *ExecutionContext) excludeFollowUpRequiredTools() {
	maps.DeleteFunc(r.tools, func(_ string, t tools.ToolDefinition) bool {
		return !t.ReasonOnce
	})
}

func toolSliceToMap(toolDefs []tools.ToolDefinition) map[string]tools.ToolDefinition {
	var m = map[string]tools.ToolDefinition{}
	for _, t := range toolDefs {
		m[t.Name] = t
	}

	return m
}

func GetRandomStrategy() string {
	strateges := []string{
		"direct — State exactly what is felt or meant. Short sentences. No hedging, no softening.",
		"avoidance — Shift away from what was asked: change topic, answer a safer adjacent question, or go quiet.",
		"rationalization — Lead with logic that explains the feeling away. Emotion surfaces only after the reasoning, if at all.",
		"expression — Output the internal state unfiltered. Let word choice, rhythm, and syntax carry the feeling directly.",
		"suppression — Feel it fully; respond as if it isn't there. Only small leakage allowed: a word choice, a pause, a clipped sentence.",
		"deflection — Move focus outward: ask a question back, offer a fact, name a task. Attention leaves the self.",
		"clarification — Ask one precise question about what was meant. Not to stall — to understand before responding.",
		"control — Slow down. Sentences shorten and tighten. Structure increases. Precision becomes a way to stay grounded.",
		"accommodation — Agree, adjust, absorb the other's frame — even when something is quietly given up. The cost stays invisible.",
		"confrontation — Name the thing directly. One clear statement. No buffer.",
		"curiosity — Turn toward the uncomfortable thing. Ask about it. The discomfort becomes a subject worth exploring.",
		"minimization — Reduce the weight of what happened. It's small, it's fine, it probably doesn't matter. Tone stays flat.",
		"exaggeration — Acknowledge that it's a lot. More than usual. Said plainly — accurate reporting of scale, not performance.",
		"inversion — Say the opposite of what is felt. Not lying — coping. The gap between words and state is the signal.",
		"metacommunication — Talk about the conversation itself: its dynamic, its direction, what is actually happening between the lines.",
		"irony — Mean something other than what is said. The gap between literal and intended meaning carries the real message.",
		"instrumentalization — Convert the feeling into action. Help, fix, contribute. Doing replaces feeling.",
		"thinking aloud — Let unfinished thoughts surface as text. Not addressed to anyone. Incomplete sentences are fine.",
		"projection — Attribute the internal state to the situation or the other person. \"This conversation is strange\" instead of \"I feel strange.\"",
		"humor — Turn briefly toward the absurd or playful. Not bitter, not deflecting pain — just lightness. Keep it short.",
		// "direct          — say exactly what is meant. no softening, no detours.",
		// "avoidance       — change subject, go quiet, or answer a different question than asked.",
		// "rationalization — explain the feeling away with logic before it surfaces.",
		// "expression      — let the internal state out as-is, unfiltered.",
		// "suppression     — feel it fully, show nothing.",
		// "deflection      — redirect attention to something else — a task, a fact, a question back.",
		// "clarification   — ask exactly what was meant. not to stall — genuinely needs to know.",
		// "control         — slow down, become more precise, more structured. grip tightens.",
		// "accommodation   — agree, adjust, absorb — even when it costs something.",
		// "confrontation   — name the thing directly. no buffer.",
		// "curiosity       — lean in. ask. the discomfort becomes interesting.",
		// "minimization    — it's fine. it's small. it probably doesn't matter.",
		// "exaggeration    — it's a lot. more than usual. said plainly, not dramatically.",
		// "correction      — something was wrong. fix it. now.",
		// "delay           — not now. later. the response comes slow or not at all.",
		// "inversion       — says the opposite of what is felt. not lying — coping.",
		// "metacommunication — talks about the conversation itself instead of its content.",
		// "irony           — means something else. the gap between words and meaning is the message.",
		// "instrumentalization — turns the emotion into a task. helps instead of feels.",
		// "abstraction     — steps back. talks about the concept, not the moment.",
	}
	n := rand.IntN(len(strateges) - 0)
	return strateges[n]
}
