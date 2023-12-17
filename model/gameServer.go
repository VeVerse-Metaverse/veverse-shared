package model

import (
	"context"
	sc "dev.hackerman.me/artheon/veverse-shared/context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	pgtypeuuid "github.com/jackc/pgtype/ext/gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	GameServerReservedSlots = 3
)

//goland:noinspection GoUnusedConst
const (
	GameServerTypeOfficial  = "official"
	GameServerTypeCommunity = "community"
)

// GameServerStatus enum, community servers use only created, online and offline statuses
const (
	GameServerV2StatusCreated     = "created"     // created by the API request
	GameServerV2StatusLaunching   = "launching"   // processed by the operator, created the deployment and service, waiting for the server launcher to prepare the server
	GameServerV2StatusDownloading = "downloading" // launcher is downloading the server files to the cache directory
	GameServerV2StatusStarting    = "starting"    // launcher is starting the server (moves the files from the cache directory to the runtime directory and starts the server binary)
	GameServerV2StatusOnline      = "online"      // server is running and sending heartbeats
	GameServerV2StatusOffline     = "offline"     // launcher detected that the server has been shut down successfully
	GameServerV2StatusError       = "error"       // launcher detected that the server has been shut down unexpectedly (e.g. crashed)
)

const (
	GameServerV2PlayerStatusConnected    = "connected"
	GameServerV2PlayerStatusDisconnected = "disconnected"
)

// ValidGameServerV2Statuses List of all known and valid game server statuses (for API request validation)
var ValidGameServerV2Statuses = []string{
	GameServerV2StatusCreated,
	GameServerV2StatusLaunching,
	GameServerV2StatusDownloading,
	GameServerV2StatusStarting,
	GameServerV2StatusOnline,
	GameServerV2StatusOffline,
	GameServerV2StatusError,
}

// ValidGameServerV2PlayerStatuses List of all known and valid game server player statuses (for API request validation)
var ValidGameServerV2PlayerStatuses = []string{
	GameServerV2PlayerStatusConnected,
	GameServerV2PlayerStatusDisconnected,
}

type GameServerV2 struct {
	Entity

	// Server region id
	RegionId uuid.UUID `json:"regionId,omitempty"`

	// Optionally included full region
	Region *Region `json:"region,omitempty"`

	// Server release id
	ReleaseId uuid.UUID `json:"releaseId,omitempty"`

	// Optionally included full release
	Release *ReleaseV2 `json:"release,omitempty"`

	// Server world id
	WorldId uuid.UUID `json:"worldId,omitempty"`

	// Optionally included full world
	World *World `json:"world"`

	// Server game mode id
	GameModeId uuid.UUID `json:"gameModeId,omitempty"`

	// Optionally included full game mode
	GameMode *GameMode `json:"gameMode"`

	// Server type (official, community, etc)
	Type string `json:"type"`

	// Server host
	Host string `json:"host"`

	// Server port
	Port int32 `json:"port"`

	// Max players allowed on the server
	MaxPlayers int32 `json:"maxPlayers"`

	// Status of the server
	Status string `json:"status"`

	// Status message (error message, etc)
	StatusMessage string `json:"statusMessage"`
}

func (s *GameServerV2) ToUnstructured(ctx context.Context) (unstructured.Unstructured, error) {
	environment, ok := ctx.Value(sc.Environment).(string)
	if !ok || environment == "" {
		return unstructured.Unstructured{}, errors.New("environment not set in context")
	}

	apiV1Token, ok := ctx.Value(sc.GameServerApiV1Token).(string)
	if !ok || apiV1Token == "" {
		return unstructured.Unstructured{}, errors.New("api v1 token not set in context")
	}

	apiV2Token, ok := ctx.Value(sc.GameServerApiV2Token).(string)
	if !ok || apiV2Token == "" {
		return unstructured.Unstructured{}, errors.New("api v2 token not set in context")
	}

	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "veverse.com/v1",
			"kind":       "GameServer",
			"metadata": map[string]interface{}{
				"name": fmt.Sprintf("gs-%s", s.Id),
			},
			"spec": map[string]interface{}{
				"id": s.Id.String(),
				"settings": map[string]interface{}{
					"api": map[string]interface{}{
						"v1": map[string]interface{}{
							"url":   fmt.Sprintf("https://%s.api.veverse.com", environment),
							"token": apiV1Token,
						},
						"v2": map[string]interface{}{
							"url":   fmt.Sprintf("https://%s.api2.veverse.com/v2", environment),
							"token": apiV2Token,
						},
					},
					"appId":      s.Release.App.Id.String(),
					"releaseId":  s.ReleaseId.String(),
					"worldId":    s.WorldId.String(),
					"gameModeId": s.GameModeId.String(),
					"regionId":   s.RegionId.String(),
					"public":     s.Public,
					"maxPlayers": s.MaxPlayers,
					"reservedSlots": map[string]interface{}{
						"enabled": true,
						"count":   GameServerReservedSlots,
					},
				},
			},
		},
	}, nil
}

type GameServerV2Batch Batch[GameServerV2]

