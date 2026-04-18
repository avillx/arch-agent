package knowledgeadapter

import (
	"arch-agent/internal/app/knowledge"
	"arch-agent/internal/infra/storage/filesystem"
	"encoding/json"
)

const IndexFile = "knowledge.json"

type KnowledgeFiles struct {
	filesystem filesystem.FileSystem
}

func New(dir string) *KnowledgeFiles {
	return &KnowledgeFiles{
		filesystem: filesystem.New(dir),
	}
}

func (f *KnowledgeFiles) Read(name string) (string, error) {
	data, err := f.filesystem.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *KnowledgeFiles) AddNew(name, content string) error {
	return f.filesystem.WriteToFile(name, []byte(content))
}

func (f *KnowledgeFiles) Delete(name string) error {
	return f.filesystem.DeleteFile(name)
}

func (f *KnowledgeFiles) Override(name, content string) error {
	return f.filesystem.WriteToFile(name, []byte(content))
}

func (f *KnowledgeFiles) LoadIndex() (*knowledge.KnowledgeIndex, error) {
	data, err := f.filesystem.ReadFile(IndexFile)
	if err != nil {
		return nil, err
	}

	var index knowledge.KnowledgeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return &index, nil
}

func (f *KnowledgeFiles) SaveIndex(idx *knowledge.KnowledgeIndex) error {
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return f.filesystem.WriteToFile(IndexFile, data)
}
