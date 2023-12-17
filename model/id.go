package model

import (
	"fmt"
	"github.com/gofrs/uuid"
)

type Identifier struct {
	Id uuid.UUID `json:"id,omitempty" validate:"required"`
}

func (e Identifier) String() string {
	return fmt.Sprintf("id: %v", e.Id)
}

// GetId returns the Id of the Identifier.
// Notice the receiver is a value, not a pointer, it is important for the interface to work properly on casts
func (e Identifier) GetId() uuid.UUID {
	return e.Id
}

type Identifiable interface {
	GetId() uuid.UUID
}

func ContainsIdentifiable(a []any, id uuid.UUID) bool {
	for _, v := range a {
		identifiable, ok := v.(Identifiable)
		if ok && identifiable.GetId() == id {
			return true
		}
	}
	return false
}

func GetIdentifiableIndex(a []any, id uuid.UUID) int {
	for i, v := range a {
		identifiable, ok := v.(Identifiable)
		if ok && identifiable.GetId() == id {
			return i
		}
	}
	const IndexNotFound = -1
	return IndexNotFound
}
