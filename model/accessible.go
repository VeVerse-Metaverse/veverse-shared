package model

import (
	"fmt"
	"github.com/gofrs/uuid"
)

// Accessible Entity accessible trait
type Accessible struct {
	EntityTrait

	UserId    uuid.UUID `json:"userId"`
	Username  string    `json:"username,omitempty"`
	IsOwner   bool      `json:"isOwner"`
	CanView   bool      `json:"canView"`
	CanEdit   bool      `json:"canEdit"`
	CanDelete bool      `json:"canDelete"`

	Timestamps
}

func (a *Accessible) String() string {
	var out = a.EntityTrait.String()
	out += fmt.Sprintf("userId: %v, ", a.UserId)
	if a.Username != "" {
		out += fmt.Sprintf("username: %v, ", a.Username)
	}
	out += fmt.Sprintf("isOwner: %v, ", a.IsOwner)
	out += fmt.Sprintf("canView: %v, ", a.CanView)
	out += fmt.Sprintf("canEdit: %v, ", a.CanEdit)
	out += fmt.Sprintf("canDelete: %v, ", a.CanDelete)
	out += a.Timestamps.String()
	return out
}

type AccessibleBatch Batch[Accessible]
