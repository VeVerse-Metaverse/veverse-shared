package model

import (
	"fmt"
	"github.com/gofrs/uuid"
)

type Comment struct {
	//Entity
	EntityTrait

	UserId uuid.UUID `json:"userId"`
	Text   string    `json:"text"`
}

func (c *Comment) String() string {
	var out = c.EntityTrait.String()
	out += fmt.Sprintf("userId: %v, ", c.UserId)
	out += fmt.Sprintf("text: %v, ", c.Text)
	return out
}

type CommentBatch Batch[Comment]
