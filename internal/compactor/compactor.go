package compactor

const CompactionSystemPrompt = `You are an expert in dialogue summarization. Your sole task is to extract key facts, significant events, decisions made, ideas, and general conclusions from dialogues.

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

// type Reason string

// const (
// 	Summarize Reason = ""
// 	Report    Reason = ""
// 	Activity  Reason = ""
// )

// type Compactor struct {
// 	reasoner agent.Reasoner
// }

// func (c *Compactor) CompactWithReason(ctx context.Context, r Reason, messages []agent.Message) (string, error) {

// 	conver := agent.StringifyConversation(messages)
// 	request := []agent.Message{agent.NewUserMessage(conver)}

// 	res, err := c.reasoner.Reason(ctx, nil, request)
// 	if err != nil {
// 		return "", err
// 	}

// 	res.

// }

// func (s *Compactor) CompactMessages(ctx context.Context, sess agent.Session) (string, error) {
// 	// TODO: refactor
// 	messages := sess.Messages()
// 	half := len(messages) / 2
// 	conver := agent.StringifyConversation(messages[:half])
// 	request := []agent.Message{agent.NewUserMessage(conver)}

// 	result, err := s.reasoner.Chat(ctx, "summarizer", "", request, nil, nil)
// 	if err != nil {
// 		return err
// 	}
// 	summary := result[len(result)-1].Content()
// 	sess.AddSummary(summary)
// 	sess.OverwriteMessages(messages[half:])

// 	return nil
// }
