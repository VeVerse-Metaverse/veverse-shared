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

var SupportedJobV2Types = map[string]bool{
	"release": true, // App release building and deployment
	"package": true, // Package processing
}

var SupportedJobV2Deployments = map[string]bool{
	"client":                   true,
	"server":                   true,
	"editor":                   true,
	"launcher":                 true,
	"server-launcher":          true,
	"pixel-streaming-launcher": true,
}

var SupportedJobV2Statuses = map[string]bool{
	"unclaimed":  true, // Job is scheduled and not claimed by any worker, initial state
	"claimed":    true, // Job is claimed by a worker, processing state
	"processing": true, // Job is currently processed by a worker, processing state
	"uploading":  true, // Job is currently uploading its results to the cloud storage, processing state
	"completed":  true, // Job has been completed successfully, final state
	"failed":     true, // Job failed with an error message, final state
	"cancelled":  true, // Job has been cancelled, final state
}

// JobV2 is the model for a automation job used by build system.
type JobV2 struct {
	Identifier
	Timestamps
	EntityId      uuid.UUID `json:"entityId"`
	OwnerId       uuid.UUID `json:"ownerId"`
	WorkerId      uuid.UUID `json:"workerId"`
	Configuration string    `json:"configuration"`
	Platform      string    `json:"platform"`
	Type          string    `json:"type"`
	Target        string    `json:"target"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	Version       int64     `json:"version"`

	App     *AppV2     `json:"app,omitempty"`
	Package *Package   `json:"package,omitempty"`
	Release *ReleaseV2 `json:"release,omitempty"`
}

// CreateJobV2Request is the request body for the CreateJob handler, used to create a new job.
type CreateJobV2Request struct {
	Configuration string    `json:"configuration"` // Job configuration (Development, Test, Shipping, etc.)
	Target        string    `json:"target"`        // Job target (Client, Server, Editor, Launcher, etc.)
	Platform      string    `json:"platform"`      // Job platform (Windows, Linux, Android, iOS, etc.)
	Type          string    `json:"type"`          // Job type (Release, Package)
	EntityId      uuid.UUID `json:"entityId"`      // Entity ID (AppV2, Package, Release)
}

func CreateJobV2(ctx context.Context, requester *User, request CreateJobV2Request) (Job *JobV2, err error) {
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

	if request.EntityId.IsNil() {
		return nil, fmt.Errorf("no job entity id")
	}

	if request.Configuration == "" {
		return nil, fmt.Errorf("no job configuration")
	}

	if request.Platform == "" {
		return nil, fmt.Errorf("no job platform")
	}

	if request.Type == "" {
		return nil, fmt.Errorf("no job type")
	}

	if request.Target == "" {
		return nil, fmt.Errorf("no job deployment")
	}

	var (
		q  string
		tx pgx.Tx
	)

	// Start a transaction
	tx, err = db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Find unfinished jobs for the same entity.
	q = `select id from jobs j where (j.status != 'completed' or j.status != 'failed' or j.status != 'cancelled') and j.entity_id = $1 and j.platform = $2 and j.type = $3 and j.deployment = $4 and j.configuration = $5;`
	row := tx.QueryRow(ctx, q, request.EntityId, request.Platform, request.Type, request.Target, request.Configuration)
	job := &JobV2{}
	err = row.Scan(&job.Id)
	if err == nil {
		// job already exists, return it
		return job, nil
	} else if err == pgx.ErrNoRows {
		// no rows, create a new job

	} else {
		// error
	}

	q = `with r
         as (insert into entities (id, created_at, updated_at, entity_type, public, views) values (gen_random_uuid(), now(), null, 'launcher-v2', true, null) returning id)
insert
into launcher_v2 (id, name)
select r.id, $1
from r
returning id;`

	//row := db.QueryRow(ctx, q, request.Name)
	//launcher = &LauncherV2{}
	//err = row.Scan(&launcher.Id)
	//if err != nil {
	//	return nil, err
	//}
	//
	//return GetLauncherV2(ctx, requester, launcher.Id, "")

	return nil, nil
}

// IndexJobV2Request is the request body for the IndexJobs handler, used to get a list of jobs.
type IndexJobV2Request struct {
	Offset     *int64  `json:"offset,omitempty"`
	Limit      *int64  `json:"limit,omitempty"`
	Platform   *string `json:"platform,omitempty"`
	Type       *string `json:"type,omitempty"`
	Deployment *string `json:"deployment,omitempty"`
}

func IndexJobV2(ctx context.Context, requester *User, request IndexLauncherV2Request) (entities *LauncherV2Batch, err error) {
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

// GetJobV2StatusRequest is the request body for the GetJobStatus handler, used to get the status of a single job.
type GetJobV2StatusRequest struct {
	JobId string `json:"jobId"`
}

// UpdateJobV2StatusRequest is the request body for the UpdateJobStatus handler, used to update the status of a single job.
type UpdateJobV2StatusRequest struct {
	JobId   string `json:"jobId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
