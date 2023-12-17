package model

import (
	"fmt"
	pgtypeuuid "github.com/jackc/pgtype/ext/gofrs-uuid"
	"time"
)

// Presence struct
type Presence struct {
	UpdatedAt *time.Time       `json:"updatedAt,omitempty"`
	Status    *string          `json:"status,omitempty"`
	WorldId   *pgtypeuuid.UUID `json:"spaceId,omitempty"`
	ServerId  *pgtypeuuid.UUID `json:"serverId,omitempty"`
}

func (p *Presence) String() string {
	var out = ""
	out += fmt.Sprintf("updatedAt: %v, ", p.UpdatedAt)
	out += fmt.Sprintf("status: %v, ", p.Status)
	out += fmt.Sprintf("worldId: %v, ", p.WorldId)
	out += fmt.Sprintf("serverId: %v, ", p.ServerId)
	return out
}
