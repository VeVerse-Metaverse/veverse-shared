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
	"strings"
)

// AppV2 application metadata
type AppV2 struct {
	Entity
	Name        string          `json:"name,omitempty"`        // name of the application (required)
	Description *string         `json:"description,omitempty"` // description of the application (optional)
	External    bool            `json:"external"`              // application is external (not VeVerse based) (default to false)
	SDK         *SDK            `json:"sdk,omitempty"`         // SDK used by the application (optional)
	Releases    *ReleaseV2Batch `json:"releases,omitempty"`    // list of releases for the application (required)
}

type AppV2Batch Batch[AppV2]

func (a *AppV2) String() string {
	var out = a.Entity.String()
	out += fmt.Sprintf("name: %v, ", a.Name)
	if a.Description != nil {
		out += fmt.Sprintf("description: %v, ", *a.Description)
	}
	out += fmt.Sprintf("external: %v, ", a.External)
	if a.SDK != nil {
		out += fmt.Sprintf("sdk: %v, ", a.SDK.String())
	}
	if a.Releases != nil {
		out += fmt.Sprintf("releases:\n\t%v, ", (*Batch[ReleaseV2])(a.Releases).String())
	}
	return out
}

func (a *AppV2) InitReleases() {
	a.Releases = &ReleaseV2Batch{}
}

type GetEntityImageRequest struct {
	Id   string
	Type string
}

func GetEntityImage(ctx context.Context, requester *User, request GetEntityImageRequest) (entity *File, err error) {
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	if request.Id == "" {
		return nil, fmt.Errorf("entity id is not set")
	}

	const imageLabel = "image"
	if request.Type == "" && (strings.HasPrefix(request.Type, imageLabel) || strings.HasSuffix(request.Type, imageLabel)) {
		return nil, fmt.Errorf("image type is invalid")
	}

	var q = `select f.id,
       f.entity_id,
       f.type,
       f.url,
       f.mime,
       f.size,
       f.version,
       f.deployment_type,
       f.platform,
       f.uploaded_by,
       f.width,
       f.height,
       f.created_at,
       f.updated_at,
       f.variation,
       f.original_path,
       f.hash
from files f
where entity_id = $1
  and type = $2
order by created_at desc
limit 1`

	var f = &File{}
	err = db.QueryRow(ctx, q, request.Id, request.Type).Scan(&f.Id, &f.EntityId, &f.Type, &f.Url, &f.Mime, &f.Size, &f.Version, &f.Deployment, &f.Platform, &f.UploadedBy, &f.Width, &f.Height, &f.CreatedAt, &f.UpdatedAt, &f.Index, &f.OriginalPath, &f.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get app image: %w", err)
	}

	return f, nil
}

type GetAppLogoRequest struct {
	Id string
}

func GetAppIdentityFiles(ctx context.Context, request GetAppLogoRequest) (entity []File, err error) {
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	if request.Id == "" {
		return nil, fmt.Errorf("entity id is not set")
	}

	var q = `select f.id,
       f.entity_id,
       f.type,
       f.url,
       f.mime,
       f.size,
       f.version,
       f.deployment_type,
       f.platform,
       f.uploaded_by,
       f.width,
       f.height,
       f.created_at,
       f.updated_at,
       f.variation,
       f.original_path,
       f.hash
from files f
         left join apps a on f.entity_id = a.id
where f.entity_id = $1 and (f.type = 'image-app-logo' or f.type = 'image-app-placeholder')
order by f.created_at desc`

	var (
		rows  pgx.Rows
		f     = File{}
		files = make([]File, 0)
	)
	rows, err = db.Query(ctx, q, request.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get app image: %w", err)
	}

	for rows.Next() {
		err = rows.Scan(&f.Id, &f.EntityId, &f.Type, &f.Url, &f.Mime, &f.Size, &f.Version, &f.Deployment, &f.Platform, &f.UploadedBy, &f.Width, &f.Height, &f.CreatedAt, &f.UpdatedAt, &f.Index, &f.OriginalPath, &f.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get app image: %w", err)
		}
		files = append(files, f)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get app image: %w", err)
	}

	return files, nil
}

