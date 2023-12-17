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
)

// LauncherV2 launcher metadata describes the launcher application, can have multiple apps with different releases
type LauncherV2 struct {
	Entity                      // Entity is the base struct for the launcher v2
	Name        string          `json:"name"`               // name of the launcher (file name)
	Description string          `json:"description"`        // launcher description ()
	Releases    *ReleaseV2Batch `json:"releases,omitempty"` // list of releases for the launcher (required)
	Apps        *AppV2Batch     `json:"apps,omitempty"`     // supported apps that are available in the launcher (required)
}
type LauncherV2Batch Batch[LauncherV2]

func (e *LauncherV2) InitReleases() {
	e.Releases = &ReleaseV2Batch{}
}

func (e *LauncherV2) String() string {
	var out = e.Entity.String()
	out += fmt.Sprintf("\"name\": \"%v\", ", e.Name)
	out += fmt.Sprintf("\"description\": \"%v\", ", e.Description)
	if e.Releases != nil {
		out += fmt.Sprintf("\"releases\":\n\t{%v\n}, ", (*Batch[ReleaseV2])(e.Releases).String())
	}
	if e.Apps != nil {
		out += fmt.Sprintf("\"apps\":\n\t{%v\n}, ", (*Batch[AppV2])(e.Apps).String())
	}
	return out
}

type IndexLauncherV2Request struct {
	Offset *int64  `json:"offset,omitempty"`
	Limit  *int64  `json:"limit,omitempty"`
	Search *string `json:"search,omitempty"`
}

func IndexLauncherV2(ctx context.Context, requester *User, request IndexLauncherV2Request) (entities *LauncherV2Batch, err error) {
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
	var batch = LauncherV2Batch{
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
			qt = `select count(*) from launcher_v2 l where l.name ilike '%' || $1 || '%'`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, l.name
from launcher_v2 l
         left join entities e on l.id = e.id and e.entity_type = 'launcher-v2'
where l.name ilike '%' || $1::text || '%'
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
			qt = `select count(*) from launcher_v2`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, l.name
from launcher_v2 l
         left join entities e on l.id = e.id and e.entity_type = 'launcher-v2'
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
			qt = `select count(*) from launcher_v2 l left join entities e on l.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1 where e.public or (a.is_owner or a.can_view) and l.name ilike '%' || $2 || '%'`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, l.name
from launcher_v2 l
			 left join entities e on l.id = e.id and e.entity_type = 'launcher-v2'
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
			qt = `select count(*) from launcher_v2 l left join entities e on l.id = e.id left join accessibles a on e.id = a.entity_id and a.user_id = $1 where e.public or (a.is_owner or a.can_view)`
			// query
			q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, l.name
from launcher_v2 l
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
			launcher LauncherV2
			entity   Entity

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
			launcher.Entity = entity
			if name.Status == pgtype.Present {
				launcher.Name = name.String
			}
			if !ContainsIdentifiable(helper.ToSliceOfAny(batch.Entities), launcher.Id) {
				batch.Entities = append(batch.Entities, launcher)
			}
		}
	}

	return &batch, nil
}

