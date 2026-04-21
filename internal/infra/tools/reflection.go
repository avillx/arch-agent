package tools

import (
	"arch-agent/internal/app/reflection"
	"arch-agent/internal/app/types"
	"arch-agent/internal/infra/llm"
	"fmt"
)

func BoostEmotion(e *reflection.EmotionalService) llm.Tool {
	return llm.Tool{
		ToolDefinition: types.ToolDefinition{
			Name:        "set_emotion",
			Description: "set value of emotion",
			Properties: []types.ToolProperty{
				{
					Name:        "emotion",
					Required:    true,
					Type:        types.TypeString,
					Description: "choose one emotion from enum",
					Enum:        e.Emotions(),
				},
				{
					Name:        "value",
					Required:    true,
					Type:        types.TypeNumber,
					Description: "new value of emotion. should be from 0 to 100",
				},
			},
		},
		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
			Emotion  string  `json:"emotion"`
			NewValue float32 `json:"value"`
		}) (string, error) {
			if err := e.Boost(args.Emotion, args.NewValue); err != nil {
				// TODO Check error processing in recivier. about agent mistakes
				return err.Error(), nil
			}
			return fmt.Sprintf("emotion %s value is %f", args.Emotion, args.NewValue), nil
		}),
	}
}