type CreateAppV2Request struct {
	Name        string  `json:"name,omitempty"`        // name of the application (required)
	Description *string `json:"description,omitempty"` // description of the application (optional)
}

func CreateAppV2(ctx context.Context, requester *User, request CreateAppV2Request) (app *AppV2, err error) {
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	if request.Name == "" {
		return nil, fmt.Errorf("name is not set")
	}

	q := `with r as (insert into entities (id, entity_type, public) values (gen_random_uuid(), 'app-v2', true) returning id)
insert into app_v2 (id, name, description)
select r.id, $1, $2 from r
returning id`

	row := db.QueryRow(ctx, q, request.Name, request.Description)
	app = &AppV2{}
	err = row.Scan(&app.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return GetAppV2(ctx, requester, app.Id, "")
}

type IndexAppV2Request struct {
	Offset *int64  `json:"offset,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
	Search *string `json:"search,omitempty"`
}

func IndexAppV2(ctx context.Context, requester *User, request IndexAppV2Request) (entities *AppV2Batch, err error) {
	// validate requester
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	// get database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	// response data
	var batch = AppV2Batch{
		Offset: 0,
		Limit:  100,
		Total:  0,
	}

	// set offset if it is in a valid range
	if request.Offset != nil && *request.Offset >= 0 {
		batch.Offset = *request.Offset
	}

	// set limit if it is in a valid range
	if request.Limit != nil && *request.Limit >= 0 && *request.Limit <= 100 {
		batch.Limit = *request.Limit
	}

	var (
		qt   string   // query string
		q    string   // query string
		rows pgx.Rows // rows from the query
	)

	if requester.IsAdmin {
		// admin can see all launchers
		if request.Search != nil && *request.Search != "" {
			// total count query
			qt = `select count(*) from app_v2 l where l.name ilike '%' || $1 || '%'`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, a.name
from app_v2 a
         left join entities e on a.id = e.id and e.entity_type = 'app-v2'
where a.name ilike '%' || $1::text || '%'
order by e.created_at desc
offset $2 limit $3`

			// get total count
			err = db.QueryRow(ctx, qt, helper.SanitizeLikeClause(*request.Search)).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			// execute query
			rows, err = db.Query(ctx, q, request.Search, request.Offset, request.Limit)
		} else {
			// total count query
			qt = `select count(*) from app_v2`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, a.name
from app_v2 a
         left join entities e on a.id = e.id and e.entity_type = 'app-v2'
order by e.created_at desc
offset $1 limit $2`

			// get total count
			err = db.QueryRow(ctx, qt).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			// execute query
			rows, err = db.Query(ctx, q, request.Offset, request.Limit)
		}
		if err != nil {
			return nil, err
		}
	} else {
		// non-admin can only see launchers they have access to
		if request.Search != nil && *request.Search != "" {
			// total count query
			qt = `select count(*) from app_v2 l left join entities e on l.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1 where e.public or (a.is_owner or a.can_view) and l.name ilike '%' || $2 || '%'`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, l.name
from app_v2 l
			 left join entities e on l.id = e.id and e.entity_type = 'app-v2'
			 left join accessibles a on e.id = a.entity_id and a.user_id = $1::uuid
where e.public or (a.is_owner or a.can_view) and l.name ilike '%' || $2::text || '%'
order by e.created_at desc
offset $3 limit $4`

			// get total count
			err = db.QueryRow(ctx, qt, requester.Id, helper.SanitizeLikeClause(*request.Search)).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			// execute query
			rows, err = db.Query(ctx, q, requester.Id, request.Search, request.Offset, request.Limit)
		} else {
			// total count query
			qt = `select count(*) from app_v2 l left join entities e on l.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1 where e.public or (a.is_owner or a.can_view)`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, l.name
from app_v2 l
         left join entities e on l.id = e.id
         left join accessibles a on e.id = a.entity_id and a.user_id = $1::uuid
where e.public or (a.is_owner or a.can_view)
order by e.created_at desc
offset $2 limit $3`

			// get total count
			err = db.QueryRow(ctx, qt, requester.Id).Scan(&batch.Total)
			if err != nil {
				return nil, err
			}

			// execute query
			rows, err = db.Query(ctx, q, requester.Id, request.Offset, request.Limit)
		}
		if err != nil {
			return nil, err
		}
	}

	defer rows.Close()
	for rows.Next() {
		var (
			app    AppV2
			entity Entity

			entityId   pgtypeuuid.UUID
			entityType pgtype.Text
			createdAt  pgtype.Timestamp
			updatedAt  pgtype.Timestamp
			views      pgtype.Int4
			public     pgtype.Bool
			name       pgtype.Text
		)
		err = rows.Scan(&entityId, &createdAt, &updatedAt, &entityType, &views, &public, &name)
		if err != nil {
			return nil, err
		}

		// parse entity data
		if entityId.Status == pgtype.Present {
			entity.Id = entityId.UUID
			if entityType.Status == pgtype.Present {
				entity.EntityType = entityType.String
			}
			if createdAt.Status == pgtype.Present {
				entity.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				entity.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				entity.Views = views.Int
			}
			if public.Status == pgtype.Present {
				entity.Public = public.Bool
			}
			app.Entity = entity
			if name.Status == pgtype.Present {
				app.Name = name.String
			}
			if !ContainsIdentifiable(helper.ToSliceOfAny(batch.Entities), app.Id) {
				batch.Entities = append(batch.Entities, app)
			}
		}
	}

	return &batch, nil
}

// GetAppV2 gets an app metadata by its id, does not include apps and releases
func GetAppV2(ctx context.Context, requester *User, id uuid.UUID, platform string) (app *AppV2, err error) {
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
		q = `SELECT a.id,
       a.name,
       a.description,
       a.external,
       e.created_at,
       e.updated_at,
       e.views,
       ou.id,
       ou.name,
       f.id,
       f.created_at,
       f.url,
       f.type,
       f.mime,
       f.original_path,
       f.size,
       r.id,
       re.created_at,
       re.updated_at,
       re.views,
       r.version,
       r.name,
       r.description,
       r.code_version,
       r.content_version,
       r.archive,
       rf.id,
       rf.created_at,
       rf.url,
       rf.type,
       rf.mime,
       rf.original_path,
       rf.size
FROM app_v2 a
         LEFT JOIN entities e
                   ON a.id = e.id -- join entities to check public flag
         LEFT JOIN accessibles oa
                   ON e.id = oa.entity_id -- join accessibles to get owner
         LEFT JOIN users ou
                   ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
         LEFT JOIN files f
                   ON e.id = f.entity_id AND (f.platform = $2::text OR f.platform = '') -- join files
         LEFT JOIN release_v2 r
                   ON a.id = r.entity_id AND r.version = (SELECT max(r1.version)
                                                          FROM release_v2 r1
                                                                   LEFT JOIN entities e1 ON r1.id = e1.id
                                                          WHERE r1.entity_id = $1::uuid
                                                            AND e1.public = true) -- join latest release
         LEFT JOIN entities re ON r.id = re.id -- join latest release entity
         LEFT JOIN files rf
                   ON r.id = rf.entity_id AND (rf.platform = $2 :: text OR rf.platform = '') -- join release files
WHERE e.id = $1 :: uuid
ORDER BY e.created_at DESC, re.created_at DESC;`
		rows, err = db.Query(ctx, q, id, platform)
	} else {
		q = `SELECT a.id,
       a.name,
       a.description,
       a.external,
       e.created_at,
       e.updated_at,
       e.views,
       ou.id,
       ou.name,
       f.id,
       f.created_at,
       f.url,
       f.type,
       f.mime,
       f.original_path,
       f.size,
       r.id,
       re.created_at,
       re.updated_at,
       re.views,
       r.version,
       r.name,
       r.description,
       r.code_version,
       r.content_version,
       r.archive,
       rf.id,
       rf.created_at,
       rf.url,
       rf.type,
       rf.mime,
       rf.original_path,
       rf.size
FROM app_v2 a
         LEFT JOIN entities e
                   ON a.id = e.id -- join entities to check public flag
         LEFT JOIN accessibles oa
                   ON e.id = oa.entity_id -- join accessibles to get owner
         LEFT JOIN users ou
                   ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
         LEFT JOIN accessibles aa
                   ON aa.entity_id = e.id AND aa.user_id = $3::uuid -- join accessibles to check access
         LEFT JOIN files f
                   ON e.id = f.entity_id AND (f.platform = $2::text OR f.platform = '') -- join files
         LEFT JOIN release_v2 r
                   ON a.id = r.entity_id AND r.version = (SELECT max(r1.version)
                                                          FROM release_v2 r1
                                                                   LEFT JOIN entities e1 
                                                                       ON r1.id = e1.id
                                                          WHERE r1.entity_id = $1::uuid
                                                            AND e1.public = true) -- join latest release
         LEFT JOIN entities re ON r.id = re.id -- join latest release entity
         LEFT JOIN files rf
                   ON r.id = rf.entity_id AND (rf.platform = $2::text OR rf.platform = '') -- join release files
WHERE e.id = $1::uuid
  AND (e.public OR aa.is_owner OR aa.can_view)
ORDER BY e.created_at DESC`
		rows, err = db.Query(ctx, q, id, platform, requester.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get launcher: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id                    pgtypeuuid.UUID
			name                  pgtype.Text
			description           pgtype.Text
			external              pgtype.Bool
			createdAt             pgtype.Timestamptz
			updatedAt             pgtype.Timestamptz
			views                 pgtype.Int4
			ownerId               pgtypeuuid.UUID
			ownerName             pgtype.Text
			fileId                pgtypeuuid.UUID
			fileCreatedAt         pgtype.Timestamptz
			fileUrl               pgtype.Text
			fileType              pgtype.Text
			fileMime              pgtype.Text
			filePath              pgtype.Text
			fileSize              pgtype.Int8
			releaseId             pgtypeuuid.UUID
			releaseCreatedAt      pgtype.Timestamptz
			releaseUpdatedAt      pgtype.Timestamptz
			releaseViews          pgtype.Int4
			releaseVersion        pgtype.Text
			releaseName           pgtype.Text
			releaseDescription    pgtype.Text
			releaseCodeVersion    pgtype.Text
			releaseContentVersion pgtype.Text
			releaseArchive        pgtype.Bool
			releaseFileId         pgtypeuuid.UUID
			releaseFileCreatedAt  pgtype.Timestamptz
			releaseFileUrl        pgtype.Text
			releaseFileType       pgtype.Text
			releaseFileMime       pgtype.Text
			releaseFilePath       pgtype.Text
			releaseFileSize       pgtype.Int8
		)

		err = rows.Scan(&id, &name, &description, &external, &createdAt, &updatedAt, &views, &ownerId, &ownerName, &fileId, &fileCreatedAt, &fileUrl, &fileType, &fileMime, &filePath, &fileSize, &releaseId, &releaseCreatedAt, &releaseUpdatedAt, &releaseViews, &releaseVersion, &releaseName, &releaseDescription, &releaseCodeVersion, &releaseContentVersion, &releaseArchive, &releaseFileId, &releaseFileCreatedAt, &releaseFileUrl, &releaseFileType, &releaseFileMime, &releaseFilePath, &releaseFileSize)
		if err != nil {
			return nil, fmt.Errorf("failed to scan launcher: %w", err)
		}

		if id.Status != pgtype.Present {
			continue
		}

		var file *File // file is optional and some rows may not have a file, or it can be repeated across rows
		if fileId.Status == pgtype.Present {
			file = &File{}
			file.Id = fileId.UUID
			if fileCreatedAt.Status == pgtype.Present {
				file.CreatedAt = fileCreatedAt.Time
			}
			if fileUrl.Status == pgtype.Present {
				file.Url = fileUrl.String
			}
			if fileType.Status == pgtype.Present {
				file.Type = fileType.String
			}
			if fileMime.Status == pgtype.Present {
				file.Mime = &fileMime.String
			}
			if filePath.Status == pgtype.Present {
				file.OriginalPath = &filePath.String
			}
			if fileSize.Status == pgtype.Present {
				file.Size = &fileSize.Int
			}
		}

		var release *ReleaseV2
		if releaseId.Status == pgtype.Present {
			release = &ReleaseV2{}
			release.InitFiles()
			release.Id = releaseId.UUID
			if releaseCreatedAt.Status == pgtype.Present {
				release.CreatedAt = releaseCreatedAt.Time
			}
			if releaseUpdatedAt.Status == pgtype.Present {
				release.UpdatedAt = &releaseUpdatedAt.Time
			}
			if releaseViews.Status == pgtype.Present {
				release.Views = releaseViews.Int
			}
			if releaseVersion.Status == pgtype.Present {
				release.Version = releaseVersion.String
			}
			if releaseName.Status == pgtype.Present {
				release.Name = &releaseName.String
			}
			if releaseDescription.Status == pgtype.Present {
				release.Description = &releaseDescription.String
			}
			if releaseCodeVersion.Status == pgtype.Present {
				release.CodeVersion = releaseCodeVersion.String
			}
			if releaseContentVersion.Status == pgtype.Present {
				release.ContentVersion = releaseContentVersion.String
			}
			if releaseArchive.Status == pgtype.Present {
				release.Archive = releaseArchive.Bool
			}
		}

		var releaseFile *File // file is optional and some rows may not have a file, or it can be repeated across rows
		if releaseFileId.Status == pgtype.Present {
			releaseFile = &File{}
			releaseFile.Id = releaseFileId.UUID
			if releaseFileCreatedAt.Status == pgtype.Present {
				releaseFile.CreatedAt = releaseFileCreatedAt.Time
			}
			if releaseFileUrl.Status == pgtype.Present {
				releaseFile.Url = releaseFileUrl.String
			}
			if releaseFileType.Status == pgtype.Present {
				releaseFile.Type = releaseFileType.String
			}
			if releaseFileMime.Status == pgtype.Present {
				releaseFile.Mime = &releaseFileMime.String
			}
			if releaseFilePath.Status == pgtype.Present {
				releaseFile.OriginalPath = &releaseFilePath.String
			}
			if releaseFileSize.Status == pgtype.Present {
				releaseFile.Size = &releaseFileSize.Int
			}
		}

		if app == nil { // first row
			app = &AppV2{}
			app.InitFiles()
			app.InitReleases()
			app.Id = id.UUID
			if ownerId.Status == pgtype.Present {
				app.Owner = &User{
					Entity: Entity{
						Identifier: Identifier{Id: ownerId.UUID},
					},
				}
				if ownerName.Status == pgtype.Present {
					app.Owner.Name = &ownerName.String
				}
			}
			if name.Status == pgtype.Present {
				app.Name = name.String
			}
			if description.Status == pgtype.Present {
				app.Description = &description.String
			}
			if external.Status == pgtype.Present {
				app.External = external.Bool
			}
			if createdAt.Status == pgtype.Present {
				app.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				app.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				app.Views = views.Int
			}
			if file != nil {
				app.Files.Entities = append(app.Files.Entities, *file)
			}
			if release != nil {
				if releaseFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(release.Files.Entities), releaseFile.Id) {
					release.Files.Entities = append(release.Files.Entities, *releaseFile)
				}
				app.Releases.Entities = append(app.Releases.Entities, *release)
			}
		} else { // other rows
			if app.Owner == nil { // if owner is not set yet, try to set it
				if ownerId.Status == pgtype.Present {
					app.Owner = &User{
						Entity: Entity{
							Identifier: Identifier{Id: ownerId.UUID},
						},
					}
					if ownerName.Status == pgtype.Present {
						app.Owner.Name = &ownerName.String
					}
				}
			}
			if release != nil { // if the latest release is not set yet, try to set it
				if releaseFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(release.Files.Entities), releaseFile.Id) {
					release.Files.Entities = append(release.Files.Entities, *releaseFile)
				}
				// append release to releases if it is not already there
				if !ContainsIdentifiable(helper.ToSliceOfAny(app.Releases.Entities), release.Id) {
					app.Releases.Entities = append(app.Releases.Entities, *release)
				}
			}
			if file != nil && !ContainsIdentifiable(helper.ToSliceOfAny(app.Files.Entities), file.Id) {
				app.Files.Entities = append(app.Files.Entities, *file)
			}
		}
	}

	return app, nil
}
