package knowledge

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Index struct {
	Knowledges []*Record `json:"knowledges"`
}

func (idx *Index) DeleteRecord(name string) {
	idx.Knowledges = slices.DeleteFunc(idx.Knowledges, func(k *Record) bool {
		return k.Name == name
	})
}

func (idx *Index) FindRecord(name string) (*Record, error) {
	// index of is a array index in Knowledges
	indexOf := slices.IndexFunc(idx.Knowledges, func(k *Record) bool {
		return k.Name == name
	})
	if indexOf == -1 {
		return nil, errors.New("Knowledge is not in index")
	}
	return idx.Knowledges[indexOf], nil
}

func (idx *Index) AddRecord(k *Record) {
	idx.Knowledges = append(idx.Knowledges, k)
}

func (idx *Index) List() string {
	var sb strings.Builder
	for _, rec := range idx.Knowledges {
		stringRec := fmt.Sprintf("* %s - %s\n", rec.Name, rec.Description)
		sb.WriteString(stringRec)
	}
	return sb.String()
}

func (idx *Index) ListDetailed() string {
	var sb strings.Builder
	for _, rec := range idx.Knowledges {
		stringRec := fmt.Sprintf("* %s (%s) - %s\n", rec.Name, rec.Grade(), rec.Description)
		sb.WriteString(stringRec)
	}
	return sb.String()
}
