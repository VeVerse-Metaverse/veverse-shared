package model

import (
	"context"
	glContext "dev.hackerman.me/artheon/veverse-shared/context"
	"dev.hackerman.me/artheon/veverse-shared/helper"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	pgtypeuuid "github.com/jackc/pgtype/ext/gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Entity

	Email              *string    `json:"email,omitempty"`
	PasswordHash       *string    `json:"-"`
	ApiKey             *string    `json:"apiKey,omitempty"`
	Name               *string    `json:"name"`
	Description        *string    `json:"description,omitempty"`
	Ip                 *string    `json:"ip,omitempty"`
	GeoLocation        *string    `json:"geoLocation,omitempty"`
	IsActive           bool       `json:"isActive,omitempty"`
	IsAdmin            bool       `json:"isAdmin,omitempty"`
	IsMuted            bool       `json:"isMuted,omitempty"`
	IsBanned           bool       `json:"isBanned,omitempty"`
	IsInternal         bool       `json:"isInternal,omitempty"`
	LastSeenAt         *time.Time `json:"lastSeenAt,omitempty"`
	ActivatedAt        *time.Time `json:"activatedAt,omitempty"`
	AllowEmails        bool       `json:"allowEmails,omitempty"`
	Experience         int32      `json:"experience,omitempty"`
	Level              int32      `json:"level,omitempty"`
	Rank               string     `json:"rank,omitempty"`
	EthAddress         *string    `json:"ethAddress,omitempty"`
	Address            *string    `json:"address,omitempty"`
	DefaultPersonaId   *uuid.UUID `json:"defaultPersonaId,omitempty"`
	DefaultPersona     *Persona   `json:"defaultPersona,omitempty"`
	Presence           *Presence  `json:"presence,omitempty"`
	IsEmailConfirmed   bool       `json:"isEmailConfirmed,omitempty"`
	IsAddressConfirmed bool       `json:"isAddressConfirmed,omitempty"`
}

func (u *User) String() string {
	var out = u.Entity.String()
	if u.Email != nil {
		out += fmt.Sprintf("email: %v, ", *u.Email)
	}
	out += fmt.Sprintf("name: %v, ", u.Name)
	if u.Description != nil {
		out += fmt.Sprintf("description: %v, ", *u.Description)
	}
	out += fmt.Sprintf("isActive: %v, ", u.IsActive)
	out += fmt.Sprintf("isAdmin: %v, ", u.IsAdmin)
	out += fmt.Sprintf("isMuted: %v, ", u.IsMuted)
	out += fmt.Sprintf("isBanned: %v, ", u.IsBanned)
	out += fmt.Sprintf("isInternal: %v, ", u.IsInternal)
	if u.LastSeenAt != nil {
		out += fmt.Sprintf("lastSeenAt: %v, ", *u.LastSeenAt)
	}
	if u.ActivatedAt != nil {
		out += fmt.Sprintf("activatedAt: %v, ", *u.ActivatedAt)
	}
	out += fmt.Sprintf("allowEmails: %v, ", u.AllowEmails)
	out += fmt.Sprintf("experience: %v, ", u.Experience)
	out += fmt.Sprintf("level: %v, ", u.Level)
	out += fmt.Sprintf("rank: %v, ", u.Rank)
	if u.EthAddress != nil {
		out += fmt.Sprintf("ethAddress: %v, ", *u.EthAddress)
	}
	if u.Address != nil {
		out += fmt.Sprintf("address: %v, ", *u.Address)
	}
	if u.DefaultPersona != nil {
		out += fmt.Sprintf("defaultPersona: %v, ", u.DefaultPersona.String())
	}
	if u.Presence != nil {
		out += fmt.Sprintf("presence: %v, ", u.Presence.String())
	}
	return out
}

type UserBatch Batch[User]

