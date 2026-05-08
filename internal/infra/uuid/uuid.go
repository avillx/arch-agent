package uuid

import "github.com/google/uuid"

type UUIDGenerator struct{}

func (g *UUIDGenerator) New() string {
	id := uuid.New()
	return id.String()
}
