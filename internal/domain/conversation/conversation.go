package conversation

import (
	"slices"
)

type Message interface {
	Role() Role
	Content() string
}

const TokenLimit = 20000

type Tokenizer interface {
	Calc(string) int
}

type Conversation struct {
	tokensCount int
	tokenizer   Tokenizer
	messages    []Message
	newMessages []Message
}

func NewConversation(t Tokenizer, messages []Message) *Conversation {
	return &Conversation{
		tokenizer: t,
		messages:  messages,
	}
}

func (c *Conversation) AddToolCallResults(results []ToolCallResult) {
	for _, r := range results {
		c.addNewMessage(NewToolResultMessage(r.ID, r.Result))
	}
}

func (c *Conversation) AddToolResultMessage(id string, content string) {
	c.addNewMessage(NewToolResultMessage(id, content))
}

func (c *Conversation) AddUserMessage(content string) {
	c.addNewMessage(NewUserMessage(content))
}

func (c *Conversation) AddAgentMessage(content string, tc []*ToolCall) {
	c.addNewMessage(NewAgentMessage(content, tc))
}

func (c *Conversation) addNewMessage(m Message) {
	c.calcTokens(m.Content())
	c.newMessages = append(c.newMessages, m)
}

func (c *Conversation) calcTokens(content string) {
	c.tokensCount += c.tokenizer.Calc(content)
}

func (c *Conversation) Messages() []Message {
	return slices.Concat(c.messages, c.newMessages)
}

func (c *Conversation) NewMessages() []Message {
	return c.newMessages
}

func (c *Conversation) IsOverflow() bool {
	return c.tokensCount >= TokenLimit
}
