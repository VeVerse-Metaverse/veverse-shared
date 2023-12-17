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
	"strconv"
)

// Release struct
type Release struct {
	Entity

	AppId          *uuid.UUID `json:"appId,omitempty"`
	AppName        string     `json:"appName,omitempty"`
	AppTitle       string     `json:"appTitle,omitempty"`
	AppDescription *string    `json:"appDescription"`
	AppUrl         *string    `json:"appUrl"`
	AppExternal    *bool      `json:"appExternal"`
	Version        string     `json:"version,omitempty"`
	CodeVersion    string     `json:"codeVersion,omitempty"`
	ContentVersion string     `json:"contentVersion,omitempty"`
	Name           *string    `json:"name,omitempty"`
	Description    *string    `json:"description,omitempty"`
	Archive        *bool      `json:"archive"`
}

// ReleaseV2 release metadata, attached to an AppV2, LauncherV2 or SDK
type ReleaseV2 struct {
	Entity                    // Entity is the base struct for the ReleaseV2
	EntityId       *uuid.UUID `json:"entityId,omitempty"`       // parent Entity Identifier (required) (unique)
	Version        string     `json:"version,omitempty"`        // semantic Version of the Release (e.g. 1.0.0) (required) (unique)
	CodeVersion    string     `json:"codeVersion,omitempty"`    // semantic CodeVersion of the Release (e.g. 1.0.0) (required) (default: 1.0.0)
	ContentVersion string     `json:"contentVersion,omitempty"` // semantic ContentVersion of the Release (e.g. 1.0.0) (required) (default: 1.0.0)
	Name           *string    `json:"name,omitempty"`           // Name of the Release (optional) (default: "Release " + Version)
	Description    *string    `json:"description,omitempty"`    // Description of the Release (optional) (default: "")
	Archive        bool       `json:"archive"`                  // Release is distributed as an Archive instead of a list of separate files (optional) (default to false)
	App            *AppV2     `json:"app,omitempty"`            // AppV2 is the parent AppV2 of the ReleaseV2 (optional)
}

func (r ReleaseV2) String() string {
	var out = r.Entity.String()
	out += fmt.Sprintf("\"entityId\": \"%v\", ", r.EntityId)
	out += fmt.Sprintf("\"version\": \"%v\", ", r.Version)
	out += fmt.Sprintf("\"codeVersion\": \"%v\", ", r.CodeVersion)
	out += fmt.Sprintf("\"contentVersion\": \"%v\", ", r.ContentVersion)
	if r.Name != nil {
		out += fmt.Sprintf("\"name\": \"%v\", ", *r.Name)
	}
	if r.Description != nil {
		out += fmt.Sprintf("\"description\": \"%v\", ", *r.Description)
	}
	out += fmt.Sprintf("\"archive\": \"%v\", ", r.Archive)
	if r.App != nil {
		out += fmt.Sprintf("\"app\": {\n\t%v\n}, ", r.App.String())
	}
	return out
}

type ReleaseV2Batch Batch[ReleaseV2]

