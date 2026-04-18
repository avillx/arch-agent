package knowledge

import (
	"errors"
	"slices"
)

type Knowledge struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type KnowledgeIndex struct {
	Knowledges []Knowledge `json:"knowledges"`
}

func (idx *KnowledgeIndex) DeleteKnowledge(name string) {
	idx.Knowledges = slices.DeleteFunc(idx.Knowledges, func(k Knowledge) bool {
		return k.Name == name
	})
}

func (idx *KnowledgeIndex) FindKnowledge(name string) (*Knowledge, error) {
	// index of is a array index in Knowledges
	indexOf := slices.IndexFunc(idx.Knowledges, func(k Knowledge) bool {
		return k.Name == name
	})
	if indexOf == -1 {
		return nil, errors.New("Knowledge is not in index")
	}
	return &idx.Knowledges[indexOf], nil
}
func (idx *KnowledgeIndex) AddKnowledge(name, description string) {
	idx.Knowledges = append(idx.Knowledges, Knowledge{
		Name:        name,
		Description: description,
	})
}
