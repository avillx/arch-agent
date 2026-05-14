package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/task"
	"arch-agent/internal/domain/types"
	"fmt"
	"strings"
	"time"
)

func NewTaskTS(s *TaskService) *InternalServer {
	return NewInternalServer(
		"self_task",
		func(agentID agent.ID) string {

			tasks, _ := s.All()

			if !(len(tasks) > 0) {
				return ""
			}

			var sb strings.Builder

			sb.WriteString("<tasks>")
			sb.WriteString("this is your actual tasks")
			for _, t := range tasks {
				if !t.Active {
					continue
				}

				sb.WriteString(fmt.Sprintf("* %s %s - %s", t.Name, t.Reglament.String(), t.Description))
			}
			sb.WriteString("</tasks>")

			return sb.String()
		},
		CreateTask(s),
	)
}

func CreateTask(s *TaskService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "new_task",
			Description: "create a task that will be sended for you after delay expires",
			Properties: []types.ToolProperty{
				{
					Name:        "name",
					Required:    true,
					Type:        types.TypeString,
					Description: "short name for task",
				},
				{
					Name:        "description",
					Required:    true,
					Type:        types.TypeString,
					Description: "one-line hook description of task",
				},
				{
					Name:        "delay",
					Required:    true,
					Type:        types.TypeNumber,
					Description: "Delay in minutes, you receve this task after delay expires",
				},
				{
					Name:        "request",
					Required:    true,
					Type:        types.TypeString,
					Description: "This is request for yourself, what is need to do, create a detailed explaination of task",
				},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			Name        string        `json:"name"`
			Description string        `json:"description"`
			Request     string        `json:"request"`
			Delay       time.Duration `json:"delay"`
		}, agentID string) (string, error) {

			taskID, err := s.New(
				task.NewTask(
					args.Name,
					args.Description,
					[]task.Recipient{
						task.Recipient(agentID),
					},
					args.Request,
					task.Every{
						D: args.Delay * time.Minute,
					},
					true,
				),
			)

			if err != nil {
				return "task is not created", err
			}

			if err := s.Start(taskID); err != nil {
				return "task was not runned", err
			}

			return "task has been created", nil
		}),
	}
}
