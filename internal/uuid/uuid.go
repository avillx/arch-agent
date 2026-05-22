package uuid

import "github.com/google/uuid"

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator { return &UUIDGenerator{} }

func (g *UUIDGenerator) New() string {
	id := uuid.New()
	return id.String()
}
