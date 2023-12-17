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
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type World struct {
	Entity

	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Map         string    `json:"map"`
	PackageId   uuid.UUID `json:"modId"`
	Type        string    `json:"type"`
	Scheduled   bool      `json:"scheduled"`
	GameMode    string    `json:"gameMode"`
	Package     *Package  `json:"mod"`
}

var knownColumns = map[string]string{
	"id":          "w.id",
	"name":        "w.name",
	"description": "w.description",
	"map":         "w.map",
	"packageId":   "w.mod_id",
	"type":        "w.type",
	"scheduled":   "w.scheduled",
	"gameMode":    "w.game_mode",
	"views":       "e.views",
	"createdAt":   "e.created_at",
	"updatedAt":   "e.updated_at",
	"public":      "e.public",
	"likes":       "likes",
	"dislikes":    "dislikes",
	"pakFile":     "pkf.url",
	"previewFile": "pf.url",
}

var knownDirections = map[string]bool{
	"asc":  true,
	"desc": true,
}

type WorldBatch Batch[World]

type WorldRequestPakOptions struct {
	Platform   string `json:"platform"`
	Deployment string `json:"deployment"`
}

type WorldRequestOptions struct {
	Pak        bool                    `json:"pak"`
	PakOptions *WorldRequestPakOptions `json:"pakOptions"`
	Preview    bool                    `json:"preview"`
	Likes      bool                    `json:"likes"`
	Owner      bool                    `json:"owner"`
}

type IndexWorldRequest struct {
	Offset  *int64               `json:"offset"`
	Limit   *int64               `json:"limit"`
	Search  *string              `json:"search"`
	Sort    []IndexRequestSort   `json:"sort"`
	Options *WorldRequestOptions `json:"options"`
}

