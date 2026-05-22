package tokenizer

import (
	"log/slog"

	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
)

type Tokenizer struct {
	tokenizer tokenizer.Codec
}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		tokenizer: codec.NewCl100kBase(),
	}
}

func (c *Tokenizer) Calc(content string) int {
	ids, _, err := c.tokenizer.Encode(content)

	if err != nil {
		slog.Error("bad tokenization")
	}
	return len(ids)

}