type IndexUserRequest struct {
	Offset *int64  `json:"offset,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
	Search *string `json:"search,omitempty"`
}

func IndexUser(ctx context.Context, requester *User, request IndexUserRequest) (entities *UserBatch, err error) {
	// validate requester
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	var batch = UserBatch{
		Offset: 0,
		Limit:  100,
		Total:  0,
	}

	if request.Offset != nil && *request.Offset >= 0 {
		batch.Offset = *request.Offset
	}

	if request.Limit != nil && *request.Limit > 0 && *request.Limit <= 100 {
		batch.Limit = *request.Limit
	}

	var (
		qt          string
		qtArgs      []interface{}
		qtArgsIndex = 1
		q           string
		qArgs       []interface{}
		qArgsIndex  = 1
		rows        pgx.Rows
	)

	qt = `select count(*) from users u left join entities e on u.id = e.id `
	q = `select e.id,
       e.created_at,
       e.updated_at,
       e.entity_type,
       e.views,
       e.public,
       u.email,
       u.name,
       u.description,
       u.ip,
       u.geolocation,
       u.is_active,
       u.is_admin,
       u.is_muted,
       u.is_banned,
       u.is_internal,
       u.last_seen_at,
       u.activated_at,
       u.allow_emails,
       u.experience,
       u.eth_address,
       u.address,
       u.default_persona_id,
       u.is_email_confirmed,
       u.is_address_confirmed
from users u
         left join entities e on u.id = e.id `

	// check access, if the requester is an admin, they can see all users, otherwise they can only see themselves, their friends and public users
	if requester.IsAdmin {
		qt += ` where true `
		q += ` where true `
	} else {

		qt += ` left join accessibles a on (a.entity_id = e.id and a.user_id = $` + strconv.Itoa(qtArgsIndex)
		qtArgs = append(qtArgs, requester.Id)
		qtArgsIndex++
		qt += `::uuid) where (e.public = true or u.id = $` + strconv.Itoa(qtArgsIndex)
		qtArgs = append(qtArgs, requester.Id)
		qtArgsIndex++
		qt += `::uuid or a.can_view) `

		q += ` left join accessibles a on (a.entity_id = e.id and a.user_id = $` + strconv.Itoa(qArgsIndex)
		qArgs = append(qArgs, requester.Id)
		qArgsIndex++
		q += `::uuid) where (e.public = true or u.id = $` + strconv.Itoa(qArgsIndex)
		qArgs = append(qArgs, requester.Id)
		qArgsIndex++
		q += `::uuid or a.can_view) `
	}

	// additional params (search)
	if request.Search != nil {
		search := "%" + helper.SanitizeLikeClause(*request.Search) + "%"

		qt += ` and (name ilike $` + strconv.Itoa(qtArgsIndex)
		qtArgs = append(qtArgs, search)
		qtArgsIndex++
		qt += `::text or email ilike $` + strconv.Itoa(qtArgsIndex)
		qtArgs = append(qtArgs, search)
		qtArgsIndex++
		qt += `::text) `

		q += ` and (name ilike $` + strconv.Itoa(qArgsIndex)
		qArgs = append(qArgs, search)
		qArgsIndex++
		q += `::text or email ilike $` + strconv.Itoa(qArgsIndex)
		qArgs = append(qArgs, search)
		qArgsIndex++
		q += `::text) `
	}

	// order, offset and limit
	qArgs = append(qArgs, batch.Offset, batch.Limit)
	q += ` order by e.created_at desc offset $` + strconv.Itoa(qArgsIndex) + `::int8 limit $` + strconv.Itoa(qArgsIndex+1) + `::int8;`

	// get the total
	err = db.QueryRow(ctx, qt, qtArgs...).Scan(&batch.Total)
	if err != nil {
		return nil, err
	}

	if batch.Total == 0 {
		return &batch, nil
	}

	fmt.Println(q, qArgs)

	// get the entities
	rows, err = db.Query(ctx, q, qArgs...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			user   User
			entity Entity

			entityId           pgtypeuuid.UUID
			entityCreatedAt    pgtype.Timestamptz
			entityUpdatedAt    pgtype.Timestamptz
			entityType         pgtype.Text
			entityViews        pgtype.Int4
			entityPublic       pgtype.Bool
			email              pgtype.Text
			name               pgtype.Text
			description        pgtype.Text
			ip                 pgtype.Text
			geoLocation        pgtype.Text
			isActive           pgtype.Bool
			isAdmin            pgtype.Bool
			isMuted            pgtype.Bool
			isBanned           pgtype.Bool
			isInternal         pgtype.Bool
			lastSeenAt         pgtype.Timestamptz
			activatedAt        pgtype.Timestamptz
			allowEmails        pgtype.Bool
			experience         pgtype.Int4
			ethAddress         pgtype.Text
			address            pgtype.Text
			defaultPersonaId   pgtypeuuid.UUID
			isEmailConfirmed   pgtype.Bool
			isAddressConfirmed pgtype.Bool
		)
		err = rows.Scan(&entityId,
			&entityCreatedAt,
			&entityUpdatedAt,
			&entityType,
			&entityViews,
			&entityPublic,
			&email,
			&name,
			&description,
			&ip,
			&geoLocation,
			&isActive,
			&isAdmin,
			&isMuted,
			&isBanned,
			&isInternal,
			&lastSeenAt,
			&activatedAt,
			&allowEmails,
			&experience,
			&ethAddress,
			&address,
			&defaultPersonaId,
			&isEmailConfirmed,
			&isAddressConfirmed)
		if err != nil {
			return nil, err
		}

		if entityId.Status == pgtype.Present {
			entity.Id = entityId.UUID
			if entityCreatedAt.Status == pgtype.Present {
				entity.CreatedAt = entityCreatedAt.Time
			}
			if entityUpdatedAt.Status == pgtype.Present {
				entity.UpdatedAt = &entityUpdatedAt.Time
			}
			if entityType.Status == pgtype.Present {
				entity.EntityType = entityType.String
			}
			if entityViews.Status == pgtype.Present {
				entity.Views = entityViews.Int
			}
			if entityPublic.Status == pgtype.Present {
				entity.Public = entityPublic.Bool
			}
			user.Entity = entity
			if email.Status == pgtype.Present {
				user.Email = &email.String
			}
			if name.Status == pgtype.Present {
				user.Name = &name.String
			}
			if description.Status == pgtype.Present {
				user.Description = &description.String
			}
			if ip.Status == pgtype.Present {
				user.Ip = &ip.String
			}
			if geoLocation.Status == pgtype.Present {
				user.GeoLocation = &geoLocation.String
			}
			if isActive.Status == pgtype.Present {
				user.IsActive = isActive.Bool
			}
			if isAdmin.Status == pgtype.Present {
				user.IsAdmin = isAdmin.Bool
			}
			if isMuted.Status == pgtype.Present {
				user.IsMuted = isMuted.Bool
			}
			if isBanned.Status == pgtype.Present {
				user.IsBanned = isBanned.Bool
			}
			if isInternal.Status == pgtype.Present {
				user.IsInternal = isInternal.Bool
			}
			if lastSeenAt.Status == pgtype.Present {
				user.LastSeenAt = &lastSeenAt.Time
			}
			if activatedAt.Status == pgtype.Present {
				user.ActivatedAt = &activatedAt.Time
			}
			if allowEmails.Status == pgtype.Present {
				user.AllowEmails = allowEmails.Bool
			}
			if experience.Status == pgtype.Present {
				user.Experience = experience.Int
			}
			if ethAddress.Status == pgtype.Present {
				user.EthAddress = &ethAddress.String
			}
			if address.Status == pgtype.Present {
				user.Address = &address.String
			}
			if defaultPersonaId.Status == pgtype.Present {
				user.DefaultPersonaId = &defaultPersonaId.UUID
			}
			if isEmailConfirmed.Status == pgtype.Present {
				user.IsEmailConfirmed = isEmailConfirmed.Bool
			}
			if isAddressConfirmed.Status == pgtype.Present {
				user.IsAddressConfirmed = isAddressConfirmed.Bool
			}
			if !ContainsIdentifiable(helper.ToSliceOfAny(batch.Entities), user.Id) {
				batch.Entities = append(batch.Entities, user)
			}
		}
	}

	return &batch, nil
}

type GetUserByIdRequest struct {
	Id uuid.UUID
}

func GetUserById(ctx context.Context, request GetUserByIdRequest) (e *User, err error) {
	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	q := `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, u.email, u.name, u.description, u.ip, u.geolocation, u.is_active, u.is_admin, u.is_muted, u.is_banned, u.is_internal, u.last_seen_at, u.activated_at, u.allow_emails, u.experience, u.eth_address, u.address, u.default_persona_id, u.is_email_confirmed, u.is_address_confirmed
	from entities e
	inner join users u on e.id = u.id
	where u.id = $1::uuid`

	var (
		user   User
		entity Entity

		entityId           pgtypeuuid.UUID
		entityCreatedAt    pgtype.Timestamptz
		entityUpdatedAt    pgtype.Timestamptz
		entityType         pgtype.Text
		entityViews        pgtype.Int4
		entityPublic       pgtype.Bool
		email              pgtype.Text
		name               pgtype.Text
		description        pgtype.Text
		ip                 pgtype.Text
		geoLocation        pgtype.Text
		isActive           pgtype.Bool
		isAdmin            pgtype.Bool
		isMuted            pgtype.Bool
		isBanned           pgtype.Bool
		isInternal         pgtype.Bool
		lastSeenAt         pgtype.Timestamptz
		activatedAt        pgtype.Timestamptz
		allowEmails        pgtype.Bool
		experience         pgtype.Int4
		ethAddress         pgtype.Text
		address            pgtype.Text
		defaultPersonaId   pgtypeuuid.UUID
		isEmailConfirmed   pgtype.Bool
		isAddressConfirmed pgtype.Bool
	)
	err = db.QueryRow(ctx, q, request.Id).Scan(&entityId,
		&entityCreatedAt,
		&entityUpdatedAt,
		&entityType,
		&entityViews,
		&entityPublic,
		&email,
		&name,
		&description,
		&ip,
		&geoLocation,
		&isActive,
		&isAdmin,
		&isMuted,
		&isBanned,
		&isInternal,
		&lastSeenAt,
		&activatedAt,
		&allowEmails,
		&experience,
		&ethAddress,
		&address,
		&defaultPersonaId,
		&isEmailConfirmed,
		&isAddressConfirmed)
	if err != nil {
		return nil, err
	}

	if entityId.Status == pgtype.Present {
		entity.Id = entityId.UUID
		if entityCreatedAt.Status == pgtype.Present {
			entity.CreatedAt = entityCreatedAt.Time
		}
		if entityUpdatedAt.Status == pgtype.Present {
			entity.UpdatedAt = &entityUpdatedAt.Time
		}
		if entityType.Status == pgtype.Present {
			entity.EntityType = entityType.String
		}
		if entityViews.Status == pgtype.Present {
			entity.Views = entityViews.Int
		}
		if entityPublic.Status == pgtype.Present {
			entity.Public = entityPublic.Bool
		}
		user.Entity = entity
		if email.Status == pgtype.Present {
			user.Email = &email.String
		}
		if name.Status == pgtype.Present {
			user.Name = &name.String
		}
		if description.Status == pgtype.Present {
			user.Description = &description.String
		}
		if ip.Status == pgtype.Present {
			user.Ip = &ip.String
		}
		if geoLocation.Status == pgtype.Present {
			user.GeoLocation = &geoLocation.String
		}
		if isActive.Status == pgtype.Present {
			user.IsActive = isActive.Bool
		}
		if isAdmin.Status == pgtype.Present {
			user.IsAdmin = isAdmin.Bool
		}
		if isMuted.Status == pgtype.Present {
			user.IsMuted = isMuted.Bool
		}
		if isBanned.Status == pgtype.Present {
			user.IsBanned = isBanned.Bool
		}
		if isInternal.Status == pgtype.Present {
			user.IsInternal = isInternal.Bool
		}
		if lastSeenAt.Status == pgtype.Present {
			user.LastSeenAt = &lastSeenAt.Time
		}
		if activatedAt.Status == pgtype.Present {
			user.ActivatedAt = &activatedAt.Time
		}
		if allowEmails.Status == pgtype.Present {
			user.AllowEmails = allowEmails.Bool
		}
		if experience.Status == pgtype.Present {
			user.Experience = experience.Int
		}
		if ethAddress.Status == pgtype.Present {
			user.EthAddress = &ethAddress.String
		}
		if address.Status == pgtype.Present {
			user.Address = &address.String
		}
		if defaultPersonaId.Status == pgtype.Present {
			user.DefaultPersonaId = &defaultPersonaId.UUID
		}
		if isEmailConfirmed.Status == pgtype.Present {
			user.IsEmailConfirmed = isEmailConfirmed.Bool
		}
		if isAddressConfirmed.Status == pgtype.Present {
			user.IsAddressConfirmed = isAddressConfirmed.Bool
		}
	}

	return &user, nil
}

type GetUserByEmailRequest struct {
	Email string
}

func GetUserByEmail(ctx context.Context, request GetUserByEmailRequest) (e *User, err error) {
	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	q := `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, u.email, u.name, u.description, u.ip, u.geolocation, u.is_active, u.is_admin, u.is_muted, u.is_banned, u.is_internal, u.last_seen_at, u.activated_at, u.allow_emails, u.experience, u.eth_address, u.address, u.default_persona_id, u.is_email_confirmed, u.is_address_confirmed
	from entities e
	inner join users u on e.id = u.id
	where u.email = $1::text`

	var (
		user   User
		entity Entity

		entityId           pgtypeuuid.UUID
		entityCreatedAt    pgtype.Timestamptz
		entityUpdatedAt    pgtype.Timestamptz
		entityType         pgtype.Text
		entityViews        pgtype.Int4
		entityPublic       pgtype.Bool
		email              pgtype.Text
		name               pgtype.Text
		description        pgtype.Text
		ip                 pgtype.Text
		geoLocation        pgtype.Text
		isActive           pgtype.Bool
		isAdmin            pgtype.Bool
		isMuted            pgtype.Bool
		isBanned           pgtype.Bool
		isInternal         pgtype.Bool
		lastSeenAt         pgtype.Timestamptz
		activatedAt        pgtype.Timestamptz
		allowEmails        pgtype.Bool
		experience         pgtype.Int4
		ethAddress         pgtype.Text
		address            pgtype.Text
		defaultPersonaId   pgtypeuuid.UUID
		isEmailConfirmed   pgtype.Bool
		isAddressConfirmed pgtype.Bool
	)
	err = db.QueryRow(ctx, q, request.Email).Scan(&entityId,
		&entityCreatedAt,
		&entityUpdatedAt,
		&entityType,
		&entityViews,
		&entityPublic,
		&email,
		&name,
		&description,
		&ip,
		&geoLocation,
		&isActive,
		&isAdmin,
		&isMuted,
		&isBanned,
		&isInternal,
		&lastSeenAt,
		&activatedAt,
		&allowEmails,
		&experience,
		&ethAddress,
		&address,
		&defaultPersonaId,
		&isEmailConfirmed,
		&isAddressConfirmed)
	if err != nil {
		return nil, err
	}

	if entityId.Status == pgtype.Present {
		entity.Id = entityId.UUID
		if entityCreatedAt.Status == pgtype.Present {
			entity.CreatedAt = entityCreatedAt.Time
		}
		if entityUpdatedAt.Status == pgtype.Present {
			entity.UpdatedAt = &entityUpdatedAt.Time
		}
		if entityType.Status == pgtype.Present {
			entity.EntityType = entityType.String
		}
		if entityViews.Status == pgtype.Present {
			entity.Views = entityViews.Int
		}
		if entityPublic.Status == pgtype.Present {
			entity.Public = entityPublic.Bool
		}
		user.Entity = entity
		if email.Status == pgtype.Present {
			user.Email = &email.String
		}
		if name.Status == pgtype.Present {
			user.Name = &name.String
		}
		if description.Status == pgtype.Present {
			user.Description = &description.String
		}
		if ip.Status == pgtype.Present {
			user.Ip = &ip.String
		}
		if geoLocation.Status == pgtype.Present {
			user.GeoLocation = &geoLocation.String
		}
		if isActive.Status == pgtype.Present {
			user.IsActive = isActive.Bool
		}
		if isAdmin.Status == pgtype.Present {
			user.IsAdmin = isAdmin.Bool
		}
		if isMuted.Status == pgtype.Present {
			user.IsMuted = isMuted.Bool
		}
		if isBanned.Status == pgtype.Present {
			user.IsBanned = isBanned.Bool
		}
		if isInternal.Status == pgtype.Present {
			user.IsInternal = isInternal.Bool
		}
		if lastSeenAt.Status == pgtype.Present {
			user.LastSeenAt = &lastSeenAt.Time
		}
		if activatedAt.Status == pgtype.Present {
			user.ActivatedAt = &activatedAt.Time
		}
		if allowEmails.Status == pgtype.Present {
			user.AllowEmails = allowEmails.Bool
		}
		if experience.Status == pgtype.Present {
			user.Experience = experience.Int
		}
		if ethAddress.Status == pgtype.Present {
			user.EthAddress = &ethAddress.String
		}
		if address.Status == pgtype.Present {
			user.Address = &address.String
		}
		if defaultPersonaId.Status == pgtype.Present {
			user.DefaultPersonaId = &defaultPersonaId.UUID
		}
		if isEmailConfirmed.Status == pgtype.Present {
			user.IsEmailConfirmed = isEmailConfirmed.Bool
		}
		if isAddressConfirmed.Status == pgtype.Present {
			user.IsAddressConfirmed = isAddressConfirmed.Bool
		}
	}

	return &user, nil
}

type GetUserByEthAddressRequest struct {
	EthAddress string
}

func GetUserByEthAddress(ctx context.Context, request GetUserByEthAddressRequest) (u *User, err error) {
	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	q := `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, u.email, u.name, u.description, u.ip, u.geolocation, u.is_active, u.is_admin, u.is_muted, u.is_banned, u.is_internal, u.last_seen_at, u.activated_at, u.allow_emails, u.experience, u.eth_address, u.address, u.default_persona_id, u.is_email_confirmed, u.is_address_confirmed
	from entities e
	inner join users u on e.id = u.id
	where u.eth_address = $1::text`

	var (
		user   User
		entity Entity

		entityId           pgtypeuuid.UUID
		entityCreatedAt    pgtype.Timestamptz
		entityUpdatedAt    pgtype.Timestamptz
		entityType         pgtype.Text
		entityViews        pgtype.Int4
		entityPublic       pgtype.Bool
		email              pgtype.Text
		name               pgtype.Text
		description        pgtype.Text
		ip                 pgtype.Text
		geoLocation        pgtype.Text
		isActive           pgtype.Bool
		isAdmin            pgtype.Bool
		isMuted            pgtype.Bool
		isBanned           pgtype.Bool
		isInternal         pgtype.Bool
		lastSeenAt         pgtype.Timestamptz
		activatedAt        pgtype.Timestamptz
		allowEmails        pgtype.Bool
		experience         pgtype.Int4
		ethAddress         pgtype.Text
		address            pgtype.Text
		defaultPersonaId   pgtypeuuid.UUID
		isEmailConfirmed   pgtype.Bool
		isAddressConfirmed pgtype.Bool
	)
	err = db.QueryRow(ctx, q, request.EthAddress).Scan(&entityId,
		&entityCreatedAt,
		&entityUpdatedAt,
		&entityType,
		&entityViews,
		&entityPublic,
		&email,
		&name,
		&description,
		&ip,
		&geoLocation,
		&isActive,
		&isAdmin,
		&isMuted,
		&isBanned,
		&isInternal,
		&lastSeenAt,
		&activatedAt,
		&allowEmails,
		&experience,
		&ethAddress,
		&address,
		&defaultPersonaId,
		&isEmailConfirmed,
		&isAddressConfirmed)
	if err != nil {
		return nil, err
	}

	if entityId.Status == pgtype.Present {
		entity.Id = entityId.UUID
		if entityCreatedAt.Status == pgtype.Present {
			entity.CreatedAt = entityCreatedAt.Time
		}
		if entityUpdatedAt.Status == pgtype.Present {
			entity.UpdatedAt = &entityUpdatedAt.Time
		}
		if entityType.Status == pgtype.Present {
			entity.EntityType = entityType.String
		}
		if entityViews.Status == pgtype.Present {
			entity.Views = entityViews.Int
		}
		if entityPublic.Status == pgtype.Present {
			entity.Public = entityPublic.Bool
		}
		user.Entity = entity
		if email.Status == pgtype.Present {
			user.Email = &email.String
		}
		if name.Status == pgtype.Present {
			user.Name = &name.String
		}
		if description.Status == pgtype.Present {
			user.Description = &description.String
		}
		if ip.Status == pgtype.Present {
			user.Ip = &ip.String
		}
		if geoLocation.Status == pgtype.Present {
			user.GeoLocation = &geoLocation.String
		}
		if isActive.Status == pgtype.Present {
			user.IsActive = isActive.Bool
		}
		if isAdmin.Status == pgtype.Present {
			user.IsAdmin = isAdmin.Bool
		}
		if isMuted.Status == pgtype.Present {
			user.IsMuted = isMuted.Bool
		}
		if isBanned.Status == pgtype.Present {
			user.IsBanned = isBanned.Bool
		}
		if isInternal.Status == pgtype.Present {
			user.IsInternal = isInternal.Bool
		}
		if lastSeenAt.Status == pgtype.Present {
			user.LastSeenAt = &lastSeenAt.Time
		}
		if activatedAt.Status == pgtype.Present {
			user.ActivatedAt = &activatedAt.Time
		}
		if allowEmails.Status == pgtype.Present {
			user.AllowEmails = allowEmails.Bool
		}
		if experience.Status == pgtype.Present {
			user.Experience = experience.Int
		}
		if ethAddress.Status == pgtype.Present {
			user.EthAddress = &ethAddress.String
		}
		if address.Status == pgtype.Present {
			user.Address = &address.String
		}
		if defaultPersonaId.Status == pgtype.Present {
			user.DefaultPersonaId = &defaultPersonaId.UUID
		}
		if isEmailConfirmed.Status == pgtype.Present {
			user.IsEmailConfirmed = isEmailConfirmed.Bool
		}
		if isAddressConfirmed.Status == pgtype.Present {
			user.IsAddressConfirmed = isAddressConfirmed.Bool
		}
	}

	return &user, nil
}

type NonceRequestMetadata struct {
	Address string `query:"address,required"`
}

type RegisterUserRequestFromOAuthWithEmail struct {
	Email string `json:"email" validate:"required,email"`
}

func RegisterUserFromOAuthWithEmail(ctx context.Context, request RegisterUserRequestFromOAuthWithEmail) (u *User, err error) {
	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	rand.Seed(time.Now().UnixNano())
	randomName := fmt.Sprintf("User %06d", rand.Intn(1000000-100000)+100000)

	q := `with r as (insert into entities (id, created_at, updated_at, entity_type, public, views)
    values (gen_random_uuid(), now(), null, 'user', true, null)
    returning id)
insert
into users (id, email, name, is_email_confirmed, api_key, eth_address, is_active, is_admin, is_muted, is_banned, is_internal, allow_emails, experience)
select r.id, $1::text, $2::text, true, $3::text, '', true, false, false, false, false, true, 0
from r
returning id;`

	randomKeyUuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	randomKey := strings.ReplaceAll(randomKeyUuid.String(), "-", "")

	row := db.QueryRow(ctx, q, request.Email, randomName, randomKey)
	user := &User{}
	err = row.Scan(&user.Id)
	if err != nil {
		return nil, err
	}

	q = `with r as (insert into entities (id, created_at, updated_at, entity_type, public, views) values (gen_random_uuid(), now(), null, 'persona', true, null) returning id)
insert into personas (id, name, type, configuration, user_id) select r.id, $1::text, $2::text, $3::jsonb, $4::uuid from r`
	_, err = db.Exec(ctx, q, randomName, "RPM", "{}", user.Id)
	if err != nil {
		return nil, err
	}

	q = `update users set default_persona_id = (select id from personas where user_id = $1::uuid limit 1) where id = $1::uuid returning users.default_persona_id;`
	row = db.QueryRow(ctx, q, user.Id)
	if err != nil {
		return nil, err
	}

	persona := &struct {
		Id string
	}{}
	err = row.Scan(&persona.Id)
	if err != nil {
		return nil, err
	}

	q = `insert into files (id, entity_id, type, url, mime, size, version, deployment_type, platform, uploaded_by, width, height, created_at, updated_at, variation, original_path, hash)
    values (gen_random_uuid(), $1::uuid, 'mesh_avatar', $2::text, 'model/gltf-binary', 0, 0, '', '', $3::uuid, 0, 0, now(), null, 0, '', '')`
	_, err = db.Exec(ctx, q, persona.Id, "https://models.readyplayer.me/643902682a163b9bdfefe7b5.glb", user.Id)
	if err != nil {
		return nil, err
	}

	req := GetUserByIdRequest{Id: user.Id}
	return GetUserById(ctx, req)
}

type RegisterUserRequestFromOAuthWithId struct {
	Id string `json:"id" validate:"required,id"`
}

func RegisterUserFromOAuthWithId(ctx context.Context, request RegisterUserRequestFromOAuthWithId) (u *User, err error) {
	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	rand.Seed(time.Now().UnixNano())
	randomName := fmt.Sprintf("User %06d", rand.Intn(1000000-100000)+100000)

	q := `with r as (insert into entities (id, created_at, updated_at, entity_type, public, views)
    values ($1::uuid, now(), null, 'user', true, null)
    returning id)
insert
into users (id, email, name, is_email_confirmed, api_key, eth_address, is_active, is_admin, is_muted, is_banned, is_internal, allow_emails, experience)
select r.id, '', $2::text, false, $3::text, '', true, false, false, false, false, true, 0
from r
returning id;`

	randomKeyUuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	randomKey := strings.ReplaceAll(randomKeyUuid.String(), "-", "")

	row := db.QueryRow(ctx, q, request.Id, randomName, randomKey)
	user := &User{}
	err = row.Scan(&user.Id)
	if err != nil {
		return nil, err
	}

	q = `with r as (insert into entities (id, created_at, updated_at, entity_type, public, views) values (gen_random_uuid(), now(), null, 'persona', true, null) returning id)
insert into personas (id, name, type, configuration, user_id) select r.id, $1::text, $2::text, $3::jsonb, $4::uuid from r`
	_, err = db.Exec(ctx, q, randomName, "RPM", "{}", user.Id)
	if err != nil {
		return nil, err
	}

	q = `update users set default_persona_id = (select id from personas where user_id = $1::uuid limit 1) where id = $1::uuid returning users.default_persona_id;`
	row = db.QueryRow(ctx, q, user.Id)
	if err != nil {
		return nil, err
	}

	persona := &struct {
		Id string
	}{}
	err = row.Scan(&persona.Id)
	if err != nil {
		return nil, err
	}

	q = `insert into files (id, entity_id, type, url, mime, size, version, deployment_type, platform, uploaded_by, width, height, created_at, updated_at, variation, original_path, hash)
    values (gen_random_uuid(), $1::uuid, 'mesh_avatar', $2::text, 'model/gltf-binary', 0, 0, '', '', $3::uuid, 0, 0, now(), null, 0, '', '')`
	_, err = db.Exec(ctx, q, persona.Id, "https://models.readyplayer.me/643902682a163b9bdfefe7b5.glb", user.Id)
	if err != nil {
		return nil, err
	}

	req := GetUserByIdRequest{Id: user.Id}
	return GetUserById(ctx, req)
}

type RegisterUserRequestFromOAuthWithEthAddress struct {
	EthAddress string `json:"ethAddress" validate:"required"`
}

func RegisterUserFromOAuthWithEthAddress(ctx context.Context, request RegisterUserRequestFromOAuthWithEthAddress) (u *User, err error) {
	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	rand.Seed(time.Now().UnixNano())
	randomName := fmt.Sprintf("User %06d", rand.Intn(1000000-100000)+100000)

	q := `with r as (insert into entities (id, created_at, updated_at, entity_type, public, views)
	values (gen_random_uuid(), now(), null, 'user', true, null)
	returning id)
insert
into users (id, eth_address, name, is_address_confirmed, api_key, is_active, is_admin, is_muted, is_banned, is_internal, allow_emails, experience)
select r.id, $1::text, $2::text, true, $3::text, true, false, false, false, false, true, 0
from r
returning id;`

	randomKeyUuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	randomKey := strings.ReplaceAll(randomKeyUuid.String(), "-", "")

	row := db.QueryRow(ctx, q, request.EthAddress, randomName, randomKey)
	user := &User{}
	err = row.Scan(&user.Id)
	if err != nil {
		return nil, err
	}

	q = `with r as (insert into entities (id, created_at, updated_at, entity_type, public, views) values (gen_random_uuid(), now(), null, 'persona', true, null) returning id)
insert into personas (id, name, type, configuration, user_id) select r.id, $1::text, $2::text, $3::jsonb, $4::uuid from r`
	_, err = db.Exec(ctx, q, randomName, "RPM", "{}", user.Id)
	if err != nil {
		return nil, err
	}

	q = `update users set default_persona_id = (select id from personas where user_id = $1::uuid limit 1) where id = $1::uuid returning users.default_persona_id;`
	row = db.QueryRow(ctx, q, user.Id)
	if err != nil {
		return nil, err
	}

	persona := &struct {
		Id string
	}{}
	err = row.Scan(&persona.Id)
	if err != nil {
		return nil, err
	}

	q = `insert into files (id, entity_id, type, url, mime, size, version, deployment_type, platform, uploaded_by, width, height, created_at, updated_at, variation, original_path, hash)
    values (gen_random_uuid(), $1::uuid, 'mesh_avatar', $2::text, 'model/gltf-binary', 0, 0, '', '', $3::uuid, 0, 0, now(), null, 0, '', '')`
	_, err = db.Exec(ctx, q, persona.Id, "https://models.readyplayer.me/643902682a163b9bdfefe7b5.glb", user.Id)
	if err != nil {
		return nil, err
	}

	req := GetUserByIdRequest{Id: user.Id}
	return GetUserById(ctx, req)
}