// GetLauncherV2 gets a launcher metadata by its id, does not include apps and releases
func GetLauncherV2(ctx context.Context, requester *User, launcherId uuid.UUID, platform string) (launcher *LauncherV2, err error) {
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
		q = `SELECT l.id,
       l.name,
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
       rf.platform,
       rf.mime,
       rf.original_path,
       rf.size
FROM launcher_v2 l
         LEFT JOIN entities e
                   ON l.id = e.id -- join entities to check public flag
         LEFT JOIN accessibles oa
                   ON e.id = oa.entity_id -- join accessibles to get owner
         LEFT JOIN users ou
                   ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
         left join files f -- join files
                   on e.id = f.entity_id and
                      case
                          when $2::text is not null and $2::text != ''
                              then (f.platform = $2::text OR f.platform = '')
                          else true
                          end
         LEFT JOIN release_v2 r
                   ON l.id = r.entity_id AND r.version = (SELECT max(r1.version)
                                                          FROM release_v2 r1
                                                                   LEFT JOIN entities e1 ON r1.id = e1.id
                                                          WHERE r1.entity_id = $1 :: uuid
                                                            AND e1.public = true) -- join latest release
         LEFT JOIN entities re ON r.id = re.id -- join latest release entity
         left join files rf -- join release files
                   on r.id = rf.entity_id and
                      case
                          when $2::text is not null and $2::text != ''
                              then (rf.platform = $2::text OR rf.platform = '')
                          else true
                          end
WHERE e.id = $1::uuid
ORDER BY e.created_at DESC, re.created_at DESC;`
		rows, err = db.Query(ctx, q, launcherId, platform)
	} else {
		q = `SELECT l.id,
       l.name,
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
       rf.platform,
       rf.mime,
       rf.original_path,
       rf.size
FROM launcher_v2 l
         LEFT JOIN entities e
                   ON l.id = e.id -- join entities to check public flag
         LEFT JOIN accessibles oa
                   ON e.id = oa.entity_id -- join accessibles to get owner
         LEFT JOIN users ou
                   ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
         LEFT JOIN accessibles a
                   ON a.entity_id = e.id AND a.user_id = $3::uuid -- join accessibles to check access
         left join files f -- join files
                   on e.id = f.entity_id and
                      case
                          when $2::text is not null and $2::text != ''
                              then (f.platform = $2::text OR f.platform = '')
                          else true
                          end
         LEFT JOIN release_v2 r
                   ON l.id = r.entity_id AND r.version = (SELECT max(r1.version)
                                                          FROM release_v2 r1
                                                                   LEFT JOIN entities e1
                                                                             ON r1.id = e1.id
                                                          WHERE r1.entity_id = $1::uuid
                                                            AND e1.public = true) -- join latest release
         LEFT JOIN entities re ON r.id = re.id -- join latest release entity
         left join files rf -- join release files
                   on r.id = rf.entity_id and
                      case
                          when $2::text is not null and $2::text != ''
                              then (rf.platform = $2::text OR rf.platform = '')
                          else true
                          end
WHERE e.id = $1::uuid
  AND (e.public OR a.is_owner OR a.can_view)
ORDER BY e.created_at DESC`
		rows, err = db.Query(ctx, q, launcherId, platform, requester.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get launcher: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id                    pgtypeuuid.UUID
			name                  pgtype.Text
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
			releaseFilePlatform   pgtype.Text
			releaseFileMime       pgtype.Text
			releaseFilePath       pgtype.Text
			releaseFileSize       pgtype.Int8
		)

		err = rows.Scan(&id, &name, &createdAt, &updatedAt, &views, &ownerId, &ownerName, &fileId, &fileCreatedAt, &fileUrl, &fileType, &fileMime, &filePath, &fileSize, &releaseId, &releaseCreatedAt, &releaseUpdatedAt, &releaseViews, &releaseVersion, &releaseName, &releaseDescription, &releaseCodeVersion, &releaseContentVersion, &releaseArchive, &releaseFileId, &releaseFileCreatedAt, &releaseFileUrl, &releaseFileType, &releaseFilePlatform, &releaseFileMime, &releaseFilePath, &releaseFileSize)
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
			if releaseFilePlatform.Status == pgtype.Present {
				releaseFile.Platform = releaseFilePlatform.String
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

		if launcher == nil { // first row
			launcher = &LauncherV2{}
			launcher.InitFiles()
			launcher.InitReleases()
			launcher.Id = id.UUID
			if ownerId.Status == pgtype.Present {
				launcher.Owner = &User{
					Entity: Entity{
						Identifier: Identifier{Id: ownerId.UUID},
					},
				}
				if ownerName.Status == pgtype.Present {
					launcher.Owner.Name = &ownerName.String
				}
			}
			if name.Status == pgtype.Present {
				launcher.Name = name.String
			}
			if createdAt.Status == pgtype.Present {
				launcher.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				launcher.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				launcher.Views = views.Int
			}
			if file != nil {
				launcher.Files.Entities = append(launcher.Files.Entities, *file)
			}
			if release != nil {
				if releaseFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(release.Files.Entities), releaseFile.Id) {
					release.Files.Entities = append(release.Files.Entities, *releaseFile)
				}
				launcher.Releases.Entities = append(launcher.Releases.Entities, *release)
			}
		} else { // other rows
			if launcher.Owner == nil { // if owner is not set yet, try to set it
				if ownerId.Status == pgtype.Present {
					launcher.Owner = &User{
						Entity: Entity{
							Identifier: Identifier{Id: ownerId.UUID},
						},
					}
					if ownerName.Status == pgtype.Present {
						launcher.Owner.Name = &ownerName.String
					}
				}
			}
			if release != nil { // if the latest release is not set yet, try to set it
				if releaseFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(release.Files.Entities), releaseFile.Id) {
					release.Files.Entities = append(release.Files.Entities, *releaseFile)
				}
				// append release to releases if it is not already there
				if !ContainsIdentifiable(helper.ToSliceOfAny(launcher.Releases.Entities), release.Id) {
					launcher.Releases.Entities = append(launcher.Releases.Entities, *release)
				}
			}
			if file != nil && !ContainsIdentifiable(helper.ToSliceOfAny(launcher.Files.Entities), file.Id) {
				launcher.Files.Entities = append(launcher.Files.Entities, *file)
			}
		}
	}

	return launcher, nil
}

