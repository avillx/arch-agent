package service

import (
	"arch-agent/internal/domain/agent"
	"arch-agent/internal/domain/types"
	"fmt"
	"log/slog"
)

func NewKnowledgeReaderTS(s *KnowldegeService) *InternalServer {
	return NewInternalServer(
		"knowledge",
		func(_ agent.ID) string {
			knowledges, err := s.KnowledgesList(false)
			if err != nil {
				slog.Error("knowedgeReaderts agent instructions", "error", err)
			}

			return ConcatStrings(
				"<knowledges>",
				"# Guide:",
				"- All information in files form knowledges is your personal expirience",
				"- When in context is introduced somthing wired with knowledge from list get it by read_knowledge",
				"- Do not use read_knowledge it if you already have all data you need to know.",
				"# List of knowledges:",
				knowledges,
				"</knowledges>",
			)

		},
		ReadKnowledge(s),
	)
}

func NewKnowledgeTS(s *KnowldegeService) *InternalServer {
	return NewInternalServer(
		"knowledge",
		func(_ agent.ID) string {
			return "- use read_knowledge for find knowledges for the context, do not use it if you already have all data you need to know." +
				"- use knowledges_list to get all available knowledge files before deciding which to read or edit." +
				"- use create_knowledge to store new information that may be useful in the future." +
				"- use edit_knowledge_content to update the body of an existing knowledge file." +
				"- if file content much differ of previus content use edit_knowledge_description" +
				"- when current name is not explain of content use edit_knowledge_name " +
				"- use delete_knowledge only when a knowledge file is explicitly outdated or no longer needed."
		},
		ReadKnowledge(s),
		KnowledgesList(s),
		CreateKnowledge(s),
		EditKnowledgeContent(s),
		EditKnowledgeDescription(s),
		EditKnowledgeName(s),
		DeleteKnowledge(s),
	)
}

func ReadKnowledge(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "read_knowledge",
			Description: "Read the full content of a knowledge file by its filename.",
			Properties: []types.ToolProperty{
				{
					Name:        "filename",
					Required:    true,
					Type:        types.TypeString,
					Description: "Name of the knowledge file to read (e.g. some_topic.md)",
				},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			FileName string `json:"filename"`
		}, agentID string) (string, error) {
			return s.Read(args.FileName)
		}),
	}
}

func KnowledgesList(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "knowledges_list",
			Description: "Get a list of all available knowledge files.",
			Properties:  []types.ToolProperty{},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct{}, sign string) (string, error) {
			knowledgesList, err := s.KnowledgesList(true)
			if err != nil {
				return "", err
			}

			return "<knowledge-files>\n" + knowledgesList + "\n</knowledge-files>", nil
		}),
	}
}

func CreateKnowledge(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "create_knowledge",
			Description: "Create a new knowledge file with a name, description and content.",
			Properties: []types.ToolProperty{
				{Name: "name", Required: true, Type: types.TypeString, Description: "Unique filename for the knowledge (e.g. topic.md)"},
				{Name: "description", Required: true, Type: types.TypeString, Description: "Short description of what this knowledge contains"},
				{Name: "content", Required: true, Type: types.TypeString, Description: "Full content of the knowledge file"},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Content     string `json:"content"`
		}, agentID string) (string, error) {

			if err := s.CreateKnowledge(args.Name, args.Description, args.Content); err != nil {
				return "", err
			}
			return fmt.Sprintf("created knowledge %s ", args.Name), nil
		}),
	}
}

func EditKnowledgeContent(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "edit_knowledge_content",
			Description: "Replace the content of an existing knowledge file.",
			Properties: []types.ToolProperty{
				{Name: "name", Required: true, Type: types.TypeString, Description: "Filename of the knowledge to update"},
				{Name: "content", Required: true, Type: types.TypeString, Description: "New full content to write into the file"},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}, agentID string) (string, error) {

			if err := s.EditKnowledge(args.Name, args.Content); err != nil {
				return "", err
			}
			return fmt.Sprintf("knowledge %s has been edited", args.Name), nil
		}),
	}
}

func EditKnowledgeDescription(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "edit_knowledge_description",
			Description: "Update the description of an existing knowledge file.",
			Properties: []types.ToolProperty{
				{Name: "name", Required: true, Type: types.TypeString, Description: "Filename of the knowledge to update"},
				{Name: "description", Required: true, Type: types.TypeString, Description: "New description"},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}, agentID string) (string, error) {
			if err := s.EditDescription(args.Name, args.Description); err != nil {
				return "", err
			}
			return fmt.Sprintf("knowledge %s description has been edited", args.Name), nil
		}),
	}
}

func EditKnowledgeName(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "edit_knowledge_name",
			Description: "Rename an existing knowledge file.",
			Properties: []types.ToolProperty{
				{Name: "old_name", Required: true, Type: types.TypeString, Description: "Current filename of the knowledge"},
				{Name: "new_name", Required: true, Type: types.TypeString, Description: "New filename for the knowledge"},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			OldName string `json:"old_name"`
			NewName string `json:"new_name"`
		}, agentID string) (string, error) {

			if err := s.EditName(args.OldName, args.NewName); err != nil {
				return "", err
			}

			return fmt.Sprintf("knowledge %s renamed to %s", args.OldName, args.NewName), nil
		}),
	}
}

func DeleteKnowledge(s *KnowldegeService) *InternalTool {
	return &InternalTool{
		ToolDefinition: types.ToolDefinition{
			Name:        "delete_knowledge",
			Description: "Delete a knowledge file by its filename.",
			Properties: []types.ToolProperty{
				{Name: "name", Required: true, Type: types.TypeString, Description: "Filename of the knowledge to delete"},
			},
		},
		CallRsolver: WrapArgumentedCallResolver(func(args struct {
			Name string `json:"name"`
		}, agentID string) (string, error) {
			if err := s.Delete(args.Name); err != nil {
				return "", err
			}

			return fmt.Sprintf("knowledge %s has been deleted", args.Name), nil
		}),
	}
}