// IndexGameServersV2 returns a batch of game servers by app id and version.
//
//goland:noinspection GoUnusedExportedFunction
func IndexGameServersV2(ctx context.Context, requester *User, releaseId uuid.UUID, offset, limit int64) (entities GameServerV2Batch, err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
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
		q = `select e.id,
       e.created_at,
       e.updated_at,
       e.public,
       gs.type,
       gs.host,
       gs.port,
       pc.num_players,
       gs.max_players,
       gs.status,
       gs.status_message,
       gs.region_id,
       r.name,
       gs.release_id,
       r2e.created_at,
       r2e.updated_at,
       r2e.public,
       r2.name,
       r2.description,
       r2.version,
       r2.code_version,
       r2.content_version,
       r2.archive,
       a.id,
       ae.created_at,
       ae.updated_at,
       ae.public,
       a.name,
       a.description,
       a.external,
       gs.world_id,
       we.created_at,
       we.updated_at,
       we.public,
       w.name,
       w.description,
       w.map,
       w.mod_id,
       gs.game_mode_id,
       gme.created_at,
       gme.updated_at,
       gme.public,
       gm.name,
       gm.path
from game_server_v2 gs
         left join entities e on gs.id = e.id
         left join region r on gs.region_id = r.id
         left join release_v2 r2 on gs.release_id = r2.id
         left join entities r2e on r2.id = r2e.id
         left join app_v2 a on r2.entity_id = a.id
         left join entities ae on a.id = ae.id
         left join spaces w on gs.world_id = w.id
         left join entities we on w.id = we.id
         left join game_mode gm on gs.game_mode_id = gm.id
         left join entities gme on gm.id = gme.id
         left join (select server_id,
                           count(*) as num_players
                    from game_server_player_v2
                    where (status = 'connected' or status = 'connecting')
                      and updated_at > now() - interval '1 minutes' -- filter by connected players
                      -- and server_id = $1 -- todo: how to do this?
                    group by server_id) as pc on gs.id = pc.server_id
where r2.id = $1
order by e.updated_at desc
offset $2 limit $3`
		rows, err = db.Query(ctx, q, releaseId, offset, limit)
	} else {
		q = `select e.id,
       e.created_at,
       e.updated_at,
       e.public,
       gs.type,
       gs.host,
       gs.port,
       pc.num_players,
       gs.max_players,
       gs.status,
       gs.status_message,
       gs.region_id,
       r.name,
       gs.release_id,
       r2e.created_at,
       r2e.updated_at,
       r2e.public,
       r2.name,
       r2.description,
       r2.version,
       r2.code_version,
       r2.content_version,
       r2.archive,
       a.id,
       ae.created_at,
       ae.updated_at,
       ae.public,
       a.name,
       a.description,
       a.external,
       gs.world_id,
       we.created_at,
       we.updated_at,
       we.public,
       w.name,
       w.description,
       w.map,
       w.mod_id,
       gs.game_mode_id,
       gme.created_at,
       gme.updated_at,
       gme.public,
       gm.name,
       gm.path
from game_server_v2 gs
         left join entities e on gs.id = e.id
         left join accessibles ea on e.id = ea.entity_id and ea.user_id = $1
         left join region r on gs.region_id = r.id
         left join release_v2 r2 on gs.release_id = r2.id
         left join entities r2e on r2.id = r2e.id
         left join accessibles r2a on r2.id = r2a.entity_id and r2a.user_id = $1
         left join app_v2 a on r2.entity_id = a.id
         left join entities ae on a.id = ae.id
         left join accessibles aa on a.id = aa.entity_id and aa.user_id = $1
         left join spaces w on gs.world_id = w.id
         left join entities we on w.id = we.id
         left join accessibles wa on w.id = wa.entity_id and wa.user_id = $1
         left join game_mode gm on gs.game_mode_id = gm.id
         left join entities gme on gm.id = gme.id
         left join accessibles gma on gm.id = gma.entity_id and gma.user_id = $1
         left join (select server_id,
                           count(*) as num_players
                    from game_server_player_v2
                    where (status = 'connected' or status = 'connecting')
                      and updated_at > now() - interval '1 minutes' -- filter by connected players
                      -- and server_id = $1 -- todo: how to do this?
                    group by server_id) as pc on gs.id = pc.server_id
where r2.id = $2
  and (e.public = true or ea.is_owner or ea.can_view)
  and (r2e.public = true or r2a.is_owner or r2a.can_view)
  and (ae.public = true or aa.is_owner or aa.can_view)
  and (we.public = true or wa.is_owner or wa.can_view)
  and (gme.public = true or gma.is_owner or gma.can_view)
order by e.updated_at desc
offset $3 limit $4`
		rows, err = db.Query(ctx, q, requester.Id, releaseId, offset, limit)
	}

	if err != nil {
		err = fmt.Errorf("failed to query game servers: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			id                    pgtypeuuid.UUID
			createdAt             pgtype.Timestamptz
			updatedAt             pgtype.Timestamptz
			public                pgtype.Bool
			serverType            pgtypeuuid.UUID
			host                  pgtype.Text
			port                  pgtype.Int4
			numPlayers            pgtype.Int4
			maxPlayers            pgtype.Int4
			status                pgtype.Text
			statusMessage         pgtype.Text
			regionId              pgtypeuuid.UUID
			regionName            pgtype.Text
			releaseId             pgtypeuuid.UUID
			releaseCreatedAt      pgtype.Timestamptz
			releaseUpdatedAt      pgtype.Timestamptz
			releasePublic         pgtype.Bool
			releaseName           pgtype.Text
			releaseDescription    pgtype.Text
			releaseVersion        pgtype.Text
			releaseCodeVersion    pgtype.Text
			releaseContentVersion pgtype.Text
			releaseArchive        pgtype.Bool
			appId                 pgtypeuuid.UUID
			appCreatedAt          pgtype.Timestamptz
			appUpdatedAt          pgtype.Timestamptz
			appPublic             pgtype.Bool
			appName               pgtype.Text
			appDescription        pgtype.Text
			appExternal           pgtype.Bool
			worldId               pgtypeuuid.UUID
			worldCreatedAt        pgtype.Timestamptz
			worldUpdatedAt        pgtype.Timestamptz
			worldPublic           pgtype.Bool
			worldName             pgtype.Text
			worldDescription      pgtype.Text
			worldMap              pgtype.Text
			worldPackageId        pgtypeuuid.UUID
			gameModeId            pgtypeuuid.UUID
			gameModeName          pgtype.Text
			gameModePath          pgtype.Text
		)

		err = rows.Scan(&id,
			&createdAt,
			&updatedAt,
			&public,
			&serverType,
			&host,
			&port,
			&numPlayers,
			&maxPlayers,
			&status,
			&statusMessage,
			&regionId,
			&regionName,
			&releaseId,
			&releaseCreatedAt,
			&releaseUpdatedAt,
			&releasePublic,
			&releaseName,
			&releaseDescription,
			&releaseVersion,
			&releaseCodeVersion,
			&releaseContentVersion,
			&releaseArchive,
			&appId,
			&appCreatedAt,
			&appUpdatedAt,
			&appPublic,
			&appName,
			&appDescription,
			&appExternal,
			&worldId,
			&worldCreatedAt,
			&worldUpdatedAt,
			&worldPublic,
			&worldName,
			&worldDescription,
			&worldMap,
			&worldPackageId,
			&gameModeId,
			&gameModeName,
			&gameModePath,
		)
		if err != nil {
			err = fmt.Errorf("failed to scan game server: %v", err)
			return
		}

		rowIndex++

		if id.Status != pgtype.Present {
			continue
		}

		var region *Region
		if regionId.Status == pgtype.Present {
			region = &Region{}
			region.Id = regionId.UUID
			if regionName.Status == pgtype.Present {
				region.Name = regionName.String
			}
		}

		var release *ReleaseV2
		if releaseId.Status == pgtype.Present {
			release = &ReleaseV2{}
			release.Id = releaseId.UUID
			if releaseCreatedAt.Status == pgtype.Present {
				release.CreatedAt = releaseCreatedAt.Time
			}
			if releaseUpdatedAt.Status == pgtype.Present {
				release.UpdatedAt = &releaseUpdatedAt.Time
			}
			if releasePublic.Status == pgtype.Present {
				release.Public = releasePublic.Bool
			}
			if releaseName.Status == pgtype.Present {
				release.Name = &releaseName.String
			}
			if releaseDescription.Status == pgtype.Present {
				release.Description = &releaseDescription.String
			}
			if releaseVersion.Status == pgtype.Present {
				release.Version = releaseVersion.String
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

		var app *AppV2
		if appId.Status == pgtype.Present {
			app = &AppV2{}
			app.Id = appId.UUID
			if appCreatedAt.Status == pgtype.Present {
				app.CreatedAt = appCreatedAt.Time
			}
			if appUpdatedAt.Status == pgtype.Present {
				app.UpdatedAt = &appUpdatedAt.Time
			}
			if appPublic.Status == pgtype.Present {
				app.Public = appPublic.Bool
			}
			if appName.Status == pgtype.Present {
				app.Name = appName.String
			}
			if appDescription.Status == pgtype.Present {
				app.Description = &appDescription.String
			}
			if appExternal.Status == pgtype.Present {
				app.External = appExternal.Bool
			}
		}

		var world *World
		world = &World{}
		if worldId.Status == pgtype.Present {
			world.Id = worldId.UUID
			if worldCreatedAt.Status == pgtype.Present {
				world.CreatedAt = worldCreatedAt.Time
			}
			if worldUpdatedAt.Status == pgtype.Present {
				world.UpdatedAt = &worldUpdatedAt.Time
			}
			if worldName.Status == pgtype.Present {
				world.Name = worldName.String
			}
			if worldDescription.Status == pgtype.Present {
				world.Description = &worldDescription.String
			}
			if worldMap.Status == pgtype.Present {
				world.Map = worldMap.String
			}
			if worldPackageId.Status == pgtype.Present {
				world.PackageId = worldPackageId.UUID
			}
		}

		var gameMode *GameMode
		gameMode = &GameMode{}
		if gameModeId.Status == pgtype.Present {
			gameMode.Id = gameModeId.UUID
			if gameModeName.Status == pgtype.Present {
				gameMode.Name = gameModeName.String
			}
			if gameModePath.Status == pgtype.Present {
				gameMode.Path = gameModePath.String
			}
		}

		if skipEntity {
			// skip the entity if we previously skipped it because of the offset
			if skippedEntityId == id.UUID {
				continue
			}
		}

		// skip the entity until we reach the offset
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

		e := GameServerV2{}
		e.Id = id.UUID
		if createdAt.Status == pgtype.Present {
			e.CreatedAt = createdAt.Time
		}
		if updatedAt.Status == pgtype.Present {
			e.UpdatedAt = &updatedAt.Time
		}
		if public.Status == pgtype.Present {
			e.Public = public.Bool
		}
		if public.Status == pgtype.Present {
			e.Public = public.Bool
		}
		if host.Status == pgtype.Present {
			e.Host = host.String
		}
		if port.Status == pgtype.Present {
			e.Port = port.Int
		}
		e.RegionId = regionId.UUID
		e.Region = region
		e.ReleaseId = releaseId.UUID
		e.Release = release
		e.Release.App = app
		e.WorldId = worldId.UUID
		e.World = world
		e.GameModeId = gameModeId.UUID
		e.GameMode = gameMode

		entities.Entities = append(entities.Entities, e)
		skipEntity = false
		entityIndex++
	}

	return
}

// GetGameServerV2 returns a game server by id.
func GetGameServerV2(ctx context.Context, requester *User, id uuid.UUID) (e *GameServerV2, err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	var (
		q    string
		rows pgx.Rows
	)

	if requester.IsAdmin {
		q = `select e.id,
       e.created_at,
       e.updated_at,
       e.public,
       gs.type,
       gs.host,
       gs.port,
       pc.num_players,
       gs.max_players,
       gs.status,
       gs.status_message,
       gs.region_id,
       r.name,
       gs.release_id,
       r2e.created_at,
       r2e.updated_at,
       r2e.public,
       r2.name,
       r2.description,
       r2.version,
       r2.code_version,
       r2.content_version,
       r2.archive,
       a.id,
       ae.created_at,
       ae.updated_at,
       ae.public,
       a.name,
       a.description,
       a.external,
       gs.world_id,
       we.created_at,
       we.updated_at,
       we.public,
       w.name,
       w.description,
       w.map,
       w.mod_id,
       gs.game_mode_id,
       gme.created_at,
       gme.updated_at,
       gme.public,
       gm.name,
       gm.path
from game_server_v2 gs
         left join entities e on gs.id = e.id
         left join region r on gs.region_id = r.id
         left join release_v2 r2 on gs.release_id = r2.id
         left join entities r2e on r2.id = r2e.id
         left join app_v2 a on r2.entity_id = a.id
         left join entities ae on a.id = ae.id
         left join spaces w on gs.world_id = w.id
         left join entities we on w.id = we.id
         left join game_mode gm on gs.game_mode_id = gm.id
         left join entities gme on gm.id = gme.id
         left join (select server_id,
                           count(*) as num_players
                    from game_server_player_v2
                    where (status = 'connected' or status = 'connecting')
                      and updated_at > now() - interval '1 minutes' -- filter by connected players
                      and server_id = $1
                    group by server_id) as pc on gs.id = pc.server_id
where e.id = $1`
		rows, err = db.Query(ctx, q, id)
	} else {
		q = `select e.id,
       e.created_at,
       e.updated_at,
       e.public,
       gs.type,
       gs.host,
       gs.port,
       pc.num_players,
       gs.max_players,
       gs.status,
       gs.status_message,
       gs.region_id,
       r.name,
       gs.release_id,
       r2e.created_at,
       r2e.updated_at,
       r2e.public,
       r2.name,
       r2.description,
       r2.version,
       r2.code_version,
       r2.content_version,
       r2.archive,
       a.id,
       ae.created_at,
       ae.updated_at,
       ae.public,
       a.name,
       a.description,
       a.external,
       gs.world_id,
       we.created_at,
       we.updated_at,
       we.public,
       w.name,
       w.description,
       w.map,
       w.mod_id,
       gs.game_mode_id,
       gme.created_at,
       gme.updated_at,
       gme.public,
       gm.name,
       gm.path
from game_server_v2 gs
         left join entities e on gs.id = e.id
         left join accessibles ea on e.id = ea.entity_id and ea.user_id = $1
         left join region r on gs.region_id = r.id
         left join release_v2 r2 on gs.release_id = r2.id
         left join entities r2e on r2.id = r2e.id
         left join accessibles r2a on r2.id = r2a.entity_id and r2a.user_id = $1
         left join app_v2 a on r2.entity_id = a.id
         left join entities ae on a.id = ae.id
         left join accessibles aa on a.id = aa.entity_id and aa.user_id = $1
         left join spaces w on gs.world_id = w.id
         left join entities we on w.id = we.id
         left join accessibles wa on w.id = wa.entity_id and wa.user_id = $1
         left join game_mode gm on gs.game_mode_id = gm.id
         left join entities gme on gm.id = gme.id
         left join accessibles gma on gm.id = gma.entity_id and gma.user_id = $1
         left join (select server_id,
                           count(*) as num_players
                    from game_server_player_v2
                    where (status = 'connected' or status = 'connecting')
                      and updated_at > now() - interval '1 minutes' -- filter by connected players
                      and server_id = $2
                    group by server_id) as pc on gs.id = pc.server_id
where e.id = $2
  and (e.public = true or ea.is_owner or ea.can_view)
  and (r2e.public = true or r2a.is_owner or r2a.can_view)
  and (ae.public = true or aa.is_owner or aa.can_view)
  and (we.public = true or wa.is_owner or wa.can_view)
  and (gme.public = true or gma.is_owner or gma.can_view)`
		rows, err = db.Query(ctx, q, requester.Id, id)
	}

	if err != nil {
		err = errors.Wrap(err, "failed to query game server")
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			id                    pgtypeuuid.UUID
			createdAt             pgtype.Timestamptz
			updatedAt             pgtype.Timestamptz
			public                pgtype.Bool
			serverType            pgtype.Text
			host                  pgtype.Text
			port                  pgtype.Int4
			numPlayers            pgtype.Int4
			maxPlayers            pgtype.Int4
			status                pgtype.Text
			statusMessage         pgtype.Text
			regionId              pgtypeuuid.UUID
			regionName            pgtype.Text
			releaseId             pgtypeuuid.UUID
			releaseCreatedAt      pgtype.Timestamptz
			releaseUpdatedAt      pgtype.Timestamptz
			releasePublic         pgtype.Bool
			releaseName           pgtype.Text
			releaseDescription    pgtype.Text
			releaseVersion        pgtype.Text
			releaseCodeVersion    pgtype.Text
			releaseContentVersion pgtype.Text
			releaseArchive        pgtype.Bool
			appId                 pgtypeuuid.UUID
			appCreatedAt          pgtype.Timestamptz
			appUpdatedAt          pgtype.Timestamptz
			appPublic             pgtype.Bool
			appName               pgtype.Text
			appDescription        pgtype.Text
			appExternal           pgtype.Bool
			worldId               pgtypeuuid.UUID
			worldCreatedAt        pgtype.Timestamptz
			worldUpdatedAt        pgtype.Timestamptz
			worldPublic           pgtype.Bool
			worldName             pgtype.Text
			worldDescription      pgtype.Text
			worldMap              pgtype.Text
			worldPackageId        pgtypeuuid.UUID
			gameModeId            pgtypeuuid.UUID
			gameModeCreatedAt     pgtype.Timestamptz
			gameModeUpdatedAt     pgtype.Timestamptz
			gameModePublic        pgtype.Bool
			gameModeName          pgtype.Text
			gameModePath          pgtype.Text
		)

		err = rows.Scan(&id,
			&createdAt,
			&updatedAt,
			&public,
			&serverType,
			&host,
			&port,
			&numPlayers,
			&maxPlayers,
			&status,
			&statusMessage,
			&regionId,
			&regionName,
			&releaseId,
			&releaseCreatedAt,
			&releaseUpdatedAt,
			&releasePublic,
			&releaseName,
			&releaseDescription,
			&releaseVersion,
			&releaseCodeVersion,
			&releaseContentVersion,
			&releaseArchive,
			&appId,
			&appCreatedAt,
			&appUpdatedAt,
			&appPublic,
			&appName,
			&appDescription,
			&appExternal,
			&worldId,
			&worldCreatedAt,
			&worldUpdatedAt,
			&worldPublic,
			&worldName,
			&worldDescription,
			&worldMap,
			&worldPackageId,
			&gameModeId,
			&gameModeCreatedAt,
			&gameModeUpdatedAt,
			&gameModePublic,
			&gameModeName,
			&gameModePath,
		)
		if err != nil {
			err = errors.Wrap(err, "failed to scan game server")
			return
		}

		if id.Status != pgtype.Present {
			continue
		}

		if e == nil {
			e = &GameServerV2{}
			e.Id = id.UUID
			if createdAt.Status == pgtype.Present {
				e.CreatedAt = createdAt.Time
			}
			if updatedAt.Status == pgtype.Present {
				e.UpdatedAt = &updatedAt.Time
			}
			if public.Status == pgtype.Present {
				e.Public = public.Bool
			}
			if serverType.Status == pgtype.Present {
				e.Type = serverType.String
			}
			if host.Status == pgtype.Present {
				e.Host = host.String
			}
			if port.Status == pgtype.Present {
				e.Port = port.Int
			}
			if maxPlayers.Status == pgtype.Present {
				e.MaxPlayers = maxPlayers.Int
			}
			if status.Status == pgtype.Present {
				e.Status = status.String
			}
			if statusMessage.Status == pgtype.Present {
				e.StatusMessage = statusMessage.String
			}
		}

		var region *Region
		if regionId.Status == pgtype.Present {
			e.RegionId = regionId.UUID
			region = &Region{}
			region.Id = regionId.UUID
			if regionName.Status == pgtype.Present {
				region.Name = regionName.String
			}
		}

		var release *ReleaseV2
		if releaseId.Status == pgtype.Present {
			e.ReleaseId = releaseId.UUID
			release = &ReleaseV2{}
			release.Id = releaseId.UUID
			if releaseCreatedAt.Status == pgtype.Present {
				release.CreatedAt = releaseCreatedAt.Time
			}
			if releaseUpdatedAt.Status == pgtype.Present {
				release.UpdatedAt = &releaseUpdatedAt.Time
			}
			if releasePublic.Status == pgtype.Present {
				release.Public = releasePublic.Bool
			}
			if releaseName.Status == pgtype.Present {
				release.Name = &releaseName.String
			}
			if releaseDescription.Status == pgtype.Present {
				release.Description = &releaseDescription.String
			}
			if releaseVersion.Status == pgtype.Present {
				release.Version = releaseVersion.String
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

		var app *AppV2
		if appId.Status == pgtype.Present {
			app = &AppV2{}
			app.Id = appId.UUID
			if appCreatedAt.Status == pgtype.Present {
				app.CreatedAt = appCreatedAt.Time
			}
			if appUpdatedAt.Status == pgtype.Present {
				app.UpdatedAt = &appUpdatedAt.Time
			}
			if appPublic.Status == pgtype.Present {
				app.Public = appPublic.Bool
			}
			if appName.Status == pgtype.Present {
				app.Name = appName.String
			}
			if appDescription.Status == pgtype.Present {
				app.Description = &appDescription.String
			}
			if appExternal.Status == pgtype.Present {
				app.External = appExternal.Bool
			}
		}

		var world *World
		world = &World{}
		if worldId.Status == pgtype.Present {
			e.WorldId = worldId.UUID
			world.Id = worldId.UUID
			if worldCreatedAt.Status == pgtype.Present {
				world.CreatedAt = worldCreatedAt.Time
			}
			if worldUpdatedAt.Status == pgtype.Present {
				world.UpdatedAt = &worldUpdatedAt.Time
			}
			if worldName.Status == pgtype.Present {
				world.Name = worldName.String
			}
			if worldDescription.Status == pgtype.Present {
				world.Description = &worldDescription.String
			}
			if worldMap.Status == pgtype.Present {
				world.Map = worldMap.String
			}
			if worldPackageId.Status == pgtype.Present {
				world.PackageId = worldPackageId.UUID
			}
		}

		var gameMode *GameMode
		gameMode = &GameMode{}
		if gameModeId.Status == pgtype.Present {
			e.GameModeId = gameModeId.UUID
			gameMode.Id = gameModeId.UUID
			if gameModeName.Status == pgtype.Present {
				gameMode.Name = gameModeName.String
			}
			if gameModePath.Status == pgtype.Present {
				gameMode.Path = gameModePath.String
			}
			if gameModeCreatedAt.Status == pgtype.Present {
				gameMode.CreatedAt = gameModeCreatedAt.Time
			}
			if gameModeUpdatedAt.Status == pgtype.Present {
				gameMode.UpdatedAt = &gameModeUpdatedAt.Time
			}
			if gameModePublic.Status == pgtype.Present {
				gameMode.Public = gameModePublic.Bool
			}
		}

		e.Region = region
		e.Release = release
		e.Release.App = app
		e.World = world
		e.GameMode = gameMode
	}

	return
}

type FindGameServerV2Args struct {
	RegionId   uuid.UUID
	ReleaseId  uuid.UUID
	WorldId    uuid.UUID
	GameModeId *uuid.UUID
	Type       string
}

// FindGameServerV2 returns a game server that matches the given criteria.
func FindGameServerV2(ctx context.Context, requester *User, args FindGameServerV2Args) (e *GameServerV2, err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	var (
		q    string
		rows pgx.Rows
	)

	if requester.IsAdmin {
		q = `select e.id,
       e.created_at,
       e.updated_at,
       e.public,
       gs.type,
       gs.host,
       gs.port,
       pc.num_players,
       gs.max_players,
       gs.status,
       gs.status_message,
       gs.region_id,
       r.name,
       gs.release_id,
       r2e.created_at,
       r2e.updated_at,
       r2e.public,
       r2.name,
       r2.description,
       r2.version,
       r2.code_version,
       r2.content_version,
       r2.archive,
       a.id,
       ae.created_at,
       ae.updated_at,
       ae.public,
       a.name,
       a.description,
       a.external,
       gs.world_id,
       we.created_at,
       we.updated_at,
       we.public,
       w.name,
       w.description,
       w.map,
       w.mod_id,
       gs.game_mode_id,
       gme.created_at,
       gme.updated_at,
       gme.public,
       gm.name,
       gm.path
from game_server_v2 gs
         left join entities e on gs.id = e.id
         left join region r on gs.region_id = r.id
         left join release_v2 r2 on gs.release_id = r2.id
         left join entities r2e on r2.id = r2e.id
         left join app_v2 a on r2.entity_id = a.id
         left join entities ae on a.id = ae.id
         left join spaces w on gs.world_id = w.id
         left join entities we on w.id = we.id
         left join game_mode gm on gs.game_mode_id = gm.id
         left join entities gme on gm.id = gme.id
         left join (select server_id,
                           count(*) as num_players
                    from game_server_player_v2
                    where (status = 'connected' or status = 'connecting')
                      and updated_at > now() - interval '1 minutes' -- filter by connected players
                      -- and server_id = $1
                    group by server_id) as pc on gs.id = pc.server_id
where case when $1 != '00000000-0000-0000-0000-000000000000'::uuid then gs.region_id = $1 else true end -- region is optional 
  and gs.release_id = $2 -- release is required 
  and gs.world_id = $3 -- world is required
  and case when $4 != '00000000-0000-0000-0000-000000000000'::uuid then gs.game_mode_id = $4 else true end -- game mode is optional
  and gs.type = $5 -- type is required
  and pc.num_players < gs.max_players -- admin can join servers with reserved slots, but not full servers`
		rows, err = db.Query(ctx, q, args.RegionId, args.ReleaseId, args.WorldId, args.GameModeId, args.Type)
	} else {
		q = `select e.id,
       e.created_at,
       e.updated_at,
       e.public,
       gs.type,
       gs.host,
       gs.port,
       pc.num_players,
       gs.max_players,
       gs.status,
       gs.status_message,
       gs.region_id,
       r.name,
       gs.release_id,
       r2e.created_at,
       r2e.updated_at,
       r2e.public,
       r2.name,
       r2.description,
       r2.version,
       r2.code_version,
       r2.content_version,
       r2.archive,
       a.id,
       ae.created_at,
       ae.updated_at,
       ae.public,
       a.name,
       a.description,
       a.external,
       gs.world_id,
       we.created_at,
       we.updated_at,
       we.public,
       w.name,
       w.description,
       w.map,
       w.mod_id,
       gs.game_mode_id,
       gme.created_at,
       gme.updated_at,
       gme.public,
       gm.name,
       gm.path
from game_server_v2 gs
         left join entities e on gs.id = e.id
         left join accessibles ea on e.id = ea.entity_id and ea.user_id = $1
         left join region r on gs.region_id = r.id
         left join release_v2 r2 on gs.release_id = r2.id
         left join entities r2e on r2.id = r2e.id
         left join accessibles r2a on r2.id = r2a.entity_id and r2a.user_id = $1
         left join app_v2 a on r2.entity_id = a.id
         left join entities ae on a.id = ae.id
         left join accessibles aa on a.id = aa.entity_id and aa.user_id = $1
         left join spaces w on gs.world_id = w.id
         left join entities we on w.id = we.id
         left join accessibles wa on w.id = wa.entity_id and wa.user_id = $1
         left join game_mode gm on gs.game_mode_id = gm.id
         left join entities gme on gm.id = gme.id
         left join accessibles gma on gm.id = gma.entity_id and gma.user_id = $1
         left join (select server_id,
                           count(*) as num_players
                    from game_server_player_v2
                    where (status = 'connected' or status = 'connecting')
                      and updated_at > now() - interval '1 minutes' -- filter by connected players
                      -- and server_id = $2
                    group by server_id) as pc on gs.id = pc.server_id
where case when $2 != '00000000-0000-0000-0000-000000000000'::uuid then gs.region_id = $2 else true end -- region is optional 
  and gs.release_id = $3 -- release is required 
  and gs.world_id = $4 -- world is required
  and case when $5 != '00000000-0000-0000-0000-000000000000'::uuid then gs.game_mode_id = $5 else true end -- game mode is optional
  and (e.public = true or ea.is_owner or ea.can_view) -- requester must have access to the game server
  and (r2e.public = true or r2a.is_owner or r2a.can_view) -- requester must have access to the release
  and (ae.public = true or aa.is_owner or aa.can_view) -- requester must have access to the app
  and (we.public = true or wa.is_owner or wa.can_view) -- requester must have access to the world
  and (gme.public = true or gma.is_owner or gma.can_view) -- requester must have access to the game mode
  and gs.type = $6 -- type is required
  and ((pc.num_players < (gs.max_players - $7)) or pc.num_players is null) -- check for free slots available, $6 is the number of player slots to reserve`
		rows, err = db.Query(ctx, q, requester.Id, args.RegionId, args.ReleaseId, args.WorldId, args.GameModeId, args.Type, GameServerReservedSlots)
	}

	if err != nil {
		err = errors.Wrap(err, "failed to query game servers")
		return
	}

	defer rows.Close()

	err = rows.Err()
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNoRows
		}
	}

	for rows.Next() {
		var (
			id                    pgtypeuuid.UUID
			createdAt             pgtype.Timestamptz
			updatedAt             pgtype.Timestamptz
			public                pgtype.Bool
			serverType            pgtypeuuid.UUID
			host                  pgtype.Text
			port                  pgtype.Int4
			numPlayers            pgtype.Int4
			maxPlayers            pgtype.Int4
			status                pgtype.Text
			statusMessage         pgtype.Text
			regionId              pgtypeuuid.UUID
			regionName            pgtype.Text
			releaseId             pgtypeuuid.UUID
			releaseCreatedAt      pgtype.Timestamptz
			releaseUpdatedAt      pgtype.Timestamptz
			releasePublic         pgtype.Bool
			releaseName           pgtype.Text
			releaseDescription    pgtype.Text
			releaseVersion        pgtype.Text
			releaseCodeVersion    pgtype.Text
			releaseContentVersion pgtype.Text
			releaseArchive        pgtype.Bool
			appId                 pgtypeuuid.UUID
			appCreatedAt          pgtype.Timestamptz
			appUpdatedAt          pgtype.Timestamptz
			appPublic             pgtype.Bool
			appName               pgtype.Text
			appDescription        pgtype.Text
			appExternal           pgtype.Bool
			worldId               pgtypeuuid.UUID
			worldCreatedAt        pgtype.Timestamptz
			worldUpdatedAt        pgtype.Timestamptz
			worldPublic           pgtype.Bool
			worldName             pgtype.Text
			worldDescription      pgtype.Text
			worldMap              pgtype.Text
			worldPackageId        pgtypeuuid.UUID
			gameModeId            pgtypeuuid.UUID
			gameModeName          pgtype.Text
			gameModePath          pgtype.Text
		)

		err = rows.Scan(&id,
			&createdAt,
			&updatedAt,
			&public,
			&serverType,
			&host,
			&port,
			&numPlayers,
			&maxPlayers,
			&status,
			&statusMessage,
			&regionId,
			&regionName,
			&releaseId,
			&releaseCreatedAt,
			&releaseUpdatedAt,
			&releasePublic,
			&releaseName,
			&releaseDescription,
			&releaseVersion,
			&releaseCodeVersion,
			&releaseContentVersion,
			&releaseArchive,
			&appId,
			&appCreatedAt,
			&appUpdatedAt,
			&appPublic,
			&appName,
			&appDescription,
			&appExternal,
			&worldId,
			&worldCreatedAt,
			&worldUpdatedAt,
			&worldPublic,
			&worldName,
			&worldDescription,
			&worldMap,
			&worldPackageId,
			&gameModeId,
			&gameModeName,
			&gameModePath)
		if err != nil {
			err = errors.Wrap(err, "failed to scan game server")
			return
		}

		if id.Status != pgtype.Present {
			continue
		}

		var region *Region
		if regionId.Status == pgtype.Present {
			region = &Region{}
			region.Id = regionId.UUID
			if regionName.Status == pgtype.Present {
				region.Name = regionName.String
			}
		}

		var release *ReleaseV2
		if releaseId.Status == pgtype.Present {
			release = &ReleaseV2{}
			release.Id = releaseId.UUID
			if releaseCreatedAt.Status == pgtype.Present {
				release.CreatedAt = releaseCreatedAt.Time
			}
			if releaseUpdatedAt.Status == pgtype.Present {
				release.UpdatedAt = &releaseUpdatedAt.Time
			}
			if releasePublic.Status == pgtype.Present {
				release.Public = releasePublic.Bool
			}
			if releaseName.Status == pgtype.Present {
				release.Name = &releaseName.String
			}
			if releaseDescription.Status == pgtype.Present {
				release.Description = &releaseDescription.String
			}
			if releaseVersion.Status == pgtype.Present {
				release.Version = releaseVersion.String
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

		var app *AppV2
		if appId.Status == pgtype.Present {
			app = &AppV2{}
			app.Id = appId.UUID
			if appCreatedAt.Status == pgtype.Present {
				app.CreatedAt = appCreatedAt.Time
			}
			if appUpdatedAt.Status == pgtype.Present {
				app.UpdatedAt = &appUpdatedAt.Time
			}
			if appPublic.Status == pgtype.Present {
				app.Public = appPublic.Bool
			}
			if appName.Status == pgtype.Present {
				app.Name = appName.String
			}
			if appDescription.Status == pgtype.Present {
				app.Description = &appDescription.String
			}
			if appExternal.Status == pgtype.Present {
				app.External = appExternal.Bool
			}
		}

		var world *World
		world = &World{}
		if worldId.Status == pgtype.Present {
			world.Id = worldId.UUID
			if worldCreatedAt.Status == pgtype.Present {
				world.CreatedAt = worldCreatedAt.Time
			}
			if worldUpdatedAt.Status == pgtype.Present {
				world.UpdatedAt = &worldUpdatedAt.Time
			}
			if worldName.Status == pgtype.Present {
				world.Name = worldName.String
			}
			if worldDescription.Status == pgtype.Present {
				world.Description = &worldDescription.String
			}
			if worldMap.Status == pgtype.Present {
				world.Map = worldMap.String
			}
			if worldPackageId.Status == pgtype.Present {
				world.PackageId = worldPackageId.UUID
			}
		}

		var gameMode *GameMode
		gameMode = &GameMode{}
		if gameModeId.Status == pgtype.Present {
			gameMode.Id = gameModeId.UUID
			if gameModeName.Status == pgtype.Present {
				gameMode.Name = gameModeName.String
			}
			if gameModePath.Status == pgtype.Present {
				gameMode.Path = gameModePath.String
			}
		}

		e.Region = region
		e.Release = release
		e.Release.App = app
		e.World = world
		e.GameMode = gameMode
	}

	if e == nil {
		return nil, ErrNoRows
	}

	return
}

type CreateGameServerV2Args struct {
	RegionId   uuid.UUID  `json:"regionId"`             // required for official servers
	ReleaseId  uuid.UUID  `json:"releaseId"`            // required
	WorldId    uuid.UUID  `json:"worldId"`              // required
	GameModeId *uuid.UUID `json:"gameModeId,omitempty"` // optional
	Type       string     `json:"type"`                 // "official" or "community"
	Public     bool       `json:"public"`               // if the game server should be public
	MaxPlayers int        `json:"maxPlayers"`           // will have several slots for admins
}

// CreateGameServerV2 creates a new game server. Note that the port is not set here, it is set by the server operator.
func CreateGameServerV2(ctx context.Context, requester *User, args CreateGameServerV2Args) (e *GameServerV2, err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	var q = `with e as (
    insert into entities (id, entity_type, public)
        values (gen_random_uuid(), 'game-server-v2', true)
        returning id, created_at, updated_at, public)
insert
into game_server_v2 (id, region_id, release_id, world_id, game_mode_id, type, max_players, status)
select e.id,
       $1,
       $2,
       $3,
       $4,
       $5,
       $6,
       $7
from e
returning id`

	row := db.QueryRow(ctx, q,
		args.RegionId,
		args.ReleaseId,
		args.WorldId,
		args.GameModeId,
		args.Type,
		args.MaxPlayers,
		GameServerV2StatusCreated)

	var id pgtypeuuid.UUID

	err = row.Scan(&id)
	if err != nil {
		return
	}

	e, err = GetGameServerV2(ctx, requester, id.UUID)

	return
}

// UpdateGameServerV2Port updates the port of a game server.
//
//goland:noinspection GoUnusedExportedFunction
func UpdateGameServerV2Port(ctx context.Context, requester *User, port int32) (err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	if !requester.IsAdmin && !requester.IsInternal {
		err = ErrNoPermission
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	var q = `update game_server_v2 set port = $1 where id = $2`

	_, err = db.Exec(ctx, q, port, requester.Id)

	return
}

type UpdateGameServerV2StatusArgs struct {
	Id              uuid.UUID   `json:"id"`
	Status          string      `json:"status"`
	OnlinePlayerIds []uuid.UUID `json:"onlinePlayerIds"`
}

// UpdateGameServerV2Status updates the status of a game server.
//
//goland:noinspection GoUnusedExportedFunction
func UpdateGameServerV2Status(ctx context.Context, requester *User, args UpdateGameServerV2StatusArgs) (err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	if !requester.IsAdmin && !requester.IsInternal {
		err = ErrNoPermission
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	var found = false
	for _, s := range ValidGameServerV2Statuses {
		if s == args.Status {
			found = true
			break
		}
	}
	if !found {
		err = ErrInvalidServerStatus
		return
	}

	var q = `update game_server_v2 set status = $1 where id = $2`
	_, err1 := db.Exec(ctx, q, args.Status, args.Id)
	if err1 != nil {
		err = errors.Wrap(err1, "failed to update game server status")
	}

	q = `update entities set updated_at = now() where id = $1`
	_, err1 = db.Exec(ctx, q, args.Id)
	if err1 != nil {
		err = errors.Wrap(err1, "failed to set game server updated time")
	}

	// update online game server player statuses
	if len(args.OnlinePlayerIds) > 0 {
		// if there are too many players, update them in batches
		if len(args.OnlinePlayerIds) > 100 {
			// split into batches of 100
			var batches = make([][]uuid.UUID, 0)
			for i := 0; i < len(args.OnlinePlayerIds); i += 100 {
				end := i + 100
				if end > len(args.OnlinePlayerIds) {
					end = len(args.OnlinePlayerIds)
				}
				batches = append(batches, args.OnlinePlayerIds[i:end])
			}

			// update each batch
			for _, batch := range batches {
				q = `update game_server_player_v2 set status = 'online' where server_id = $1 and user_id = any($2)`
				_, err1 = db.Exec(ctx, q, args.Id, batch)
				if err1 != nil {
					err = errors.Wrap(err1, "failed to update online game server player statuses")
				}
			}
		} else {
			q = `update game_server_player_v2 set status = 'online' where server_id = $1 and user_id = any($2)`
			_, err1 = db.Exec(ctx, q, args.Id, args.OnlinePlayerIds)
			if err1 != nil {
				err = errors.Wrap(err1, "failed to update online game server player statuses")
			}
		}
	}

	return
}

// MatchGameServerV2Args contains the arguments for matching a game server.
type MatchGameServerV2Args struct {
	RegionId   uuid.UUID  `json:"regionId"`             // required for official servers
	ReleaseId  uuid.UUID  `json:"releaseId"`            // required
	WorldId    uuid.UUID  `json:"worldId"`              // required
	GameModeId *uuid.UUID `json:"gameModeId,omitempty"` // optional
	Type       string     `json:"type"`                 // "official" or "community"
}

// MatchGameServerV2 returns a game server that matches the given criteria or creates a new one.
//
//goland:noinspection GoUnusedExportedFunction
func MatchGameServerV2(ctx context.Context, requester *User, args MatchGameServerV2Args) (e *GameServerV2, created bool, err error) {
	findArgs := FindGameServerV2Args{
		RegionId:   args.RegionId,
		ReleaseId:  args.ReleaseId,
		WorldId:    args.WorldId,
		GameModeId: args.GameModeId,
		Type:       args.Type,
	}

	// Try to find an online game server that matches the given criteria.
	e, err = FindGameServerV2(ctx, requester, findArgs)

	// If no game server is found, create a new one.
	if err == ErrNoRows {
		createArgs := CreateGameServerV2Args{
			RegionId:   args.RegionId,
			ReleaseId:  args.ReleaseId,
			WorldId:    args.WorldId,
			GameModeId: args.GameModeId,
			Type:       args.Type,
			Public:     true,
			MaxPlayers: 100, // todo: consider to take it from the world
		}
		e, err = CreateGameServerV2(ctx, requester, createArgs)
		created = true
	}

	if err != nil {
		err = errors.Wrap(err, "failed to match game server")
		return
	}

	return
}

// AddPlayerToGameServerV2Args contains the arguments for adding a player to a game server.
type AddPlayerToGameServerV2Args struct {
	GameServerId uuid.UUID `json:"gameServerId"`
	UserId       uuid.UUID `json:"userId"`
}

// AddPlayerToGameServerV2 adds a player to a game server.
//
//goland:noinspection GoUnusedExportedFunction
func AddPlayerToGameServerV2(ctx context.Context, requester *User, args AddPlayerToGameServerV2Args) (err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	if !requester.IsAdmin && !requester.IsInternal {
		err = ErrNoPermission
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	// Check if the player is already connected to the game server.
	var q = `select count(*) from game_server_player_v2 where server_id = $1 and user_id = $2 and status = $3`

	var connectedPlayers int32

	err = db.QueryRow(ctx, q, args.GameServerId, args.UserId, GameServerV2PlayerStatusConnected).Scan(&connectedPlayers)
	if err != nil {
		err = errors.Wrap(err, "failed to check if player is already connected to game server")
		return
	}

	if connectedPlayers > 0 {
		err = ErrPlayerAlreadyConnected
		return
	}

	// Check if server has a free slot.
	q = `select count(*) from game_server_player_v2 where server_id = $1 and status = $2`

	err = db.QueryRow(ctx, q, args.GameServerId, GameServerV2PlayerStatusConnected).Scan(&connectedPlayers)
	if err != nil {
		err = errors.Wrap(err, "failed to check if game server has space for player")
		return
	}

	var server *GameServerV2
	server, err = GetGameServerV2(ctx, requester, args.GameServerId)
	if err != nil {
		err = errors.Wrap(err, "failed to find game server")
		return
	}

	// If the requester is an admin, allow to connect to reserved slots even if the server is full.
	if requester.IsAdmin {
		if connectedPlayers >= server.MaxPlayers {
			err = ErrNoFreeSlots
			return
		}
	} else {
		if connectedPlayers >= (server.MaxPlayers - GameServerReservedSlots) {
			err = ErrNoFreeSlots
			return
		}
	}

	q = `insert into game_server_player_v2 (server_id, user_id, status) values ($1, $2, $3)`

	_, err = db.Exec(ctx, q, args.GameServerId, args.UserId, GameServerV2PlayerStatusConnected)
	if err != nil {
		err = errors.Wrap(err, "failed to add player to game server")
		return
	}

	return
}

// UpdateGameServerV2PlayerStatusArgs contains the arguments for updating the status of a player on a game server.
type UpdateGameServerV2PlayerStatusArgs struct {
	GameServerId uuid.UUID `json:"gameServerId"`
	UserId       uuid.UUID `json:"userId"`
	Status       string    `json:"status"`
}

// UpdateGameServerV2PlayerStatus updates the status of a player on a game server.
//
//goland:noinspection GoUnusedExportedFunction
func UpdateGameServerV2PlayerStatus(ctx context.Context, requester *User, args UpdateGameServerV2PlayerStatusArgs) (err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	if !requester.IsAdmin && !requester.IsInternal {
		err = ErrNoPermission
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	var found = false
	for _, s := range ValidGameServerV2PlayerStatuses {
		if s == args.Status {
			found = true
			break
		}
	}
	if !found {
		err = ErrInvalidServerStatus
		return
	}

	// check if the player is connected to the game server
	var q = `select count(*) from game_server_player_v2 where server_id = $1 and user_id = $2 and status = $3`

	var connectedPlayers int32

	err = db.QueryRow(ctx, q, args.GameServerId, args.UserId, GameServerV2PlayerStatusConnected).Scan(&connectedPlayers)
	if err != nil {
		err = errors.Wrap(err, "failed to check if player is connected to game server")
		return
	}

	if connectedPlayers == 0 {
		err = ErrPlayerNotConnected
		return
	}

	q = `update game_server_player_v2 set status = $1 where server_id = $2 and user_id = $3`
	_, err1 := db.Exec(ctx, q, args.Status, args.GameServerId, args.UserId)
	if err1 != nil {
		err = errors.Wrap(err1, "failed to update game server player status")
	}

	return
}

// RemovePlayerFromGameServerV2Args contains the arguments for removing a player from a game server.
type RemovePlayerFromGameServerV2Args struct {
	GameServerId uuid.UUID `json:"gameServerId"`
	UserId       uuid.UUID `json:"userId"`
}

// RemovePlayerFromGameServerV2 removes a player from a game server.
//
//goland:noinspection GoUnusedExportedFunction
func RemovePlayerFromGameServerV2(ctx context.Context, requester *User, args RemovePlayerFromGameServerV2Args) (err error) {
	if requester == nil {
		err = ErrNoRequester
		return
	}

	if !requester.IsAdmin && !requester.IsInternal {
		err = ErrNoPermission
		return
	}

	db, ok := ctx.Value(sc.Database).(*pgxpool.Pool)
	if !ok || db == nil {
		err = ErrNoDatabase
		return
	}

	// check if the player is connected to the game server
	var q = `select count(*) from game_server_player_v2 where server_id = $1 and user_id = $2 and status = $3`

	var connectedPlayers int32

	err = db.QueryRow(ctx, q, args.GameServerId, args.UserId, GameServerV2PlayerStatusConnected).Scan(&connectedPlayers)
	if err != nil {
		err = errors.Wrap(err, "failed to check if player is connected to game server")
		return
	}

	if connectedPlayers == 0 {
		err = ErrPlayerNotConnected
		return
	}

	q = `update game_server_player_v2 set status = $1 where server_id = $2 and user_id = $3`
	_, err1 := db.Exec(ctx, q, GameServerV2PlayerStatusDisconnected, args.GameServerId, args.UserId)
	if err1 != nil {
		err = errors.Wrap(err1, "failed to update game server player status")
	}

	return
}
