package tasktools

import (
	"arch-agent/internal/task"
	"arch-agent/internal/tools"
)

type TasksToolServer struct {
	*tools.BuildInToolServer
}

func NewTasksToolServer(s *task.Service) *TasksToolServer {
	return &TasksToolServer{
		BuildInToolServer: tools.NewBuildInToolServer(
			&AddTaskTool{taskSvc: s},
			&GetTasksTool{taskSvc: s},
			&DeleteTasksTool{taskSvc: s},
			&EditTaskTool{taskSvc: s},
		),
	}
}
