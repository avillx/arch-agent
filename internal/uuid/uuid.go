package uuid

import (
	"github.com/rs/xid"
)

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator { return &UUIDGenerator{} }

func (g *UUIDGenerator) New() string {
	id := xid.New()
	return id.String()
}
