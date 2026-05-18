package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"context"
	"fmt"
	"log/slog"
	"strings"
)

func NewA2ATS(s *A2AService) *InternalServer {
	return NewInternalServer(
		"a2a",
		func(agentID agent.ID) string {

			contacts, err := s.AgentContacts(agentID)
			if err != nil {
				slog.Error("A2A tool guidance, bad ", "error", err)
			}

			var sb strings.Builder

			sb.WriteString("<Agents>")
			sb.WriteString("When addressing another agent, be precise and structured:\n")
			sb.WriteString("State your goal, provide necessary context, specify expected output format, and avoid ambiguity.\n")
			sb.WriteString("Available agents:\n")
			for _, c := range contacts {
				sb.WriteString(fmt.Sprintf("* %s - %s\n", c.ID, c.CallGuide))
			}
			sb.WriteString("</Agents>")

			return sb.String()
		},
		CallAgentTool(s),
	)
}

func CallAgentTool(s *A2AService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "call_agent",
			Description: "send request to another agent",
			Properties: []types.ToolProperty{
				{
					Name:        "name",
					Required:    true,
					Type:        types.TypeString,
					Description: "name of agent",
				},
				{
					Name:        "request",
					Required:    true,
					Type:        types.TypeString,
					Description: "Is a structured message from one agent to another containing a clear goal, required context, and expected output format.",
				},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			Name    string `json:"name"`
			Request string `json:"request"`
		}, agentID string) (string, error) {
			contacts, err := s.AgentContacts(agent.ID(agentID))
			if err != nil {
				return "", err
			}

			for _, contact := range contacts {
				if contact.ID == agent.ID(args.Name) {
					return s.Call(
						context.Background(),
						agent.ID(agentID),
						agent.ID(contact.ID),
						args.Request,
					)
				}
			}

			return "", fmt.Errorf("agent %s not found", args.Name)
		}),
	}
}
