package ports

import (
	"context"
	"encoding/json"
)

type Recivier interface {
	Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error)
}

//
// memory repo
//
// summarizer
//
// reasoner
//
// reflector
