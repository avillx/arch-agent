package knowledgeadapter

// // TODO rename filestorage to storage

// import (
// 	"arch-agent/internal/app/types"
// 	"arch-agent/internal/infra/llm"
// )

// // TODO: Tool wrappers must be in dream service because is a app logic
// func KnowledgeExplorerTools(e *KnowledgeExplorer) []llm.Tool {
// 	return []llm.Tool{
// 		ReadKnowledgeTool(e),
// 		CreateKnowledgeTool(e),
// 		EditKnowledgeTool(e),
// 		AppendKnowledgeTool(e),
// 		DeleteKnowledgeTool(e),
// 		EditKnowledgeIndexTool(e),
// 	}
// }

// func ReadKnowledgeTool(e *KnowledgeExplorer) llm.Tool {
// 	return llm.Tool{
// 		ToolDefinition: types.ToolDefinition{
// 			Name:        "read",
// 			Description: "Read the full content of a knowledge file by its filename.",
// 			Properties: []types.ToolProperty{
// 				{
// 					Name:        "filename",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Name of the knowledge file to read (e.g. some_topic.md)",
// 				},
// 			},
// 		},
// 		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
// 			FileName string `json:"filename"`
// 		}) (string, error) {
// 			return e.Read(args.FileName)
// 		}),
// 	}
// }

// func CreateKnowledgeTool(e *KnowledgeExplorer) llm.Tool {
// 	return llm.Tool{
// 		ToolDefinition: types.ToolDefinition{
// 			Name:        "create",
// 			Description: "Create a new knowledge file with the given content. Fails if the file already exists.",
// 			Properties: []types.ToolProperty{
// 				{
// 					Name:        "filename",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Name of the new knowledge file (e.g. some_topic.md)",
// 				},
// 				{
// 					Name:        "description",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "One-line hook description of knowledge",
// 				},
// 				{
// 					Name:        "content",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Initial content to write into the new file",
// 				},
// 			},
// 		},
// 		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
// 			FileName    string `json:"filename"`
// 			Description string `json:"description"`
// 			Content     string `json:"content"`
// 		}) (string, error) {
// 			return e.Create(args.FileName, args.Description, args.Content)
// 		}),
// 	}
// }

// func EditKnowledgeTool(e *KnowledgeExplorer) llm.Tool {
// 	return llm.Tool{
// 		ToolDefinition: types.ToolDefinition{
// 			Name:        "edit",
// 			Description: "Replace an exact piece of text in a knowledge file. Use this to update existing content.",
// 			Properties: []types.ToolProperty{
// 				{
// 					Name:        "filename",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Name of the knowledge file to edit (e.g. some_topic.md)",
// 				},
// 				{
// 					Name:        "old",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Exact substring to find and replace.",
// 				},
// 				{
// 					Name:        "new",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "New string that replaces the old one",
// 				},
// 			},
// 		},
// 		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
// 			FileName string `json:"filename"`
// 			Old      string `json:"old"`
// 			New      string `json:"new"`
// 		}) (string, error) {
// 			return e.Edit(args.FileName, args.Old, args.New)
// 		}),
// 	}
// }

// func AppendKnowledgeTool(e *KnowledgeExplorer) llm.Tool {
// 	return llm.Tool{
// 		ToolDefinition: types.ToolDefinition{
// 			Name:        "append",
// 			Description: "Append new content to the end of an existing knowledge file.",
// 			Properties: []types.ToolProperty{
// 				{
// 					Name:        "filename",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Name of the knowledge file to append to (e.g. some_topic.md)",
// 				},
// 				{
// 					Name:        "content",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Content to append at the end of the file",
// 				},
// 			},
// 		},
// 		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
// 			FileName string `json:"filename"`
// 			Content  string `json:"content"`
// 		}) (string, error) {
// 			return e.Append(args.FileName, args.Content)
// 		}),
// 	}
// }

// func DeleteKnowledgeTool(e *KnowledgeExplorer) llm.Tool {
// 	return llm.Tool{
// 		ToolDefinition: types.ToolDefinition{
// 			Name:        "delete",
// 			Description: "Permanently delete a knowledge file.",
// 			Properties: []types.ToolProperty{
// 				{
// 					Name:        "filename",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "Name of the knowledge file to delete (e.g. some_topic.md)",
// 				},
// 			},
// 		},
// 		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
// 			FileName string `json:"filename"`
// 		}) (string, error) {
// 			return e.Delete(args.FileName)
// 		}),
// 	}
// }

// func EditKnowledgeIndexTool(e *KnowledgeExplorer) llm.Tool {
// 	return llm.Tool{
// 		ToolDefinition: types.ToolDefinition{
// 			Name:        "edit_description",
// 			Description: "Replace an one line hook description of knowledge file",
// 			Properties: []types.ToolProperty{
// 				{
// 					Name:        "filename",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "filename of knowledge index that description should be replaced",
// 				},
// 				{
// 					Name:        "description",
// 					Required:    true,
// 					Type:        types.TypeString,
// 					Description: "New description that replaces the old one",
// 				},
// 			},
// 		},
// 		CallRsolver: llm.WrapArgumentedCallResolver(func(args struct {
// 			FileName    string `json:"filename"`
// 			Description string `json:"description"`
// 		}) (string, error) {
// 			return e.EditDescription(args.FileName, args.Description)
// 		}),
// 	}
// }