type IndexReleaseV2Request struct {
	Offset *int64  `json:"offset,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
	Search *string `json:"search,omitempty"`
}

func IndexReleaseV2(ctx context.Context, requester *User, request IndexReleaseV2Request) (entities *ReleaseV2Batch, err error) {
	// validate requester
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	// get the database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	var batch = ReleaseV2Batch{
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
		qt   string
		q    string
		rows pgx.Rows
	)

	if requester.IsAdmin {
		if request.Search != nil && *request.Search != "" {
			qt = `select count(*)
from release_v2
where name ilike $1::text`
			q = `select e.id,
       e.created_at,
       e.updated_at,
       e.entity_type,
       e.views,
       e.public,
       r.version,
       r.code_version,
       r.content_version,
       r.name,
       r.description,
       r.archive
from release_v2 r
         left join entities e on r.id = e.id
where r.name ilike $1::text
order by e.created_at desc
offset $2::int8 limit $3::int8`

			err = db.QueryRow(ctx, qt, fmt.Sprintf("%%%s%%", helper.SanitizeLikeClause(*request.Search))).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			rows, err = db.Query(ctx, q, fmt.Sprintf("%%%s%%", helper.SanitizeLikeClause(*request.Search)), batch.Offset, batch.Limit)
		} else {
			qt = `select count(*)
from release_v2`
			q = `select e.id,
       e.created_at,
       e.updated_at,
       e.entity_type,
       e.views,
       e.public,
       r.version,
       r.code_version,
       r.content_version,
       r.name,
       r.description,
       r.archive
from release_v2 r
         left join entities e on r.id = e.id
order by e.created_at desc
offset $1::int8 limit $2::int8`
		}

		err = db.QueryRow(ctx, qt).Scan(&batch.Total)
		if err != nil {
			return nil, err
		}

		rows, err = db.Query(ctx, q, batch.Offset, batch.Limit)
	} else {
		if request.Search != nil && *request.Search != "" {
			qt = `select count(*) from release_v2 r left join entities e on r.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1::uuid where e.public = true or a.user_id = $1::uuid or a.user_id is null and r.name ilike $2::text`
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, r.version, r.code_version, r.content_version, r.name, r.description, r.archive from release_v2 r left join entities e on r.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1::uuid where e.public = true or a.user_id = $1::uuid or a.user_id is null and r.name ilike $2::text order by e.created_at desc offset $3::int8 limit $4::int8`

			err = db.QueryRow(ctx, qt, requester.Id, fmt.Sprintf("%%%s%%", helper.SanitizeLikeClause(*request.Search))).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			rows, err = db.Query(ctx, q, requester.Id, fmt.Sprintf("%%%s%%", helper.SanitizeLikeClause(*request.Search)), batch.Offset, batch.Limit)
		} else {
			qt = `select count(*) from release_v2 r left join entities e on r.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1::uuid where e.public = true or a.user_id = $1::uuid or a.user_id is null`
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, r.version, r.code_version, r.content_version, r.name, r.description, r.archive from release_v2 r left join entities e on r.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1::uuid where e.public = true or a.user_id = $1::uuid or a.user_id is null order by e.created_at desc offset $2::int8 limit $3::int8`

			err = db.QueryRow(ctx, qt, requester.Id).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			rows, err = db.Query(ctx, q, requester.Id, batch.Offset, batch.Limit)
		}
	}
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			release ReleaseV2
			entity  Entity

			entityId        pgtypeuuid.UUID
			entityCreatedAt pgtype.Timestamptz
			entityUpdatedAt pgtype.Timestamptz
			entityType      pgtype.Text
			entityViews     pgtype.Int4
			entityPublic    pgtype.Bool
			version         pgtype.Text
			codeVersion     pgtype.Text
			contentVersion  pgtype.Text
			name            pgtype.Text
			description     pgtype.Text
			archive         pgtype.Bool
		)
		err = rows.Scan(&entityId, &entityCreatedAt, &entityUpdatedAt, &entityType, &entityViews, &entityPublic, &version, &codeVersion, &contentVersion, &name, &description, &archive)
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
			release.Entity = entity
			if version.Status == pgtype.Present {
				release.Version = version.String
			}
			if codeVersion.Status == pgtype.Present {
				release.CodeVersion = codeVersion.String
			}
			if contentVersion.Status == pgtype.Present {
				release.ContentVersion = contentVersion.String
			}
			if name.Status == pgtype.Present {
				release.Name = &name.String
			}
			if description.Status == pgtype.Present {
				release.Description = &description.String
			}
			if archive.Status == pgtype.Present {
				release.Archive = archive.Bool
			}
			if !ContainsIdentifiable(helper.ToSliceOfAny(batch.Entities), release.Id) {
				batch.Entities = append(batch.Entities, release)
			}
		}
	}

	return &batch, nil
}

func GetReleaseV2(ctx context.Context, requester *User, releaseId uuid.UUID) (release *ReleaseV2, err error) {
	if requester == nil {
		return nil, ErrNoRequester
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	var (
		q    string
		rows pgx.Rows
	)

	if requester.IsAdmin {
		q = `select r.id,
       e.created_at,
       e.updated_at,
       e.views,
       e.public,
       ou.id,
       ou.name,
       r.version,
       r.code_version,
       r.content_version,
       r.name,
       r.description,
       r.archive
from release_v2 r
         left join entities e on r.id = e.id
         left join accessibles oa on e.id = oa.entity_id
         left join users ou on oa.user_id = ou.id and oa.is_owner
where r.id = $1::uuid
order by e.created_at desc`
		rows, err = db.Query(ctx, q, releaseId)
	} else {
		q = `select r.id,
       e.created_at,
       e.updated_at,
       e.views,
       e.public,
       ou.id,
       ou.name,
       r.version,
       r.code_version,
       r.content_version,
       r.name,
       r.description,
       r.archive
from release_v2 r
         left join entities e on r.id = e.id
         left join accessibles oa on e.id = oa.entity_id
         left join users ou on oa.user_id = ou.id and oa.is_owner
         left join accessibles a on e.id = a.entity_id and a.user_id = $2::uuid
where r.id = $1::uuid
  and (e.public or a.is_owner or a.can_view)
order by e.created_at desc`
		rows, err = db.Query(ctx, q, releaseId, requester.Id)
	}

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var (
			id             pgtypeuuid.UUID
			createdAt      pgtype.Timestamptz
			updatedAt      pgtype.Timestamptz
			views          pgtype.Int4
			public         pgtype.Bool
			ownerId        pgtypeuuid.UUID
			ownerName      pgtype.Text
			version        pgtype.Text
			codeVersion    pgtype.Text
			contentVersion pgtype.Text
			name           pgtype.Text
			description    pgtype.Text
			archive        pgtype.Bool
		)

		err = rows.Scan(&id, &createdAt, &updatedAt, &views, &public, &ownerId, &ownerName, &version, &codeVersion, &contentVersion, &name, &description, &archive)
		if err != nil {
			return nil, err
		}

		if id.Status != pgtype.Present {
			continue
		}

		if release == nil {
			release = &ReleaseV2{}
			release.Id = id.UUID
			if createdAt.Status == pgtype.Present {
				release.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				release.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				release.Views = views.Int
			}
			if public.Status == pgtype.Present {
				release.Public = public.Bool
			}
			if ownerId.Status == pgtype.Present {
				release.Owner = &User{}
				release.Owner.Id = ownerId.UUID
				if ownerName.Status == pgtype.Present {
					release.Owner.Name = &ownerName.String
				}
			}
			if version.Status == pgtype.Present {
				release.Version = version.String
			}
			if codeVersion.Status == pgtype.Present {
				release.CodeVersion = codeVersion.String
			}
			if contentVersion.Status == pgtype.Present {
				release.ContentVersion = contentVersion.String
			}
			if name.Status == pgtype.Present {
				release.Name = &name.String
			}
			if description.Status == pgtype.Present {
				release.Description = &description.String
			}
			if archive.Status == pgtype.Present {
				release.Archive = archive.Bool
			}
		} else {
			if release.Owner == nil && ownerId.Status == pgtype.Present {
				release.Owner = &User{}
				release.Owner.Id = ownerId.UUID
				if ownerName.Status == pgtype.Present {
					release.Owner.Name = &ownerName.String
				}
			}
		}
	}

	return release, nil
}

type LatestReleaseRequestFileOptions struct {
	Platform string `json:"platform"`
	Target   string `json:"target"`
}

type LatestReleaseRequestOptions struct {
	Files       bool                             `json:"files"`
	FileOptions *LatestReleaseRequestFileOptions `json:"fileOptions"`
	Owner       bool                             `json:"owner"`
}

type GetLatestReleaseRequest struct {
	AppId   uuid.UUID
	Options *LatestReleaseRequestOptions `json:"options"`
}

func GetLatestReleaseV2Public(ctx context.Context, request GetLatestReleaseRequest) (release *ReleaseV2, err error) {
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	// validate request options
	if request.Options != nil {
		if request.Options.Files {
			if request.Options.FileOptions == nil {
				return nil, fmt.Errorf("missing file options")
			} else if request.Options.FileOptions.Platform == "" {
				return nil, fmt.Errorf("missing file options platform")
			} else if request.Options.FileOptions.Target == "" {
				return nil, fmt.Errorf("missing file options target")
			}
		}
	}

	var (
		q       string
		qArgs   = make([]any, 0)
		qArgNum = 0
		rows    pgx.Rows
	)

	// build query
	q = `select r.id, e.created_at, e.updated_at, e.views, e.public, r.version, r.code_version, r.content_version, r.name, r.description, r.archive` // 11
	if request.Options != nil {
		if request.Options.Owner {
			// add owner id and name
			q += `, u.id, u.name, u.description, u.eth_address, u.is_banned` // 16 (+5)
		}
		if request.Options.Files {
			// add release file columns
			q += `, f.id, f.entity_id, f.type, f.url, f.mime, f.size, f.version, f.deployment_type, f.platform, f.uploaded_by, f.created_at, f.updated_at, f.variation, f.original_path, f.hash` // 31 (+15)
		}
	}

	// query from
	q += ` from release_v2 r left join entities e on r.id = e.id`
	if request.Options != nil {
		if request.Options.Owner {
			q += ` left join accessibles a on e.id = a.entity_id and a.is_owner`
			q += ` left join users u on a.user_id = u.id`
		}
		if request.Options.Files {
			q += ` left join files f on e.id = f.entity_id and (f.type = 'release' or f.type = 'release-archive' or f.type = 'release-archive-sdk')`
		}
	}

	// query where
	qArgNum++
	qArgs = append(qArgs, request.AppId)
	q += ` where r.entity_id = $` + strconv.Itoa(qArgNum)
	q += ` and r.version = (select max(version) from release_v2 ri left join entities ei on ei.id = ri.id where ri.entity_id = $` + strconv.Itoa(qArgNum) + ` and ei.public)`

	if request.Options != nil {
		if request.Options.Files && request.Options.FileOptions != nil {
			qArgNum++
			qArgs = append(qArgs, request.Options.FileOptions.Target)
			q += ` and f.deployment_type = $` + strconv.Itoa(qArgNum)

			qArgNum++
			qArgs = append(qArgs, request.Options.FileOptions.Platform)
			q += ` and f.platform = $` + strconv.Itoa(qArgNum)
		}
	}

	rows, err = db.Query(ctx, q, qArgs...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			owner User
			file  *File
		)
		var (
			id                   pgtypeuuid.UUID
			createdAt, updatedAt pgtype.Timestamptz
			views                pgtype.Int4
			public               pgtype.Bool
			version              pgtype.Text
			codeVersion          pgtype.Text
			contentVersion       pgtype.Text
			name                 pgtype.Text
			description          pgtype.Text
			archive              pgtype.Bool
		)

		allFields := []interface{}{
			&id, &createdAt, &updatedAt, &views, &public, &version, &codeVersion, &contentVersion, &name, &description, &archive,
		}

		var (
			fileId             pgtypeuuid.UUID
			fileEntityId       pgtypeuuid.UUID
			fileType           pgtype.Text
			fileUrl            pgtype.Text
			fileMime           pgtype.Text
			fileSize           pgtype.Int8
			fileVersion        pgtype.Int8
			fileDeploymentType pgtype.Text
			filePlatform       pgtype.Text
			fileUploadedBy     pgtypeuuid.UUID
			fileCreatedAt      pgtype.Timestamptz
			fileUpdatedAt      pgtype.Timestamptz
			fileVariation      pgtype.Int8
			fileOriginalPath   pgtype.Text
			fileHash           pgtype.Text
		)
		fileFields := []interface{}{
			&fileId, &fileEntityId, &fileType, &fileUrl, &fileMime, &fileSize, &fileVersion, &fileDeploymentType, &filePlatform, &fileUploadedBy, &fileCreatedAt, &fileUpdatedAt, &fileVariation, &fileOriginalPath, &fileHash,
		}

		var (
			ownerId          pgtypeuuid.UUID
			ownerName        pgtype.Text
			ownerDescription pgtype.Text
			ownerEthAddress  pgtype.Text
			ownerIsBanned    pgtype.Bool
		)
		ownerFields := []interface{}{
			&ownerId, &ownerName, &ownerDescription, &ownerEthAddress, &ownerIsBanned,
		}

		if request.Options != nil {
			if request.Options.Owner {
				allFields = append(allFields, ownerFields...)
			}

			if request.Options.Files {
				allFields = append(allFields, fileFields...)
			}
		}

		err = rows.Scan(allFields...)
		if err != nil {
			return nil, err
		}

		if id.Status != pgtype.Present {
			continue
		}

		if fileId.Status == pgtype.Present {
			file = &File{}
			file.Id = fileId.UUID
			if fileType.Status == pgtype.Present {
				file.Type = fileType.String
			}
			if fileUrl.Status == pgtype.Present {
				file.Url = fileUrl.String
			}
			if fileMime.Status == pgtype.Present {
				file.Mime = &fileMime.String
			}
			if fileSize.Status == pgtype.Present {
				file.Size = &fileSize.Int
			}
			if fileVersion.Status == pgtype.Present {
				file.Version = fileVersion.Int
			}
			if fileDeploymentType.Status == pgtype.Present {
				file.Deployment = fileDeploymentType.String
			}
			if filePlatform.Status == pgtype.Present {
				file.Platform = filePlatform.String
			}
			if fileUploadedBy.Status == pgtype.Present {
				file.UploadedBy = &fileUploadedBy.UUID
			}
			if fileCreatedAt.Status == pgtype.Present {
				file.CreatedAt = fileCreatedAt.Time
			}
			if fileUpdatedAt.Status == pgtype.Present {
				file.UpdatedAt = &fileUpdatedAt.Time
			}
			if fileVariation.Status == pgtype.Present {
				file.Index = fileVariation.Int
			}
			if fileOriginalPath.Status == pgtype.Present {
				file.OriginalPath = &fileOriginalPath.String
			}
			if fileHash.Status == pgtype.Present {
				file.Hash = &fileHash.String
			}
		}

		if release == nil {
			release = &ReleaseV2{}
			release.Id = id.UUID
			if createdAt.Status == pgtype.Present {
				release.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				release.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				release.Views = views.Int
			}
			if public.Status == pgtype.Present {
				release.Public = public.Bool
			}
			if version.Status == pgtype.Present {
				release.Version = version.String
			}
			if codeVersion.Status == pgtype.Present {
				release.CodeVersion = codeVersion.String
			}
			if contentVersion.Status == pgtype.Present {
				release.ContentVersion = contentVersion.String
			}
			if name.Status == pgtype.Present {
				release.Name = &name.String
			}
			if description.Status == pgtype.Present {
				release.Description = &description.String
			}
			if archive.Status == pgtype.Present {
				release.Archive = archive.Bool
			}
			if ownerId.Status == pgtype.Present {
				owner.Id = ownerId.UUID
				if ownerName.Status == pgtype.Present {
					owner.Name = &ownerName.String
				} else {
					unknownName := "Unknown"
					owner.Name = &unknownName
				}
				if ownerDescription.Status == pgtype.Present {
					owner.Description = &ownerDescription.String
				}
				if ownerEthAddress.Status == pgtype.Present {
					owner.EthAddress = &ownerEthAddress.String
				}
				if ownerIsBanned.Status == pgtype.Present {
					owner.IsBanned = ownerIsBanned.Bool
				}
				release.Owner = &owner
			}
			if file != nil {
				release.Files = &FileBatch{}
				release.Files.Entities = append(release.Files.Entities, *file)
			}
		} else {
			if release.Files == nil {
				release.Files = &FileBatch{}
			}
			if file != nil && !ContainsIdentifiable(helper.ToSliceOfAny(release.Files.Entities), file.Id) {
				release.Files.Entities = append(release.Files.Entities, *file)
			}
			if release.Owner == nil && ownerId.Status == pgtype.Present {
				owner.Id = ownerId.UUID
				if ownerName.Status == pgtype.Present {
					owner.Name = &ownerName.String
				} else {
					unknownName := "Unknown"
					owner.Name = &unknownName
				}
				if ownerDescription.Status == pgtype.Present {
					owner.Description = &ownerDescription.String
				}
				if ownerEthAddress.Status == pgtype.Present {
					owner.EthAddress = &ownerEthAddress.String
				}
				if ownerIsBanned.Status == pgtype.Present {
					owner.IsBanned = ownerIsBanned.Bool
				}
				release.Owner = &owner
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return release, nil
}
