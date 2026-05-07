package service

import "arch-agent/internal/domain/knowledge"

type Tokenizer interface {
	Calc(string) int
}

type KnowledgesRepo interface {
	Read(name string) (string, error)
	AddNew(name, content string) error
	Delete(name string) error
	Override(name, content string) error
	LoadIndex() (*knowledge.Index, error)
	SaveIndex(*knowledge.Index) error
}

type KnowldegeService struct {
	knowledgesRepo KnowledgesRepo
	tokenizer      Tokenizer
}

func NewService(knowledgesRepo KnowledgesRepo, tokenizer Tokenizer) *KnowldegeService {
	return &KnowldegeService{
		knowledgesRepo: knowledgesRepo,
		tokenizer:      tokenizer,
	}
}

func (s *KnowldegeService) KnowledgesList(sizeGrade bool) (string, error) {
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return "", err
	}

	return idx.List(), nil
}

func (s *KnowldegeService) Read(name string) (string, error) {
	return s.knowledgesRepo.Read(name)
}

func (s *KnowldegeService) CreateKnowledge(name, description, content string) error {
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	if err := s.knowledgesRepo.AddNew(name, content); err != nil {
		return err
	}

	idx.AddRecord(knowledge.NewRecord(name, description))
	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *KnowldegeService) EditDescription(name, newDescription string) error {
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	knowledge, err := idx.FindRecord(name)
	if err != nil {
		return err
	}

	knowledge.Description = newDescription

	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *KnowldegeService) EditName(oldName, newName string) error {

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

	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	record, err := idx.FindRecord(oldName)
	if err != nil {
		return err
	}

	idx.AddRecord(&knowledge.Record{
		Name:        newName,
		Description: record.Description,
		Size:        record.Size,
	})

	idx.DeleteRecord(oldName)

	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *KnowldegeService) EditKnowledge(name string, newContent string) error { // tool substring but service is override full file
	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}

	if err := s.knowledgesRepo.Override(name, newContent); err != nil {
		return err
	}

	knowledge, err := idx.FindRecord(name)
	if err != nil {
		return err
	}

	knowledge.Size = s.tokenizer.Calc(newContent)

	return s.knowledgesRepo.SaveIndex(idx)
}

func (s *KnowldegeService) Delete(name string) error {
	if err := s.knowledgesRepo.Delete(name); err != nil {
		return err
	}

	idx, err := s.knowledgesRepo.LoadIndex()
	if err != nil {
		return err
	}
	idx.DeleteRecord(name)
	return s.knowledgesRepo.SaveIndex(idx)
}
