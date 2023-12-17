package model

import (
	"fmt"
	"time"
)

// Timestamps Base created/updated timestamps
type Timestamps struct {
	CreatedAt time.Time  `json:"createdAt,omitempty" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
}

func (t *Timestamps) String() string {
	var out = fmt.Sprintf("createdAt: %v, ", t.CreatedAt)
	out += fmt.Sprintf("updatedAt: %v, ", t.UpdatedAt)
	return out
}