func IndexWorld(ctx context.Context, requester *User, request IndexWorldRequest) (batch *WorldBatch, err error) {
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
	batch = &WorldBatch{
		Offset: 0,
		Limit:  100,
		Total:  0,
	}

	// set offset if it is in a valid range
	if request.Offset != nil && *request.Offset >= 0 {
		batch.Offset = *request.Offset
	}

	// set limit if it is in a valid range
	if request.Limit != nil && *request.Limit > 0 && *request.Limit <= 100 {
		batch.Limit = *request.Limit
	}

	// validate request options
	if request.Options != nil {
		if request.Options.Pak {
			if request.Options.PakOptions == nil {
				return nil, fmt.Errorf("pak options are required when pak is requested")
			} else if request.Options.PakOptions.Platform == "" {
				return nil, fmt.Errorf("pak platform option is required when pak is requested")
			} else if request.Options.PakOptions.Deployment == "" {
				return nil, fmt.Errorf("pak deployment option is required when pak is requested")
			}
		}
	}

	var (
		qt        string                      // total query
		qtArgs    = make([]any, 0)            // total query args
		qtArgNum  = 0                         // total query arg number
		q         string                      // query
		qArgs     = make([]any, 0)            // query args
		qArgNum   = 0                         // query arg number
		qSort     = make([]string, 0)         // query sort
		rows      pgx.Rows                    // rows
		ei        int64               = 0     // processed entity index
		skip                          = false // skip row
		skippedId uuid.UUID                   // skipped id
	)

	// base for total query
	qt = `select count(w.id) from spaces w left join entities e on w.id = e.id`

	// query select
	q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, w.name, w.description, w.map, w.mod_id, w.type, w.scheduled, w.game_mode` // 13
	if request.Options != nil {
		if request.Options.Likes {
			// add like columns
			q += `, rl.value as liked, sum(case when l.value >= 0 then l.value end) as likes, sum(case when l.value < 0 then l.value end) as dislikes` // 16
		}
		if request.Options.Preview {
			// add preview file columns
			q += `, pf.id, pf.entity_id, pf.type, pf.url, pf.mime, pf.size, pf.version, pf.deployment_type, pf.platform, pf.uploaded_by, pf.created_at, pf.updated_at, pf.variation, pf.original_path, pf.hash` // 29
		}
		if request.Options.Pak {
			// add package columns
			q += `, pk.id, pk.name, pk.title` // 32
			// add pak file columns
			q += `, pkf.id, pkf.entity_id, pkf.type, pkf.url, pkf.mime, pkf.size, pkf.version, pkf.deployment_type, pkf.platform, pkf.uploaded_by, pkf.created_at, pkf.updated_at, pkf.variation, pkf.original_path, pkf.hash` // 54
		}
		if request.Options.Owner {
			// add owner columns
			q += `, u.id, u.name, u.description, u.eth_address, u.is_banned` // 62
		}
	}

	// query from
	q += ` from spaces w left join entities e on w.id = e.id`
	if request.Options != nil {
		if request.Options.Likes {
			// add like join
			qArgNum++
			qArgs = append(qArgs, requester.Id)
			q += ` left join likables l on w.id = l.entity_id left join likables rl on w.id = rl.entity_id and rl.user_id = $` + strconv.Itoa(qArgNum)
		}
		if request.Options.Preview {
			// add preview file join
			q += ` left join files pf on e.id = pf.entity_id and pf.type = 'image_preview'`
		}
		if request.Options.Pak {
			// add pak file join
			qArgNum += 2
			qArgs = append(qArgs, request.Options.PakOptions.Platform, request.Options.PakOptions.Deployment)
			qt += ` left join mods pk on pk.id = w.mod_id left join entities epk on epk.id = pk.id`
			q += ` left join mods pk on pk.id = w.mod_id left join entities epk on epk.id = pk.id left join files pkf on epk.id = pkf.entity_id and pkf.type = 'pak' and pkf.platform = $` + strconv.Itoa(qArgNum-1) + ` and pkf.deployment_type = $` + strconv.Itoa(qArgNum)
		}
		if request.Options.Owner {
			// add owner accessibles (to determine owner relationship) and owner user join
			q += ` left join accessibles au on e.id = au.entity_id and au.is_owner`
			q += ` left join users u on au.user_id = u.id`
		}
	}

	// query where
	if !requester.IsAdmin {
		// if the requester is not an admin, only show public entities and entities the requester has access to
		qArgNum++
		qArgs = append(qArgs, requester.Id)
		qtArgNum++
		qtArgs = append(qtArgs, requester.Id)
		qt += ` left join accessibles a on e.id = a.entity_id and a.user_id = $` + strconv.Itoa(qtArgNum)
		q += ` left join accessibles a on e.id = a.entity_id and a.user_id = $` + strconv.Itoa(qArgNum)

		// if pak is requested, only show pak files that are public
		if request.Options != nil && request.Options.Pak {
			qt += ` left join accessibles ap on epk.id = ap.entity_id and ap.user_id = $` + strconv.Itoa(qtArgNum) + ` where (e.public or (a.is_owner or a.can_view)) and (epk.public or (ap.is_owner or ap.can_view))`
			q += ` left join accessibles ap on epk.id = ap.entity_id and ap.user_id = $` + strconv.Itoa(qArgNum) + ` where (e.public or (a.is_owner or a.can_view)) and (epk.public or (ap.is_owner or ap.can_view))`
		} else {
			qt += ` where (e.public or (a.is_owner or a.can_view))`
			q += ` where (e.public or (a.is_owner or a.can_view))`
		}
	} else {
		// if the requester is an admin, show all entities
		qt += ` where true`
		q += ` where true`
	}

	// query search by name or description
	if request.Search != nil {
		qArgNum++
		qArgs = append(qArgs, "%"+helper.SanitizeLikeClause(*request.Search)+"%")
		qtArgNum++
		qtArgs = append(qtArgs, "%"+helper.SanitizeLikeClause(*request.Search)+"%")
		qt += ` and (w.name ilike $` + strconv.Itoa(qtArgNum) + ` or w.description ilike $` + strconv.Itoa(qtArgNum) + `) `
		q += ` and (w.name ilike $` + strconv.Itoa(qArgNum) + ` or w.description ilike $` + strconv.Itoa(qArgNum) + `) `
	}

	// add group by if likes are requested
	if request.Options != nil {
		if request.Options.Likes {
			q += ` group by e.id, rl.value, w.name, w.description, w.map, w.mod_id, w.type, w.scheduled, w.game_mode`
			if request.Options.Preview {
				q += `, pf.id, pf.entity_id, pf.type, pf.url, pf.mime, pf.size, pf.version, pf.deployment_type, pf.platform, pf.uploaded_by, pf.created_at, pf.updated_at, pf.variation, pf.original_path, pf.hash`
			}
			if request.Options.Pak {
				q += `, pk.id, pk.name, pk.description`
				q += `, pkf.id, pkf.entity_id, pkf.type, pkf.url, pkf.mime, pkf.size, pkf.version, pkf.deployment_type, pkf.platform, pkf.uploaded_by, pkf.created_at, pkf.updated_at, pkf.variation, pkf.original_path, pkf.hash`
			}
			if request.Options.Owner {
				q += `, u.id, u.name, u.description, u.eth_address, u.is_banned`
			}
		}
	}

	// apply sort
	if len(request.Sort) > 0 {
		for _, s := range request.Sort {
			var (
				column string
				ok     bool
			)
			if column, ok = knownColumns[s.Column]; !ok {
				continue
			}
			if s.Direction != "" && !knownDirections[s.Direction] {
				continue
			}
			// skip likes if not requested
			if !request.Options.Likes && s.Column == "likes" || s.Column == "dislikes" {
				continue
			}
			qSort = append(qSort, column+" "+s.Direction)
		}
		q += ` order by ` + strings.Join(qSort, ", ")
	}

	// query total
	err = db.QueryRow(ctx, qt, qtArgs...).Scan(&batch.Total)
	if err != nil {
		return nil, err
	}

	// apply limit and offset to query
	//qArgs = append(qArgs, batch.Limit)
	//qArgs = append(qArgs, batch.Offset)
	//q += ` limit $` + strconv.Itoa(qArgNum+1) + ` offset $` + strconv.Itoa(qArgNum+2)

	// query entities
	rows, err = db.Query(ctx, q, qArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			entity      Entity
			world       World
			pack        Package
			owner       User
			previewFile *File
			pakFile     *File
		)
		var (
			id          pgtypeuuid.UUID
			createdAt   pgtype.Timestamp
			updatedAt   pgtype.Timestamp
			entityType  pgtype.Text
			views       pgtype.Int4
			public      pgtype.Bool
			name        pgtype.Text
			description pgtype.Text
			worldMap    pgtype.Text
			modId       pgtypeuuid.UUID
			worldType   pgtype.Text
			scheduled   pgtype.Bool
			gameMode    pgtype.Text
		)
		allFields := []interface{}{
			&id, &createdAt, &updatedAt, &entityType, &views, &public, &name, &description, &worldMap, &modId, &worldType, &scheduled, &gameMode,
		}

		var (
			likedByRequester pgtype.Int4
			likeCount        pgtype.Int4
			dislikeCount     pgtype.Int4
		)
		likeFields := []interface{}{
			&likedByRequester, &likeCount, &dislikeCount,
		}

		var (
			previewFileId           pgtypeuuid.UUID
			previewFileEntityId     pgtypeuuid.UUID
			previewFileType         pgtype.Text
			previewFileUrl          pgtype.Text
			previewFileMime         pgtype.Text
			previewFileSize         pgtype.Int8
			previewFileVersion      pgtype.Int8
			previewFileDeployment   pgtype.Text
			previewFilePlatform     pgtype.Text
			previewFileUploadedBy   pgtypeuuid.UUID
			previewFileCreatedAt    pgtype.Timestamp
			previewFileUpdatedAt    pgtype.Timestamp
			previewFileVariation    pgtype.Int8
			previewFileOriginalPath pgtype.Text
			previewFileHash         pgtype.Text
		)
		previewFields := []interface{}{
			&previewFileId, &previewFileEntityId, &previewFileType, &previewFileUrl, &previewFileMime, &previewFileSize, &previewFileVersion, &previewFileDeployment, &previewFilePlatform, &previewFileUploadedBy, &previewFileCreatedAt, &previewFileUpdatedAt, &previewFileVariation, &previewFileOriginalPath, &previewFileHash,
		}

		var (
			packageId           pgtypeuuid.UUID
			packageName         pgtype.Text
			packageDescription  pgtype.Text
			pakFileId           pgtypeuuid.UUID
			pakFileEntityId     pgtypeuuid.UUID
			pakFileType         pgtype.Text
			pakFileUrl          pgtype.Text
			pakFileMime         pgtype.Text
			pakFileSize         pgtype.Int8
			pakFileVersion      pgtype.Int8
			pakFileDeployment   pgtype.Text
			pakFilePlatform     pgtype.Text
			pakFileUploadedBy   pgtypeuuid.UUID
			pakFileCreatedAt    pgtype.Timestamp
			pakFileUpdatedAt    pgtype.Timestamp
			pakFileVariation    pgtype.Int8
			pakFileOriginalPath pgtype.Text
			pakFileHash         pgtype.Text
		)
		packageFields := []interface{}{
			&packageId, &packageName, &packageDescription,
		}
		pakFileFields := []interface{}{
			&pakFileId, &pakFileEntityId, &pakFileType, &pakFileUrl, &pakFileMime, &pakFileSize, &pakFileVersion, &pakFileDeployment, &pakFilePlatform, &pakFileUploadedBy, &pakFileCreatedAt, &pakFileUpdatedAt, &pakFileVariation, &pakFileOriginalPath, &pakFileHash,
		}

		var (
			ownerId          pgtypeuuid.UUID
			ownerName        pgtype.Text
			ownerDescription pgtype.Text
			ownerEthAddress  pgtype.Text
			ownerIsBanned    pgtype.Bool
		)

		if request.Options != nil {
			if request.Options.Likes {
				allFields = append(allFields, likeFields...)
			}

			if request.Options.Preview {
				allFields = append(allFields, previewFields...)
			}

			if request.Options.Pak {
				allFields = append(allFields, packageFields...)
				allFields = append(allFields, pakFileFields...)
			}

			if request.Options.Owner {
				allFields = append(allFields, &ownerId, &ownerName, &ownerDescription, &ownerEthAddress, &ownerIsBanned)
			}
		}

		err = rows.Scan(allFields...)
		if err != nil {
			return nil, err
		}

		// skip if id is null
		if id.Status != pgtype.Present {
			continue
		}

		// pak file
		if pakFileId.Status == pgtype.Present {
			pakFile = &File{}
			pakFile.Id = pakFileId.UUID
			if pakFileType.Status == pgtype.Present {
				pakFile.Type = pakFileType.String
			}
			if pakFileMime.Status == pgtype.Present {
				pakFile.Mime = &pakFileMime.String
			}
			if pakFileUrl.Status == pgtype.Present {
				pakFile.Url = pakFileUrl.String
			}
			if pakFileSize.Status == pgtype.Present {
				pakFile.Size = &pakFileSize.Int
			}
			if pakFileVersion.Status == pgtype.Present {
				pakFile.Version = pakFileVersion.Int
			}
			if pakFileDeployment.Status == pgtype.Present {
				pakFile.Deployment = pakFileDeployment.String
			}
			if pakFilePlatform.Status == pgtype.Present {
				pakFile.Platform = pakFilePlatform.String
			}
			if pakFileUploadedBy.Status == pgtype.Present {
				pakFile.UploadedBy = &pakFileUploadedBy.UUID
			}
			if pakFileCreatedAt.Status == pgtype.Present {
				pakFile.CreatedAt = pakFileCreatedAt.Time
			}
			if pakFileUpdatedAt.Status == pgtype.Present {
				pakFile.UpdatedAt = &pakFileUpdatedAt.Time
			}
			if pakFileVariation.Status == pgtype.Present {
				pakFile.Index = pakFileVariation.Int
			}
			if pakFileOriginalPath.Status == pgtype.Present {
				pakFile.OriginalPath = &pakFileOriginalPath.String
			}
			if pakFileHash.Status == pgtype.Present {
				pakFile.Hash = &pakFileHash.String
			}
		}

		// preview file
		if previewFileId.Status == pgtype.Present {
			previewFile = &File{}
			previewFile.Id = previewFileId.UUID
			if previewFileType.Status == pgtype.Present {
				previewFile.Type = previewFileType.String
			}
			if previewFileMime.Status == pgtype.Present {
				previewFile.Mime = &previewFileMime.String
			}
			if previewFileUrl.Status == pgtype.Present {
				previewFile.Url = previewFileUrl.String
			}
			if previewFileSize.Status == pgtype.Present {
				previewFile.Size = &previewFileSize.Int
			}
			if previewFileVersion.Status == pgtype.Present {
				previewFile.Version = previewFileVersion.Int
			}
			if previewFileDeployment.Status == pgtype.Present {
				previewFile.Deployment = previewFileDeployment.String
			}
			if previewFilePlatform.Status == pgtype.Present {
				previewFile.Platform = previewFilePlatform.String
			}
			if previewFileUploadedBy.Status == pgtype.Present {
				previewFile.UploadedBy = &previewFileUploadedBy.UUID
			}
			if previewFileCreatedAt.Status == pgtype.Present {
				previewFile.CreatedAt = previewFileCreatedAt.Time
			}
			if previewFileUpdatedAt.Status == pgtype.Present {
				previewFile.UpdatedAt = &previewFileUpdatedAt.Time
			}
			if previewFileVariation.Status == pgtype.Present {
				previewFile.Index = previewFileVariation.Int
			}
			if previewFileOriginalPath.Status == pgtype.Present {
				previewFile.OriginalPath = &previewFileOriginalPath.String
			}
			if previewFileHash.Status == pgtype.Present {
				previewFile.Hash = &previewFileHash.String
			}
		}

		// entity already present in batch
		if i := GetIdentifiableIndex(helper.ToSliceOfAny(batch.Entities), id.UUID); i >= 0 {
			// add pak file to entity
			if batch == nil {
				logrus.Errorf("batch is nil")
			} else if batch.Entities == nil {
				logrus.Errorf("batch.Entities is nil")
			} else if batch.Entities[i].Files == nil {
				logrus.Errorf("batch.Entities[i].Files is nil")
			}
			if batch.Entities[i].Files != nil {
				if previewFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(batch.Entities[i].Files.Entities), previewFile.Id) {
					batch.Entities[i].Files.Entities = append(batch.Entities[i].Files.Entities, *previewFile)
				}
			}

			// add pak file to entity
			if batch.Entities[i].Package == nil {
				logrus.Errorf("batch.Entities[i].Package is nil")
			} else if batch.Entities[i].Package.Files == nil {
				logrus.Errorf("batch.Entities[i].Package.Files is nil")
			}
			if batch.Entities[i].Package != nil && batch.Entities[i].Package.Files != nil {
				if pakFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(batch.Entities[i].Package.Files.Entities), pakFile.Id) {
					batch.Entities[i].Package.Files.Entities = append(batch.Entities[i].Package.Files.Entities, *pakFile)
				}
			}
		} else {
			// check if entity should be skipped
			if skip {
				if id.UUID == skippedId {
					continue
				}
			}

			// check if entity should be added
			if ei < batch.Offset {
				ei++
				skip = true
				skippedId = id.UUID
				continue
			}

			// check if batch limit is reached
			if ei-batch.Offset >= batch.Limit {
				break
			}

			// fill entity
			entity.Id = id.UUID
			if createdAt.Status == pgtype.Present {
				entity.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				entity.UpdatedAt = &updatedAt.Time
			}
			if entityType.Status == pgtype.Present {
				entity.EntityType = entityType.String
			}
			if views.Status == pgtype.Present {
				entity.Views = views.Int
			}
			if public.Status == pgtype.Present {
				entity.Public = public.Bool
			}
			if likeCount.Status == pgtype.Present {
				entity.Likes = &likeCount.Int
			}
			if dislikeCount.Status == pgtype.Present {
				entity.Dislikes = &dislikeCount.Int
			}
			if likedByRequester.Status == pgtype.Present {
				entity.Liked = &likedByRequester.Int
			}
			// fill world
			world.Entity = entity
			if name.Status == pgtype.Present {
				world.Name = name.String
			}
			if description.Status == pgtype.Present {
				world.Description = &description.String
			}
			if worldMap.Status == pgtype.Present {
				world.Map = worldMap.String
			}
			if modId.Status == pgtype.Present {
				world.PackageId = modId.UUID
			}
			if worldType.Status == pgtype.Present {
				world.Type = worldType.String
			}
			if scheduled.Status == pgtype.Present {
				world.Scheduled = scheduled.Bool
			}
			if gameMode.Status == pgtype.Present {
				world.GameMode = gameMode.String
			}
			if previewFile != nil {
				world.Files = &FileBatch{}
				world.Files.Entities = append(world.Files.Entities, *previewFile)
			}
			// fill package
			if packageId.Status == pgtype.Present {
				pack.Id = packageId.UUID
				if packageName.Status == pgtype.Present {
					pack.Name = packageName.String
				}
				if packageDescription.Status == pgtype.Present {
					pack.Description = packageDescription.String
				}
				if pakFile != nil {
					pack.Files = &FileBatch{}
					pack.Files.Entities = append(pack.Files.Entities, *pakFile)
				}
				world.Package = &pack
			}
			// fill owner
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
				world.Owner = &owner
			}

			batch.Entities = append(batch.Entities, world)
			skip = false
			ei++
		}
	}

	return batch, nil
}

type GetWorldRequest struct {
	Id      uuid.UUID
	Options *WorldRequestOptions `json:"options"`
}

func GetWorld(ctx context.Context, requester *User, request GetWorldRequest) (world *World, err error) {
	// validate requester
	if requester == nil || requester.Id == uuid.Nil {
		return nil, ErrNoRequester
	}

	// get database connection
	db, ok := ctx.Value(glContext.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		return nil, ErrNoDatabase
	}

	// validate request options
	if request.Options != nil {
		if request.Options.Pak {
			if request.Options.PakOptions == nil {
				return nil, fmt.Errorf("pak options are required when pak is requested")
			} else if request.Options.PakOptions.Platform == "" {
				return nil, fmt.Errorf("pak platform option is required when pak is requested")
			} else if request.Options.PakOptions.Deployment == "" {
				return nil, fmt.Errorf("pak deployment option is required when pak is requested")
			}
		}
	}

	var (
		q       string           // query
		qArgs   = make([]any, 0) // query args
		qArgNum = 0              // query arg number
		rows    pgx.Rows         // rows
	)

	// build query
	q = `select e.id, e.created_at, e.updated_at, e.entity_type, e.views, e.public, w.name, w.description, w.map, w.mod_id, w.type, w.scheduled, w.game_mode` // 13
	if request.Options != nil {
		if request.Options.Likes {
			// add like columns
			q += `, rl.value as liked, sum(case when l.value >= 0 then l.value end) as likes, sum(case when l.value < 0 then l.value end) as dislikes` // 16 (+3)
		}
		if request.Options.Preview {
			// add preview file columns
			q += `, pf.id, pf.entity_id, pf.type, pf.url, pf.mime, pf.size, pf.version, pf.deployment_type, pf.platform, pf.uploaded_by, pf.created_at, pf.updated_at, pf.variation, pf.original_path, pf.hash` // 31 (+15)
		}
		if request.Options.Pak {
			// add package columns
			q += `, pk.id, pk.name, pk.title` // 34 (+3)
			// add pak file columns
			q += `, pkf.id, pkf.entity_id, pkf.type, pkf.url, pkf.mime, pkf.size, pkf.version, pkf.deployment_type, pkf.platform, pkf.uploaded_by, pkf.created_at, pkf.updated_at, pkf.variation, pkf.original_path, pkf.hash` // 49 (+15)
			// add extra package file columns
			q += `, pkef.id, pkef.entity_id, pkef.type, pkef.url, pkef.mime, pkef.size, pkef.version, pkef.deployment_type, pkef.platform, pkef.uploaded_by, pkef.created_at, pkef.updated_at, pkef.variation, pkef.original_path, pkef.hash` // 64 (+15)
		}
		if request.Options.Owner {
			// add owner columns
			q += `, u.id, u.name, u.description, u.eth_address, u.is_banned` // 69 (+5)
		}
	}

	// query from
	q += ` from spaces w left join entities e on w.id = e.id`
	if request.Options != nil {
		if request.Options.Likes {
			// add like join
			qArgNum++
			qArgs = append(qArgs, requester.Id)
			q += ` left join likables l on e.id = l.entity_id left join likables rl on w.id = rl.entity_id and rl.user_id = $` + strconv.Itoa(qArgNum)
		}
		if request.Options.Preview {
			// add preview file joins
			q += ` left join files pf on e.id = pf.entity_id and pf.type = 'image_preview'`
		}
		if request.Options.Pak {
			// add package joins
			qArgNum += 2
			qArgs = append(qArgs, request.Options.PakOptions.Platform, request.Options.PakOptions.Deployment)
			q += ` left join mods pk on pk.id = w.mod_id left join entities epk on epk.id = pk.id left join files pkf on epk.id = pkf.entity_id and pkf.type = 'pak' and pkf.platform = $` + strconv.Itoa(qArgNum-1) + ` and pkf.deployment_type = $` + strconv.Itoa(qArgNum)
			q += ` left join files pkef on epk.id = pkef.entity_id and pkef.type = 'pak-extra-content'`
		}
		if request.Options.Owner {
			// add owner accessibles (to determine owner relationship) and owner user join
			q += ` left join accessibles au on e.id = au.entity_id and au.is_owner`
			q += ` left join users u on au.user_id = u.id`
		}
	}

	// query where
	if !requester.IsAdmin {
		// If the requester is not an admin, they can only see public worlds or worlds they have explicit access to
		qArgNum++
		qArgs = append(qArgs, requester.Id)
		q += ` left join accessibles a on e.id = a.entity_id and a.user_id = $` + strconv.Itoa(qArgNum)

		// if pak is requested, we need to check if the requester has access to the pak
		if request.Options != nil && request.Options.Pak {
			q += ` left join accessibles ap on epk.id = ap.entity_id and ap.user_id = $` + strconv.Itoa(qArgNum) + ` where (e.public or (a.is_owner or a.can_view)) and (epk.public or (ap.is_owner or ap.can_view))`
		} else {
			q += ` where (e.public or (a.is_owner or a.can_view))`
		}

		qArgNum++
		qArgs = append(qArgs, request.Id)
		q += ` and e.id = $` + strconv.Itoa(qArgNum)
	} else {
		// If the requester is an admin, they can see all worlds
		qArgNum++
		qArgs = append(qArgs, request.Id)
		q += ` where e.id = $` + strconv.Itoa(qArgNum)
	}

	// add group by if likes are requested
	if request.Options != nil {
		if request.Options.Likes {
			q += ` group by e.id, rl.value, w.name, w.description, w.map, w.mod_id, w.type, w.scheduled, w.game_mode`
			if request.Options.Preview {
				q += `, pf.id, pf.entity_id, pf.type, pf.url, pf.mime, pf.size, pf.version, pf.deployment_type, pf.platform, pf.uploaded_by, pf.created_at, pf.updated_at, pf.variation, pf.original_path, pf.hash`
			}
			if request.Options.Pak {
				q += `, pk.id, pk.name, pk.description`
				q += `, pkf.id, pkf.entity_id, pkf.type, pkf.url, pkf.mime, pkf.size, pkf.version, pkf.deployment_type, pkf.platform, pkf.uploaded_by, pkf.created_at, pkf.updated_at, pkf.variation, pkf.original_path, pkf.hash`
				q += `, pkef.id, pkef.entity_id, pkef.type, pkef.url, pkef.mime, pkef.size, pkef.version, pkef.deployment_type, pkef.platform, pkef.uploaded_by, pkef.created_at, pkef.updated_at, pkef.variation, pkef.original_path, pkef.hash`
			}
			if request.Options.Owner {
				q += `, u.id, u.name, u.description, u.eth_address, u.is_banned`
			}
		}
	}

	// query rows
	rows, err = db.Query(ctx, q, qArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			pack         Package
			owner        User
			previewFile  *File
			pakFile      *File
			pakExtraFile *File
		)
		var (
			id                   pgtypeuuid.UUID
			createdAt, updatedAt pgtype.Timestamptz
			entityType           pgtype.Text
			views                pgtype.Int4
			public               pgtype.Bool
			name                 pgtype.Text
			description          pgtype.Text
			worldMap             pgtype.Text
			modId                pgtypeuuid.UUID
			worldType            pgtype.Text
			scheduled            pgtype.Bool
			gameMode             pgtype.Text
		)
		allFields := []interface{}{
			&id, &createdAt, &updatedAt, &entityType, &views, &public, &name, &description, &worldMap, &modId, &worldType, &scheduled, &gameMode,
		}

		var (
			likedByRequester pgtype.Int4
			likeCount        pgtype.Int4
			dislikeCount     pgtype.Int4
		)
		likeFields := []interface{}{
			&likedByRequester, &likeCount, &dislikeCount,
		}

		var (
			previewFileId           pgtypeuuid.UUID
			previewFileEntityId     pgtypeuuid.UUID
			previewFileType         pgtype.Text
			previewFileUrl          pgtype.Text
			previewFileMime         pgtype.Text
			previewFileSize         pgtype.Int8
			previewFileVersion      pgtype.Int8
			previewFileDeployment   pgtype.Text
			previewFilePlatform     pgtype.Text
			previewFileUploadedBy   pgtypeuuid.UUID
			previewFileCreatedAt    pgtype.Timestamp
			previewFileUpdatedAt    pgtype.Timestamp
			previewFileVariation    pgtype.Int8
			previewFileOriginalPath pgtype.Text
			previewFileHash         pgtype.Text
		)
		previewFields := []interface{}{
			&previewFileId, &previewFileEntityId, &previewFileType, &previewFileUrl, &previewFileMime, &previewFileSize, &previewFileVersion, &previewFileDeployment, &previewFilePlatform, &previewFileUploadedBy, &previewFileCreatedAt, &previewFileUpdatedAt, &previewFileVariation, &previewFileOriginalPath, &previewFileHash,
		}
		var (
			packageId                pgtypeuuid.UUID
			packageName              pgtype.Text
			packageDescription       pgtype.Text
			pakFileId                pgtypeuuid.UUID
			pakFileEntityId          pgtypeuuid.UUID
			pakFileType              pgtype.Text
			pakFileUrl               pgtype.Text
			pakFileMime              pgtype.Text
			pakFileSize              pgtype.Int8
			pakFileVersion           pgtype.Int8
			pakFileDeployment        pgtype.Text
			pakFilePlatform          pgtype.Text
			pakFileUploadedBy        pgtypeuuid.UUID
			pakFileCreatedAt         pgtype.Timestamp
			pakFileUpdatedAt         pgtype.Timestamp
			pakFileVariation         pgtype.Int8
			pakFileOriginalPath      pgtype.Text
			pakFileHash              pgtype.Text
			pakExtraFileId           pgtypeuuid.UUID
			pakExtraFileEntityId     pgtypeuuid.UUID
			pakExtraFileType         pgtype.Text
			pakExtraFileUrl          pgtype.Text
			pakExtraFileMime         pgtype.Text
			pakExtraFileSize         pgtype.Int8
			pakExtraFileVersion      pgtype.Int8
			pakExtraFileDeployment   pgtype.Text
			pakExtraFilePlatform     pgtype.Text
			pakExtraFileUploadedBy   pgtypeuuid.UUID
			pakExtraFileCreatedAt    pgtype.Timestamp
			pakExtraFileUpdatedAt    pgtype.Timestamp
			pakExtraFileVariation    pgtype.Int8
			pakExtraFileOriginalPath pgtype.Text
			pakExtraFileHash         pgtype.Text
		)
		packageFields := []interface{}{
			&packageId, &packageName, &packageDescription,
		}
		pakFileFields := []interface{}{
			&pakFileId, &pakFileEntityId, &pakFileType, &pakFileUrl, &pakFileMime, &pakFileSize, &pakFileVersion, &pakFileDeployment, &pakFilePlatform, &pakFileUploadedBy, &pakFileCreatedAt, &pakFileUpdatedAt, &pakFileVariation, &pakFileOriginalPath, &pakFileHash,
		}
		pakExtraFileFields := []interface{}{
			&pakExtraFileId, &pakExtraFileEntityId, &pakExtraFileType, &pakExtraFileUrl, &pakExtraFileMime, &pakExtraFileSize, &pakExtraFileVersion, &pakExtraFileDeployment, &pakExtraFilePlatform, &pakExtraFileUploadedBy, &pakExtraFileCreatedAt, &pakExtraFileUpdatedAt, &pakExtraFileVariation, &pakExtraFileOriginalPath, &pakExtraFileHash,
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
			if request.Options.Likes {
				allFields = append(allFields, likeFields...)
			}
			if request.Options.Preview {
				allFields = append(allFields, previewFields...)
			}
			if request.Options.Pak {
				allFields = append(allFields, packageFields...)
				allFields = append(allFields, pakFileFields...)
				allFields = append(allFields, pakExtraFileFields...)
			}
			if request.Options.Owner {
				allFields = append(allFields, ownerFields...)
			}
		}

		err = rows.Scan(allFields...)
		if err != nil {
			return nil, err
		}

		if id.Status != pgtype.Present {
			continue
		}

		if pakFileId.Status == pgtype.Present {
			pakFile = &File{}
			pakFile.Id = pakFileId.UUID
			if pakFileType.Status == pgtype.Present {
				pakFile.Type = pakFileType.String
			}
			if pakFileMime.Status == pgtype.Present {
				pakFile.Mime = &pakFileMime.String
			}
			if pakFileUrl.Status == pgtype.Present {
				pakFile.Url = pakFileUrl.String
			}
			if pakFileSize.Status == pgtype.Present {
				pakFile.Size = &pakFileSize.Int
			}
			if pakFileVersion.Status == pgtype.Present {
				pakFile.Version = pakFileVersion.Int
			}
			if pakFileDeployment.Status == pgtype.Present {
				pakFile.Deployment = pakFileDeployment.String
			}
			if pakFilePlatform.Status == pgtype.Present {
				pakFile.Platform = pakFilePlatform.String
			}
			if pakFileUploadedBy.Status == pgtype.Present {
				pakFile.UploadedBy = &pakFileUploadedBy.UUID
			}
			if pakFileCreatedAt.Status == pgtype.Present {
				pakFile.CreatedAt = pakFileCreatedAt.Time
			}
			if pakFileUpdatedAt.Status == pgtype.Present {
				pakFile.UpdatedAt = &pakFileUpdatedAt.Time
			}
			if pakFileVariation.Status == pgtype.Present {
				pakFile.Index = pakFileVariation.Int
			}
			if pakFileOriginalPath.Status == pgtype.Present {
				pakFile.OriginalPath = &pakFileOriginalPath.String
			}
			if pakFileHash.Status == pgtype.Present {
				pakFile.Hash = &pakFileHash.String
			}
		}

		if pakExtraFileId.Status == pgtype.Present {
			pakExtraFile = &File{}
			pakExtraFile.Id = pakExtraFileId.UUID
			if pakExtraFileType.Status == pgtype.Present {
				pakExtraFile.Type = pakExtraFileType.String
			}
			if pakExtraFileMime.Status == pgtype.Present {
				pakExtraFile.Mime = &pakExtraFileMime.String
			}
			if pakExtraFileUrl.Status == pgtype.Present {
				pakExtraFile.Url = pakExtraFileUrl.String
			}
			if pakExtraFileSize.Status == pgtype.Present {
				pakExtraFile.Size = &pakExtraFileSize.Int
			}
			if pakExtraFileVersion.Status == pgtype.Present {
				pakExtraFile.Version = pakExtraFileVersion.Int
			}
			if pakExtraFileDeployment.Status == pgtype.Present {
				pakExtraFile.Deployment = pakExtraFileDeployment.String
			}
			if pakExtraFilePlatform.Status == pgtype.Present {
				pakExtraFile.Platform = pakExtraFilePlatform.String
			}
			if pakExtraFileUploadedBy.Status == pgtype.Present {
				pakExtraFile.UploadedBy = &pakExtraFileUploadedBy.UUID
			}
			if pakExtraFileCreatedAt.Status == pgtype.Present {
				pakExtraFile.CreatedAt = pakExtraFileCreatedAt.Time
			}
			if pakExtraFileUpdatedAt.Status == pgtype.Present {
				pakExtraFile.UpdatedAt = &pakExtraFileUpdatedAt.Time
			}
			if pakExtraFileVariation.Status == pgtype.Present {
				pakExtraFile.Index = pakExtraFileVariation.Int
			}
			if pakExtraFileOriginalPath.Status == pgtype.Present {
				pakExtraFile.OriginalPath = &pakExtraFileOriginalPath.String
			}
			if pakExtraFileHash.Status == pgtype.Present {
				pakExtraFile.Hash = &pakExtraFileHash.String
			}
		}

		// preview file
		if previewFileId.Status == pgtype.Present {
			previewFile = &File{}
			previewFile.Id = previewFileId.UUID
			if previewFileType.Status == pgtype.Present {
				previewFile.Type = previewFileType.String
			}
			if previewFileMime.Status == pgtype.Present {
				previewFile.Mime = &previewFileMime.String
			}
			if previewFileUrl.Status == pgtype.Present {
				previewFile.Url = previewFileUrl.String
			}
			if previewFileSize.Status == pgtype.Present {
				previewFile.Size = &previewFileSize.Int
			}
			if previewFileVersion.Status == pgtype.Present {
				previewFile.Version = previewFileVersion.Int
			}
			if previewFileDeployment.Status == pgtype.Present {
				previewFile.Deployment = previewFileDeployment.String
			}
			if previewFilePlatform.Status == pgtype.Present {
				previewFile.Platform = previewFilePlatform.String
			}
			if previewFileUploadedBy.Status == pgtype.Present {
				previewFile.UploadedBy = &previewFileUploadedBy.UUID
			}
			if previewFileCreatedAt.Status == pgtype.Present {
				previewFile.CreatedAt = previewFileCreatedAt.Time
			}
			if previewFileUpdatedAt.Status == pgtype.Present {
				previewFile.UpdatedAt = &previewFileUpdatedAt.Time
			}
			if previewFileVariation.Status == pgtype.Present {
				previewFile.Index = previewFileVariation.Int
			}
			if previewFileOriginalPath.Status == pgtype.Present {
				previewFile.OriginalPath = &previewFileOriginalPath.String
			}
			if previewFileHash.Status == pgtype.Present {
				previewFile.Hash = &previewFileHash.String
			}
		}
		if world == nil {
			world = &World{}
			world.Id = id.UUID
			if createdAt.Status == pgtype.Present {
				world.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				world.UpdatedAt = &updatedAt.Time
			}
			if entityType.Status == pgtype.Present {
				world.EntityType = entityType.String
			}
			if views.Status == pgtype.Present {
				world.Views = views.Int
			}
			if public.Status == pgtype.Present {
				world.Public = public.Bool
			}
			if likeCount.Status == pgtype.Present {
				world.Likes = &likeCount.Int
			}
			if dislikeCount.Status == pgtype.Present {
				world.Dislikes = &dislikeCount.Int
			}
			if likedByRequester.Status == pgtype.Present {
				world.Liked = &likedByRequester.Int
			}
			// world data
			if name.Status == pgtype.Present {
				world.Name = name.String
			}
			if description.Status == pgtype.Present {
				world.Description = &description.String
			}
			if worldMap.Status == pgtype.Present {
				world.Map = worldMap.String
			}
			if modId.Status == pgtype.Present {
				world.PackageId = modId.UUID
			}
			if worldType.Status == pgtype.Present {
				world.Type = worldType.String
			}
			if scheduled.Status == pgtype.Present {
				world.Scheduled = scheduled.Bool
			}
			if gameMode.Status == pgtype.Present {
				world.GameMode = gameMode.String
			}
			if previewFile != nil {
				world.Files = &FileBatch{}
				world.Files.Entities = append(world.Files.Entities, *previewFile)
			}
			// package
			if packageId.Status == pgtype.Present {
				pack.Id = packageId.UUID
				if packageName.Status == pgtype.Present {
					pack.Name = packageName.String
				}
				if packageDescription.Status == pgtype.Present {
					pack.Description = packageDescription.String
				}
				if pakFile != nil {
					if pack.Files == nil {
						pack.Files = &FileBatch{}
					}
					pack.Files.Entities = append(pack.Files.Entities, *pakFile)
				}
				if pakExtraFile != nil {
					if pack.Files == nil {
						pack.Files = &FileBatch{}
					}
					pack.Files.Entities = append(pack.Files.Entities, *pakExtraFile)
				}
				world.Package = &pack
			}
			// owner
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
				world.Owner = &owner
			}
		} else {
			if world.Files == nil {
				world.Files = &FileBatch{}
			}
			if previewFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(world.Files.Entities), previewFile.Id) {
				world.Files.Entities = append(world.Files.Entities, *previewFile)
			}
			if world.Package != nil && world.Package.Files == nil {
				world.Package.Files = &FileBatch{}
			}
			if world.Package != nil && pakFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(world.Package.Files.Entities), pakFile.Id) {
				world.Package.Files.Entities = append(world.Package.Files.Entities, *pakFile)
			}
			if world.Package != nil && pakExtraFile != nil && !ContainsIdentifiable(helper.ToSliceOfAny(world.Package.Files.Entities), pakExtraFile.Id) {
				world.Package.Files.Entities = append(world.Package.Files.Entities, *pakExtraFile)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return world, nil
}