type CreateLauncherV2Request struct {
	Name string `json:"name"`
}

func CreateLauncherV2(ctx context.Context, requester *User, request CreateLauncherV2Request) (launcher *LauncherV2, err error) {
	if requester == nil {
		return nil, ErrNoRequester
	}

	if !requester.IsAdmin {
		return nil, ErrNoPermission
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	q := `with r
         as (insert into entities (id, created_at, updated_at, entity_type, public, views) values (gen_random_uuid(), now(), null, 'launcher-v2', true, null) returning id)
insert
into launcher_v2 (id, name)
select r.id, $1
from r returning id;`

	row := db.QueryRow(ctx, q, request.Name)
	launcher = &LauncherV2{}
	err = row.Scan(&launcher.Id)
	if err != nil {
		return nil, err
	}

	return GetLauncherV2(ctx, requester, launcher.Id, "")
}

func UpdateLauncherV2(ctx context.Context, requester *User, id uuid.UUID, name string) (launcher *LauncherV2, err error) {
	if requester == nil {
		return nil, ErrNoRequester
	}

	canEdit, err := RequestCanEditEntity(ctx, requester, id)
	if err != nil {
		return nil, err
	}

	if !canEdit {
		return nil, ErrNoPermission
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	var (
		q string
	)

	q = `UPDATE launcher_v2
SET name = $1
WHERE id = $2`

	_, err = db.Exec(ctx, q, name, id)
	if err != nil {
		return nil, err
	}

	return GetLauncherV2(ctx, requester, id, "")
}

// IndexLauncherV2Releases returns all releases of the launcher itself
func IndexLauncherV2Releases(ctx context.Context, requester *User, launcherId uuid.UUID, platform string, offset int64, limit int64) (entities []ReleaseV2, err error) {
	if requester == nil {
		return nil, ErrNoRequester
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}
	var (
		q               string
		rows            pgx.Rows
		rowIndex        int64 = 0
		entityIndex     int64 = 0
		skipEntity            = false
		skippedEntityId uuid.UUID
	)

	if requester.IsAdmin {
		q = `SELECT r.id,
       r.name,
       r.description,
       r.version,
       r.entity_id,
       re.created_at,
       re.updated_at,
       re.views,
       ou.id,
       ou.name,
       f.id,
       f.created_at,
       f.updated_at,
       f.url,
       f.type,
       f.platform,
       f.mime,
       f.original_path,
       f.size
FROM release_v2 r
         LEFT JOIN entities le ON r.entity_id = le.id AND le.entity_type = 'launcher-v2'
         LEFT JOIN accessibles oa ON le.id = oa.entity_id -- join accessibles to get owner
         LEFT JOIN users ou ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
         LEFT JOIN entities re ON r.id = re.id -- join entities to check public flag
         LEFT JOIN files f on re.id = f.entity_id AND (f.platform = $2::text OR f.platform = '') -- join files
WHERE r.entity_id = $1::uuid
ORDER BY re.created_at DESC`
		rows, err = db.Query(ctx, q, launcherId, platform)
	} else {
		q = `SELECT r.id,
       r.name,
       r.description,
       r.version,
       r.entity_id,
       re.created_at,
       re.updated_at,
       re.views,
       ou.id,
       ou.name,
       f.id,
       f.created_at,
       f.updated_at,
       f.url,
       f.type,
       f.platform,
       f.mime,
       f.original_path,
       f.size
FROM release_v2 r
    LEFT JOIN entities le ON r.entity_id = le.id AND le.entity_type = 'launcher-v2'
    LEFT JOIN accessibles la ON le.id = la.entity_id AND la.user_id = $3 -- join accessibles to check if requester has access
	LEFT JOIN accessibles oa ON le.id = oa.entity_id -- join accessibles to get owner
	LEFT JOIN users ou ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
	LEFT JOIN entities re ON r.id = re.id -- join entities to check public flag
	LEFT JOIN files f on re.id = f.entity_id AND (f.platform = $2::text OR f.platform = '')
WHERE le.id = $1::uuid
  AND (le.public OR la.is_owner OR la.can_view)
ORDER BY le.created_at DESC`
		rows, err = db.Query(ctx, q, launcherId, platform, requester.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get launcher: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id            pgtypeuuid.UUID
			name          pgtype.Text
			description   pgtype.Text
			version       pgtype.Text
			entityId      pgtypeuuid.UUID
			createdAt     pgtype.Timestamptz
			updatedAt     pgtype.Timestamptz
			views         pgtype.Int4
			ownerId       pgtypeuuid.UUID
			ownerName     pgtype.Text
			fileId        pgtypeuuid.UUID
			fileCreatedAt pgtype.Timestamptz
			fileUpdatedAt pgtype.Timestamptz
			fileUrl       pgtype.Text
			fileType      pgtype.Text
			filePlatform  pgtype.Text
			fileMime      pgtype.Text
			filePath      pgtype.Text
			fileSize      pgtype.Int8
		)

		err = rows.Scan(&id, &name, &description, &version, &entityId, &createdAt, &updatedAt, &views, &ownerId, &ownerName, &fileId, &fileCreatedAt, &fileUpdatedAt, &fileUrl, &fileType, &filePlatform, &fileMime, &filePath, &fileSize)
		if err != nil {
			return nil, fmt.Errorf("failed to scan launcher: %w", err)
		}

		rowIndex++

		if id.Status != pgtype.Present {
			continue
		}

		var file *File
		if fileId.Status == pgtype.Present {
			file = &File{}
			file.Id = fileId.UUID
			if fileCreatedAt.Status == pgtype.Present {
				file.CreatedAt = fileCreatedAt.Time
			}
			if fileUpdatedAt.Status == pgtype.Present {
				file.UpdatedAt = &fileUpdatedAt.Time
			}
			if fileUrl.Status == pgtype.Present {
				file.Url = fileUrl.String
			}
			if fileType.Status == pgtype.Present {
				file.Type = fileType.String
			}
			if filePlatform.Status == pgtype.Present {
				file.Platform = filePlatform.String
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

		// add file to entity that has been already added to the list
		if i := GetIdentifiableIndex(helper.ToSliceOfAny(entities), id.UUID); i >= 0 {
			if file != nil && !ContainsIdentifiable(helper.ToSliceOfAny(entities[i].Files.Entities), file.Id) {
				entities[i].Files.Entities = append(entities[i].Files.Entities, *file)
			}
		} else {
			// skip entity if we previously skipped it because of offset
			if skipEntity {
				if skippedEntityId == id.UUID {
					continue
				}
			}

			// skip entities until we reach the offset
			if entityIndex < offset {
				entityIndex++
				skipEntity = true
				skippedEntityId = id.UUID
				continue
			}

			// stop if we reached the limit
			if entityIndex >= offset+limit {
				break
			}

			e := ReleaseV2{}
			e.InitFiles()
			e.Id = id.UUID
			if ownerId.Status == pgtype.Present {
				e.Owner = &User{
					Entity: Entity{
						Identifier: Identifier{Id: ownerId.UUID},
					},
				}
				if ownerName.Status == pgtype.Present {
					e.Owner.Name = &ownerName.String
				}
			}
			if name.Status == pgtype.Present {
				e.Name = &name.String
			}
			if description.Status == pgtype.Present {
				e.Description = &description.String
			}
			if createdAt.Status == pgtype.Present {
				e.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				e.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				e.Views = views.Int
			}
			if file != nil {
				e.Files.Entities = append(e.Files.Entities, *file)
			}

			entities = append(entities, e)
			skipEntity = false
			entityIndex++
		}
	}

	return entities, nil
}

// IndexLauncherV2Apps returns all apps of a launcher, does not include all app releases, only the latest one
func IndexLauncherV2Apps(ctx context.Context, requester *User, launcherId uuid.UUID, platform string, offset int64, limit int64) (entities []AppV2, err error) {
	if requester == nil {
		return nil, ErrNoRequester
	}

	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	var (
		q               string
		rows            pgx.Rows
		rowIndex        int64 = 0
		entityIndex     int64 = 0
		skipEntity            = false
		skippedEntityId uuid.UUID
	)

	if requester.IsAdmin {
		q = `SELECT a.id,
       a.name,
       a.description,
       a.external,
       a.sdk_id,
       e.created_at,
       e.updated_at,
       e.views,
       ou.id,
       ou.name,
       f.id,
       f.created_at,
       f.updated_at,
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
       rf.size,
       l.id,
       l.name,
       l.url,
       se.created_at,
       se.updated_at,
       se.views,
       sr.id,
       sre.created_at,
       sre.updated_at,
       sre.views,
       sr.version,
       sr.name,
       sr.description,
       sr.code_version,
       sr.content_version,
       sr.archive,
       srf.id,
       srf.created_at,
       srf.url,
       srf.type,
       srf.mime,
       srf.original_path,
       srf.size
FROM app_v2 a
--region Join Launcher
         LEFT JOIN launcher_apps_v2 la
                   ON la.app_id = a.id -- join launcher apps pivot table
         LEFT JOIN entities le
                   ON le.id = la.launcher_id AND le.entity_type = 'launcher-v2'
    --endregion
--region Join App
         LEFT JOIN entities e
                   ON a.id = e.id -- join app entities to check public flag
--endregion
--region Join Owner
         LEFT JOIN accessibles oa
                   ON e.id = oa.entity_id AND oa.is_owner -- join accessibles to get owner
         LEFT JOIN users ou
                   ON oa.user_id = ou.id -- join users to get owner name
--endregion
--region Join App Files
         LEFT JOIN files f
                   ON e.id = f.entity_id AND (f.platform = $2::text OR f.platform = '')
    --endregion
--region Join App Release
         LEFT JOIN release_v2 r
                   ON e.id = r.entity_id AND r.version = (SELECT max(r1.version)
                                                          FROM release_v2 r1
                                                                   LEFT JOIN entities e1
                                                                             ON r1.id = e1.id
                                                          WHERE r1.entity_id = $1::uuid
                                                            AND e1.public = true) -- join releases to get latest release for app
         LEFT JOIN entities re
                   ON r.id = re.id -- join latest release entity
         LEFT JOIN files rf
                   ON r.id = rf.entity_id AND (rf.platform = $2::text OR rf.platform = '') -- join latest release files
--endregion
--region Join App Link
         LEFT JOIN links l
                   ON e.id = l.entity_id -- join links
--endregion
--region SDK
         LEFT JOIN sdk_v2 s
                   ON a.sdk_id = s.id -- join sdk
         LEFT JOIN entities se
                   ON s.id = se.id -- join sdk entity
         LEFT JOIN files sf
                   ON s.id = sf.entity_id AND (sf.platform = $2::text OR sf.platform = '') -- join sdk files
         LEFT JOIN release_v2 sr
                   ON s.id = sr.entity_id AND sr.version = (SELECT max(r1.version)
                                                            FROM release_v2 r1
                                                                     LEFT JOIN entities e1
                                                                               ON r1.id = e1.id
                                                            WHERE r1.entity_id = $1::uuid
                                                              AND e1.public = true) -- join sdk releases to get latest release for sdk
         LEFT JOIN entities sre ON sr.id = sre.id -- join sdk latest release entity
         LEFT JOIN files srf
                   ON sr.id = srf.entity_id AND
                      (srf.platform = $2::text OR srf.platform = '') -- join sdk latest release files
--endregion
WHERE le.id = $1::uuid
ORDER BY e.created_at DESC;`
		rows, err = db.Query(ctx, q, launcherId, platform)
	} else {
		q = `SELECT a.id,
       a.name,
       a.description,
       a.external,
       a.sdk_id,
       e.created_at,
       e.updated_at,
       e.views,
       ou.id,
       ou.name,
       f.id,
       f.created_at,
       f.updated_at,
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
       rf.size,
       l.id,
       l.name,
       l.url,
       se.created_at,
       se.updated_at,
       se.views,
       sr.id,
       sre.created_at,
       sre.updated_at,
       sre.views,
       sr.version,
       sr.name,
       sr.description,
       sr.code_version,
       sr.content_version,
       sr.archive,
       srf.id,
       srf.created_at,
       srf.url,
       srf.type,
       srf.mime,
       srf.original_path,
       srf.size
FROM app_v2 a
--region Join Launcher
         LEFT JOIN launcher_apps_v2 la
                   ON la.app_id = a.id -- join launcher apps pivot table
         LEFT JOIN entities le
                   ON le.id = la.launcher_id AND le.entity_type = 'launcher-v2'
    --endregion
--region Join App
         LEFT JOIN entities e
                   ON a.id = e.id -- join app entities to check public flag
         LEFT JOIN accessibles aa
                   ON a.id = aa.entity_id AND user_id = $3::uuid -- join accessibles to check if requester has access to app
--endregion
--region Join Owner
         LEFT JOIN accessibles oa
                   ON e.id = oa.entity_id AND oa.is_owner -- join accessibles to get owner
         LEFT JOIN users ou
                   ON oa.user_id = ou.id -- join users to get owner name
--endregion
--region Join App Files
         LEFT JOIN files f
                   ON e.id = f.entity_id AND (f.platform = $2::text OR f.platform = '')
    --endregion
--region Join App Release
         LEFT JOIN release_v2 r
                   ON e.id = r.entity_id AND r.version = (SELECT max(r1.version)
                                                          FROM release_v2 r1
                                                                   LEFT JOIN entities e1
                                                                             ON r1.id = e1.id
                                                          WHERE r1.entity_id = $1::uuid
                                                            AND e1.public = true) -- join releases to get latest release for app
         LEFT JOIN entities re
                   ON r.id = re.id -- join latest release entity
         LEFT JOIN files rf
                   ON r.id = rf.entity_id AND (rf.platform = $2::text OR rf.platform = '') -- join latest release files
--endregion
--region Join App Link
         LEFT JOIN links l
                   ON e.id = l.entity_id -- join links
--endregion
--region SDK
         LEFT JOIN sdk_v2 s
                   ON a.sdk_id = s.id -- join sdk
         LEFT JOIN entities se
                   ON s.id = se.id -- join sdk entity
         LEFT JOIN accessibles sa
                   ON s.id = sa.entity_id AND sa.user_id = $3::uuid -- join accessibles to check if requester has access to sdk
         LEFT JOIN files sf
                   ON s.id = sf.entity_id AND (sf.platform = $2::text OR sf.platform = '') -- join sdk files
         LEFT JOIN release_v2 sr
                   ON s.id = sr.entity_id AND sr.version = (SELECT max(r1.version)
                                                            FROM release_v2 r1
                                                                     LEFT JOIN entities e1
                                                                               ON r1.id = e1.id
                                                            WHERE r1.entity_id = $1::uuid
                                                              AND e1.public = true) -- join sdk releases to get latest release for sdk
         LEFT JOIN entities sre ON sr.id = sre.id -- join sdk latest release entity
         LEFT JOIN files srf
                   ON sr.id = srf.entity_id AND
                      (srf.platform = $2::text OR srf.platform = '') -- join sdk latest release files
--endregion
WHERE le.id = $1::uuid
  AND (e.public OR aa.is_owner OR aa.can_view) -- check if requester has access to app
ORDER BY e.created_at DESC;`
		rows, err = db.Query(ctx, q, launcherId, platform, requester.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get launcher: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id                       pgtypeuuid.UUID
			name                     pgtype.Text
			description              pgtype.Text
			external                 pgtype.Bool
			sdkId                    pgtypeuuid.UUID
			createdAt                pgtype.Timestamptz
			updatedAt                pgtype.Timestamptz
			views                    pgtype.Int4
			ownerId                  pgtypeuuid.UUID
			ownerName                pgtype.Text
			fileId                   pgtypeuuid.UUID
			fileCreatedAt            pgtype.Timestamptz
			fileUpdatedAt            pgtype.Timestamptz
			fileUrl                  pgtype.Text
			fileType                 pgtype.Text
			fileMime                 pgtype.Text
			filePath                 pgtype.Text
			fileSize                 pgtype.Int8
			releaseId                pgtypeuuid.UUID
			releaseCreatedAt         pgtype.Timestamptz
			releaseUpdatedAt         pgtype.Timestamptz
			releaseViews             pgtype.Int4
			releaseVersion           pgtype.Text
			releaseName              pgtype.Text
			releaseDescription       pgtype.Text
			releaseCodeVersion       pgtype.Text
			releaseContentVersion    pgtype.Text
			releaseArchive           pgtype.Bool
			releaseFileId            pgtypeuuid.UUID
			releaseFileCreatedAt     pgtype.Timestamptz
			releaseFileUrl           pgtype.Text
			releaseFileType          pgtype.Text
			releaseFileMime          pgtype.Text
			releaseFilePath          pgtype.Text
			releaseFileSize          pgtype.Int8
			linkId                   pgtypeuuid.UUID
			linkUrl                  pgtype.Text
			linkName                 pgtype.Text
			sdkCreatedAt             pgtype.Timestamptz
			sdkUpdatedAt             pgtype.Timestamptz
			sdkViews                 pgtype.Int4
			sdkReleaseId             pgtypeuuid.UUID
			sdkReleaseCreatedAt      pgtype.Timestamptz
			sdkReleaseUpdatedAt      pgtype.Timestamptz
			sdkReleaseViews          pgtype.Int4
			sdkReleaseVersion        pgtype.Text
			sdkReleaseName           pgtype.Text
			sdkReleaseDescription    pgtype.Text
			sdkReleaseCodeVersion    pgtype.Text
			sdkReleaseContentVersion pgtype.Text
			sdkReleaseArchive        pgtype.Bool
			sdkReleaseFileId         pgtypeuuid.UUID
			sdkReleaseFileCreatedAt  pgtype.Timestamptz
			sdkReleaseFileUrl        pgtype.Text
			sdkReleaseFileType       pgtype.Text
			sdkReleaseFileMime       pgtype.Text
			sdkReleaseFilePath       pgtype.Text
			sdkReleaseFileSize       pgtype.Int8
		)

		err = rows.Scan(
			&id,
			&name,
			&description,
			&external,
			&sdkId,
			&createdAt,
			&updatedAt,
			&views,
			&ownerId,
			&ownerName,
			&fileId,
			&fileCreatedAt,
			&fileUpdatedAt,
			&fileUrl,
			&fileType,
			&fileMime,
			&filePath,
			&fileSize,
			&releaseId,
			&releaseCreatedAt,
			&releaseUpdatedAt,
			&releaseViews,
			&releaseVersion,
			&releaseName,
			&releaseDescription,
			&releaseCodeVersion,
			&releaseContentVersion,
			&releaseArchive,
			&releaseFileId,
			&releaseFileCreatedAt,
			&releaseFileUrl,
			&releaseFileType,
			&releaseFileMime,
			&releaseFilePath,
			&releaseFileSize,
			&linkId,
			&linkUrl,
			&linkName,
			&sdkCreatedAt,
			&sdkUpdatedAt,
			&sdkViews,
			&sdkReleaseId,
			&sdkReleaseCreatedAt,
			&sdkReleaseUpdatedAt,
			&sdkReleaseViews,
			&sdkReleaseVersion,
			&sdkReleaseName,
			&sdkReleaseDescription,
			&sdkReleaseCodeVersion,
			&sdkReleaseContentVersion,
			&sdkReleaseArchive,
			&sdkReleaseFileId,
			&sdkReleaseFileCreatedAt,
			&sdkReleaseFileUrl,
			&sdkReleaseFileType,
			&sdkReleaseFileMime,
			&sdkReleaseFilePath,
			&sdkReleaseFileSize)
		if err != nil {
			return nil, fmt.Errorf("failed to scan launcher: %w", err)
		}

		rowIndex++

		if id.Status != pgtype.Present {
			continue
		}

		var file *File
		if fileId.Status == pgtype.Present {
			file = &File{}
			file.Id = fileId.UUID
			if fileCreatedAt.Status == pgtype.Present {
				file.CreatedAt = fileCreatedAt.Time
			}
			if fileUpdatedAt.Status == pgtype.Present {
				file.UpdatedAt = &fileUpdatedAt.Time
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
			release.Entity.InitFiles()
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

		var link *Link
		if linkId.Status == pgtype.Present {
			link = &Link{}
			link.Id = linkId.UUID
			if linkUrl.Status == pgtype.Present {
				link.Url = linkUrl.String
			}
			if linkName.Status == pgtype.Present {
				link.Name = &linkName.String
			}
		}

		var sdk *SDK
		if sdkId.Status == pgtype.Present {
			sdk = &SDK{
				Releases: &ReleaseV2Batch{},
			}
			sdk.Id = sdkId.UUID
			if sdkCreatedAt.Status == pgtype.Present {
				sdk.CreatedAt = sdkCreatedAt.Time
			}
			if sdkUpdatedAt.Status == pgtype.Present {
				sdk.UpdatedAt = &sdkUpdatedAt.Time
			}
			if sdkViews.Status == pgtype.Present {
				sdk.Views = sdkViews.Int
			}
			if sdkReleaseId.Status == pgtype.Present {
				sdkRelease := ReleaseV2{}
				sdkRelease.InitFiles()
				sdkRelease.Id = sdkReleaseId.UUID
				if sdkReleaseCreatedAt.Status == pgtype.Present {
					sdkRelease.CreatedAt = sdkReleaseCreatedAt.Time
				}
				if sdkReleaseUpdatedAt.Status == pgtype.Present {
					sdkRelease.UpdatedAt = &sdkReleaseUpdatedAt.Time
				}
				if sdkReleaseViews.Status == pgtype.Present {
					sdkRelease.Views = sdkReleaseViews.Int
				}
				if sdkReleaseVersion.Status == pgtype.Present {
					sdkRelease.Version = sdkReleaseVersion.String
				}
				if sdkReleaseName.Status == pgtype.Present {
					sdkRelease.Name = &sdkReleaseName.String
				}
				if sdkReleaseDescription.Status == pgtype.Present {
					sdkRelease.Description = &sdkReleaseDescription.String
				}
				if sdkReleaseCodeVersion.Status == pgtype.Present {
					sdkRelease.CodeVersion = sdkReleaseCodeVersion.String
				}
				if sdkReleaseContentVersion.Status == pgtype.Present {
					sdkRelease.ContentVersion = sdkReleaseContentVersion.String
				}
				if sdkReleaseArchive.Status == pgtype.Present {
					sdkRelease.Archive = sdkReleaseArchive.Bool
				}
				sdk.Releases.Entities = append(sdk.Releases.Entities, sdkRelease)
			}
		}

		var releaseFile *File
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

		var sdkReleaseFile *File
		if sdkReleaseFileId.Status == pgtype.Present {
			sdkReleaseFile = &File{}
			sdkReleaseFile.Id = sdkReleaseFileId.UUID
			if sdkReleaseFileCreatedAt.Status == pgtype.Present {
				sdkReleaseFile.CreatedAt = sdkReleaseFileCreatedAt.Time
			}
			if sdkReleaseFileUrl.Status == pgtype.Present {
				sdkReleaseFile.Url = sdkReleaseFileUrl.String
			}
			if sdkReleaseFileType.Status == pgtype.Present {
				sdkReleaseFile.Type = sdkReleaseFileType.String
			}
			if sdkReleaseFileMime.Status == pgtype.Present {
				sdkReleaseFile.Mime = &sdkReleaseFileMime.String
			}
			if sdkReleaseFilePath.Status == pgtype.Present {
				sdkReleaseFile.OriginalPath = &sdkReleaseFilePath.String
			}
			if sdkReleaseFileSize.Status == pgtype.Present {
				sdkReleaseFile.Size = &sdkReleaseFileSize.Int
			}
		}

		// add file to entity that has been already added to the list
		if i := GetIdentifiableIndex(helper.ToSliceOfAny(entities), id.UUID); i >= 0 {
			if file != nil && !ContainsIdentifiable(helper.ToSliceOfAny(entities[i].Files.Entities), file.Id) {
				entities[i].Files.Entities = append(entities[i].Files.Entities, *file)
				entities[i].Files.Total++
			}

			if release != nil && !ContainsIdentifiable(helper.ToSliceOfAny(entities[i].Releases.Entities), release.Id) {
				entities[i].Releases.Entities = append(entities[i].Releases.Entities, *release)
				entities[i].Releases.Total++

				// use index 0 as we are sure that there is only one latest release
				if releaseFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(entities[i].Releases.Entities[0].Files.Entities), releaseFile.Id) {
					entities[i].Releases.Entities[0].Files.Entities = append(entities[i].Releases.Entities[0].Files.Entities, *releaseFile)
					entities[i].Releases.Entities[0].Files.Total++
				}
			}

			if link != nil && !ContainsIdentifiable(helper.ToSliceOfAny(entities[i].Links.Entities), link.Id) {
				entities[i].Links.Entities = append(entities[i].Links.Entities, *link)
				entities[i].Links.Total++
			}

			if sdk != nil && entities[i].SDK == nil {
				entities[i].SDK = sdk

				// use index 0 as we are sure that there is only one latest release
				if sdkReleaseFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(entities[i].SDK.Releases.Entities[0].Files.Entities), sdkReleaseFile.Id) {
					entities[i].SDK.Releases.Entities[0].Files.Entities = append(entities[i].SDK.Releases.Entities[0].Files.Entities, *sdkReleaseFile)
					entities[i].SDK.Releases.Entities[0].Files.Total++
				}
			}
		} else {
			// skip entity if we previously skipped it because of offset
			if skipEntity {
				if skippedEntityId == id.UUID {
					continue
				}
			}

			// skip entities until we reach the offset
			if entityIndex < offset {
				entityIndex++
				skipEntity = true
				skippedEntityId = id.UUID
				continue
			}

			// stop if we reached the limit
			if entityIndex >= offset+limit {
				break
			}

			e := AppV2{}
			e.InitFiles()
			e.InitReleases()
			e.InitLinks()
			e.Id = id.UUID
			if ownerId.Status == pgtype.Present {
				e.Owner = &User{
					Entity: Entity{
						Identifier: Identifier{Id: ownerId.UUID},
					},
				}
				if ownerName.Status == pgtype.Present {
					e.Owner.Name = &ownerName.String
				}
			}
			if name.Status == pgtype.Present {
				e.Name = name.String
			}
			if createdAt.Status == pgtype.Present {
				e.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				e.UpdatedAt = &updatedAt.Time
			}
			if views.Status == pgtype.Present {
				e.Views = views.Int
			}
			if file != nil {
				e.Files.Entities = append(e.Files.Entities, *file)
				e.Files.Total++
			}
			if release != nil {
				e.Releases.Entities = append(e.Releases.Entities, *release)
				e.Releases.Total++

				if releaseFile != nil {
					e.Releases.Entities[0].Files.Entities = append(e.Releases.Entities[0].Files.Entities, *releaseFile)
					e.Releases.Entities[0].Files.Total++
				}
			}
			if link != nil {
				e.Links.Entities = append(e.Links.Entities, *link)
				e.Links.Total++
			}
			if sdk != nil && e.SDK == nil {
				e.SDK = sdk

				if sdkReleaseFile != nil {
					e.SDK.Releases.Entities[0].Files.Entities = append(e.SDK.Releases.Entities[0].Files.Entities, *sdkReleaseFile)
					e.SDK.Releases.Entities[0].Files.Total++
				}
			}

			entities = append(entities, e)
			skipEntity = false
			entityIndex++
		}
	}

	return entities, nil
}
