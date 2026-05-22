package files

import (
	"arch-agent/internal/knowledge"
	"encoding/json"
	"os"
)

type KnowledgeFiles struct {
	fs *FileSystem
}

func NewKnowledgeFiles(fs *FileSystem) *KnowledgeFiles {
	return &KnowledgeFiles{fs: fs}
}

func (f *KnowledgeFiles) Read(name string) (string, error) {
	data, err := f.fs.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *KnowledgeFiles) AddNew(name, content string) error {
	return f.fs.WriteToFile(name, []byte(content))
}

func (f *KnowledgeFiles) Delete(name string) error {
	return f.fs.DeleteFile(name)
}

func (f *KnowledgeFiles) Override(name, content string) error {
	return f.fs.WriteToFile(name, []byte(content))
}

func (f *KnowledgeFiles) LoadIndex() (*knowledge.Index, error) {
	data, err := f.fs.ReadFile("knowledge.json")
	if err != nil {
		if os.IsNotExist(err) {
			return &knowledge.Index{}, nil
		}
		return nil, err
	}

	var index knowledge.Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return &index, nil
}

func (f *KnowledgeFiles) SaveIndex(idx *knowledge.Index) error {
	data, err := json.MarshalIndent(idx, "", "	")
	if err != nil {
		return err
	}
	return f.fs.WriteToFile("knowledge.json", data)
}
