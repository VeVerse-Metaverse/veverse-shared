package model

import (
	"fmt"
	"github.com/gofrs/uuid"
)

type EntityIdentifier struct {
	EntityId *uuid.UUID `json:"entityId,omitempty"`
}

// EntityTrait Base for traits related to the entity
type EntityTrait struct {
	Identifier
	EntityId *uuid.UUID `json:"entityId,omitempty"`
}

func (e *EntityTrait) String() string {
	var out = e.Identifier.String()
	if e.EntityId != nil {
		out += fmt.Sprintf("\"entityId\": \"%v\", ", e.EntityId)
	}
	return out
}
