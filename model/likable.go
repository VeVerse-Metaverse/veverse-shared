package model

import (
	"fmt"
	"github.com/gofrs/uuid"
)

// Likable entity trait
type Likable struct {
	EntityTrait

	UserId uuid.UUID `json:"userId"` // User who liked or disliked the Entity
	Value  int8      `json:"value"`  // 1 for likes, -1 for dislikes

	Timestamps
}

func (l *Likable) String() string {
	var out = l.EntityTrait.String()
	out += fmt.Sprintf("userId: %v, ", l.UserId)
	out += fmt.Sprintf("value: %v, ", l.Value)
	out += l.Timestamps.String()
	return out
}

type LikableBatch Batch[Likable]

type Rating struct {
	TotalLikes    int32 `json:"likes"`
	TotalDislikes int32 `json:"dislikes"`
}
