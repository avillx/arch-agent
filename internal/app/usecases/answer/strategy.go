package answer

import "math/rand/v2"

func randomStrategy() string {
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
	}
	n := rand.IntN(len(strateges) - 0)
	return strateges[n]
}
