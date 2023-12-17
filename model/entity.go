package model

import (
	"context"
	glContext "dev.hackerman.me/artheon/veverse-shared/context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Entity struct {
	Identifier
	Timestamps

	EntityType  string           `json:"entityType,omitempty"`
	Public      bool             `json:"public,omitempty"`
	Views       int32            `json:"views,omitempty"`
	Owner       *User            `json:"owner,omitempty"` // use pointer to avoid infinite recursion
	Accessibles *AccessibleBatch `json:"accessibles,omitempty"`
	Files       *FileBatch       `json:"files,omitempty"`
	Links       *LinkBatch       `json:"links,omitempty"`
	Properties  *PropertyBatch   `json:"properties,omitempty"`
	Likables    *LikableBatch    `json:"likables,omitempty"`
	Comments    *CommentBatch    `json:"comments,omitempty"`
	Liked       *int32           `json:"liked,omitempty"`
	Likes       *int32           `json:"likes,omitempty"`
	Dislikes    *int32           `json:"dislikes,omitempty"`
}

func (e *Entity) String() string {
	var out = e.Identifier.String()
	out += e.Timestamps.String()
	out += fmt.Sprintf("\"entityType\": \"%v\", ", e.EntityType)
	out += fmt.Sprintf("\"public\": \"%v\", ", e.Public)
	out += fmt.Sprintf("\"views\": \"%v\", ", e.Views)
	if e.Owner != nil {
		out += fmt.Sprintf("\"owner\":\n\t{%v\n}, ", e.Owner.String())
	}
	if e.Accessibles != nil {
		out += fmt.Sprintf("\"accessibles\":\n\t{%v\n}, ", (*Batch[Accessible])(e.Accessibles).String())
	}
	if e.Files != nil {
		out += fmt.Sprintf("\"files\":\n\t{%v\n}, ", e.Files.String())
	}
	if e.Links != nil {
		out += fmt.Sprintf("\"links\":\n\t{%v\n}, ", (*Batch[Link])(e.Links).String())
	}
	if e.Properties != nil {
		out += fmt.Sprintf("\"properties\":\n\t{%v\n}, ", (*Batch[Property])(e.Properties).String())
	}
	if e.Likables != nil {
		out += fmt.Sprintf("\"likables\":\n\t{%v\n}, ", (*Batch[Likable])(e.Likables).String())
	}
	if e.Comments != nil {
		out += fmt.Sprintf("\"comments\":\n\t{%v\n}, ", (*Batch[Comment])(e.Comments).String())
	}
	return out
}

func (e *Entity) InitAccessibles() {
	e.Accessibles = &AccessibleBatch{}
}

func (e *Entity) InitFiles() {
	e.Files = &FileBatch{}
}

func (e *Entity) InitLinks() {
	e.Links = &LinkBatch{}
}

func (e *Entity) InitProperties() {
	e.Properties = &PropertyBatch{}
}

func (e *Entity) InitLikables() {
	e.Likables = &LikableBatch{}
}

func (e *Entity) InitComments() {
	e.Comments = &CommentBatch{}
}

func RequestIsOwnerOfEntity(ctx context.Context, requester *User, id uuid.UUID) (bool, error) {
	if requester == nil {
		return false, ErrNoRequester
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return false, ErrNoDatabase
	}

	var (
		q    string
		rows pgx.Rows
	)

	q = `SELECT a.is_owner FROM entities e LEFT JOIN accessibles a ON e.id = a.entity_id WHERE e.id = $1 AND a.user_id = $2`
	rows, err := db.Query(ctx, q, id, requester.Id)
	if err != nil {
		return false, err
	}

	defer rows.Close()
	for rows.Next() {
		var a Accessible
		err := rows.Scan(&a.IsOwner)
		if err != nil {
			return false, err
		}

		if a.IsOwner {
			return true, nil
		}
	}

	return false, nil
}

func RequestCanViewEntity(ctx context.Context, requester *User, id uuid.UUID) (bool, error) {
	if requester == nil {
		return false, ErrNoRequester
	}

	if requester.IsAdmin {
		return true, nil
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return false, ErrNoDatabase
	}

	var (
		q    string
		rows pgx.Rows
	)

	q = `SELECT a.is_owner, can_view, public FROM entities e LEFT JOIN accessibles a ON e.id = a.entity_id WHERE e.id = $1 AND a.user_id = $2`
	rows, err := db.Query(ctx, q, id, requester.Id)
	if err != nil {
		return false, err
	}

	defer rows.Close()
	for rows.Next() {
		var public bool
		var a Accessible
		err := rows.Scan(&a.IsOwner, &a.CanView, &public)
		if err != nil {
			return false, err
		}

		if a.IsOwner || a.CanView || public {
			return true, nil
		}
	}

	return false, nil
}

func RequestCanEditEntity(ctx context.Context, requester *User, id uuid.UUID) (bool, error) {
	if requester == nil {
		return false, ErrNoRequester
	}

	if requester.IsAdmin {
		return true, nil
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return false, ErrNoDatabase
	}

	var (
		q    string
		rows pgx.Rows
	)

	q = `SELECT a.is_owner, can_edit, public FROM entities e LEFT JOIN accessibles a ON e.id = a.entity_id WHERE e.id = $1 AND a.user_id = $2`
	rows, err := db.Query(ctx, q, id, requester.Id)
	if err != nil {
		return false, err
	}

	defer rows.Close()
	for rows.Next() {
		var a Accessible
		err := rows.Scan(&a.IsOwner, &a.CanEdit)
		if err != nil {
			return false, err
		}

		if a.IsOwner || a.CanEdit {
			return true, nil
		}
	}

	return false, nil
}

func RequestCanDeleteEntity(ctx context.Context, requester *User, id uuid.UUID) (bool, error) {
	if requester == nil {
		return false, ErrNoRequester
	}

	if requester.IsAdmin {
		return true, nil
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return false, ErrNoDatabase
	}

	var (
		q    string
		rows pgx.Rows
	)

	q = `SELECT a.is_owner, can_delete, public FROM entities e LEFT JOIN accessibles a ON e.id = a.entity_id WHERE e.id = $1 AND a.user_id = $2`
	rows, err := db.Query(ctx, q, id, requester.Id)
	if err != nil {
		return false, err
	}

	defer rows.Close()
	for rows.Next() {
		var a Accessible
		err := rows.Scan(&a.IsOwner, &a.CanDelete)
		if err != nil {
			return false, err
		}

		if a.IsOwner || a.CanDelete {
			return true, nil
		}
	}

	return false, nil
}

type AccessEntityMetadata struct {
	UserId    uuid.UUID `json:"userId" validate:"required"`
	CanView   *bool     `json:"canView,omitempty"`
	CanEdit   *bool     `json:"canEdit,omitempty"`
	CanDelete *bool     `json:"canDelete,omitempty"`
	Public    *bool     `json:"public"`
}
