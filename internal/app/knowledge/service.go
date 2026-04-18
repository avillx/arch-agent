package knowledge

import (
	"fmt"
	"strings"
)

type KnowledgesRepository interface {
	Read(name string) (string, error)
	AddNew(name, content string) error
	Delete(name string) error
	Override(name, content string) error
	LoadIndex() (*KnowledgeIndex, error)
	SaveIndex(*KnowledgeIndex) error
}

type Service struct {
	knowledgesRepo KnowledgesRepository
}

func NewService(knowledgesRepo KnowledgesRepository) *Service {
	return &Service{
		knowledgesRepo: knowledgesRepo,
	}
}

func (s *Service) KnowledgesList() (string, error) {
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, f := range idx.Knowledges {
		sb.WriteString(fmt.Sprintf("* %s - %s\n", f.Name, f.Description))
	}
	return sb.String(), nil
}

func (s *Service) Read(name string) (string, error) {
	return s.knowledgesRepo.Read(name)
}

func (s *Service) CreateKnowledge(name, description, content string) error {
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	if err := s.knowledgesRepo.AddNew(name, content); err != nil {
		return err
	}

	idx.AddKnowledge(name, description)
	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *Service) EditDescription(name, newDescription string) error {
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	knowledge, err := idx.FindKnowledge(name)
	if err != nil {
		return err
	}

	knowledge.Description = newDescription

	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *Service) EditName(oldName, newName string) error {
	// content update
	knowledgeContent, err := s.Read(oldName)
	if err != nil {
		return err
	}

	if err := s.knowledgesRepo.AddNew(newName, knowledgeContent); err != nil {
		return err
	}

	if err := s.knowledgesRepo.Delete(oldName); err != nil {
		return err
	}

	// index update
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	knowledge, err := idx.FindKnowledge(oldName)
	if err != nil {
		return err
	}

	idx.AddKnowledge(newName, knowledge.Description)
	idx.DeleteKnowledge(oldName)
	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *Service) EditKnowledge(name string, newContent string) error { // tool substring but service is override full file
	return s.knowledgesRepo.Override(name, newContent)
}

func (s *Service) Delete(name string) error {
	if err := s.knowledgesRepo.Delete(name); err != nil {
		return err
	}

	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}
	idx.DeleteKnowledge(name)
	return s.knowledgesRepo.SaveIndex(idx)
}
