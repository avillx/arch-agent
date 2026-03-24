package conversation

import (
	"arch-agent/internal/domain/shared"
	"slices"
)

const TokenLimit = 20000

type Message interface {
	Role() shared.Role
	Content() string
}

type Tokenizer interface {
	Calc(string) int
}

type Conversation struct {
	tokensCount int
	tokenizer   Tokenizer
	messages    []Message
	newMessages []Message
}

func (c *Conversation) AddMessage(m Message) {
	c.tokensCount += c.tokenizer.Calc(m.Content())
	c.newMessages = append(c.newMessages, m)
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
