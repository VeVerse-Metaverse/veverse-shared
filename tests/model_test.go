package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"dev.hackerman.me/artheon/veverse-shared/database"
	"dev.hackerman.me/artheon/veverse-shared/model"
	"github.com/gofrs/uuid"
)

func TestIndexLaunchers(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	var (
		offset int64 = 0
		limit  int64 = 10
		search       = ""
	)

	tests := []struct {
		name    string
		user    model.User
		request model.IndexLauncherV2Request
	}{
		{
			"launchers (user)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			model.IndexLauncherV2Request{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
			},
		},
		{
			"launchers (admin)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			model.IndexLauncherV2Request{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
			},
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps, err := model.IndexLauncherV2(ctx, &tt.user, tt.request)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(apps, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal apps: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}

func TestGetLauncher(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	tests := []struct {
		name          string
		user          model.User
		launcherId    uuid.UUID
		platform      string
		expectedFiles int
	}{
		{
			"launcher (user) win64",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Win64",
			2,
		},
		{
			"launcher (user) linux",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Linux",
			1,
		},
		{
			"launcher (admin) win64",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Win64",
			2,
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			launcherV2, err := model.GetLauncherV2(ctx, &tt.user, tt.launcherId, tt.platform)
			if err != nil {
				t.Errorf("GetLauncherV2() error = %+v", err)
				return
			}

			fmt.Println(launcherV2)

			j, err := json.MarshalIndent(launcherV2, "", "\t")
			if err != nil {
				t.Errorf("failed to marshal launcher: %v", err)
				return
			}

			fmt.Println(string(j))

			if launcherV2.Owner == nil {
				t.Errorf("GetLauncherV2() owner is nil")
				return
			}

			if launcherV2.Owner.Id != tt.user.Id {
				t.Errorf("GetLauncherV2() owner id = %v, want %v", launcherV2.Owner.Id, tt.user.Id)
				return
			}

			if len(launcherV2.Files.Entities) < tt.expectedFiles {
				t.Errorf("expected %d files, got %d", tt.expectedFiles, len(launcherV2.Files.Entities))
				return
			}

			if launcherV2.Files.Total != 0 {
				t.Errorf("expected total to be 0")
				return
			}

			if launcherV2.Files.Offset != 0 {
				t.Errorf("expected offset to be 0")
				return
			}

			if launcherV2.Files.Limit != 0 {
				t.Errorf("expected limit to be 0")
				return
			}

			if len(launcherV2.Releases.Entities) == 0 {
				t.Errorf("expected latest release to be not nil")
				return
			}
		})
	}
}

func TestIndexLauncherReleases(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	tests := []struct {
		name             string
		user             model.User
		launcherId       uuid.UUID
		platform         string
		expectedReleases int
		offset           int64
		limit            int64
	}{
		{
			"releases (user) win64",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Win64",
			1,
			0,
			10,
		},
		{
			"releases (user) linux",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Linux",
			1,
			0,
			10,
		},
		{
			"releases (admin) win64",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Win64",
			1,
			0,
			10,
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releases, err := model.IndexLauncherV2Releases(ctx, &tt.user, tt.launcherId, tt.platform, tt.offset, tt.limit)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			database.LogPgxPoolStatistics(ctx, "IndexLauncherV2Releases")

			j, err := json.MarshalIndent(releases, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal releases: %v", err)
				return
			}

			fmt.Println(string(j))

			if len(releases) != tt.expectedReleases {
				t.Errorf("expected %d releases, got %d", tt.expectedReleases, len(releases))
				return
			}

			for _, release := range releases {
				if release.Owner == nil {
					t.Errorf("owner is nil")
					return
				}

				if release.Owner.Id != tt.user.Id {
					t.Errorf("owner id = %v, want %v", release.Owner.Id, tt.user.Id)
					return
				}

				//if len(release.Files.Entities) == 0 {
				//	t.Errorf("expected at least one file, got %d", len(release.Files.Entities))
				//	return
				//}
			}
		})
	}
}

func TestIndexLauncherApps(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	tests := []struct {
		name         string
		user         model.User
		launcherId   uuid.UUID
		platform     string
		expectedApps int
		offset       int64
		limit        int64
	}{
		{
			"apps (user) win64",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Win64",
			1,
			0,
			10,
		},
		{
			"apps (user) linux",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Linux",
			1,
			0,
			10,
		},
		{
			"apps (admin) win64",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			uuid.FromStringOrNil("684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5"),
			"Win64",
			1,
			0,
			10,
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apps, err := model.IndexLauncherV2Apps(ctx, &tt.user, tt.launcherId, tt.platform, tt.offset, tt.limit)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(apps, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal apps: %v", err)
				return
			}

			fmt.Println(string(j))

			if len(apps) != tt.expectedApps {
				t.Errorf("expected %d apps, got %d", tt.expectedApps, len(apps))
				return
			}

			for _, app := range apps {
				if app.Id == uuid.Nil {
					t.Errorf("app id is nil")
					return
				}

				if app.Owner == nil {
					t.Errorf("owner is nil")
					return
				}

				if app.Owner.Id != tt.user.Id {
					t.Errorf("owner id = %v, want %v", app.Owner.Id, tt.user.Id)
					return
				}

				if len(app.Files.Entities) == 0 {
					t.Errorf("expected at least one file, got %d", len(app.Files.Entities))
					return
				}

				for _, file := range app.Files.Entities {
					if file.Id == uuid.Nil {
						t.Errorf("file id is nil")
						return
					}
				}

				if app.Releases == nil || len(app.Releases.Entities) == 0 {
					t.Logf("expected latest release, got nil")
				} else {
					for _, release := range app.Releases.Entities {
						if release.Id == uuid.Nil {
							t.Errorf("release id is nil")
							return
						}

						if release.Files.Entities == nil || len(release.Files.Entities) == 0 {
							t.Errorf("expected at least one file, got %d", len(release.Files.Entities))
							return
						}
					}
				}

				if app.SDK == nil {
					t.Errorf("expected sdk, got nil")
					return
				}

				if app.SDK.Releases.Entities == nil || len(app.SDK.Releases.Entities) == 0 {
					t.Errorf("expected sdk releases, got nil")
					return
				}

				for _, release := range app.SDK.Releases.Entities {
					if release.Id == uuid.Nil {
						t.Errorf("sdk release id is nil")
						return
					}

					if release.Files.Entities == nil || len(release.Files.Entities) == 0 {
						t.Errorf("expected sdk release files, got nil")
						return
					}
				}

				if app.Links == nil {
					t.Errorf("expected links, got nil")
					return
				}

				if len(app.Links.Entities) == 0 {
					t.Errorf("expected at least one link, got %d", len(app.Links.Entities))
					return
				}
			}
		})
	}
}

func TestIndexReleases(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	var (
		offset int64 = 0
		limit  int64 = 10
		search       = ""
	)

	tests := []struct {
		name    string
		user    model.User
		request model.IndexReleaseV2Request
	}{
		{
			"releases (user)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			model.IndexReleaseV2Request{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
			},
		},
		{
			"releases (admin)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			model.IndexReleaseV2Request{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
			},
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := model.IndexReleaseV2(ctx, &tt.user, tt.request)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(entities, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal entities: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}

func TestGetRelease(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	tests := []struct {
		name      string
		user      model.User
		releaseId uuid.UUID
		private   bool
	}{
		{
			"release (user) private",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("87f30b84-c771-4055-b200-4b44a41fedb1"),
			true,
		},
		{
			"release (user) public",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			uuid.FromStringOrNil("6243C871-4890-418F-8E60-00FAFDD48907"),
			false,
		},
		{
			"release (admin) private",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			uuid.FromStringOrNil("87f30b84-c771-4055-b200-4b44a41fedb1"),
			true,
		},
		{
			"release (admin) public",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			uuid.FromStringOrNil("6243C871-4890-418F-8E60-00FAFDD48907"),
			false,
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseV2, err := model.GetReleaseV2(ctx, &tt.user, tt.releaseId)
			if err != nil {
				t.Errorf("GetReleaseV2() error = %+v", err)
				return
			}

			fmt.Println(releaseV2)

			j, err := json.MarshalIndent(releaseV2, "", "\t")
			if err != nil {
				t.Errorf("failed to marshal release: %v", err)
				return
			}

			fmt.Println(string(j))

			if tt.private && tt.user.IsAdmin {
				if releaseV2.Owner == nil {
					t.Errorf("GetReleaseV2() owner is nil")
					return
				}

				if releaseV2.Owner.Id != tt.user.Id {
					t.Errorf("GetReleaseV2() owner id = %v, want %v", releaseV2.Owner.Id, tt.user.Id)
					return
				}

				return
			}

			if tt.private && !tt.user.IsAdmin && releaseV2 != nil {
				t.Errorf("GetReleaseV2() release = %+v, want nil as had no access to private entity", releaseV2)
			}
		})
	}
}

func TestIndexAnalytics(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	var (
		offset            int64 = 0
		limit             int64 = 10
		appId                   = "52f16ed3-b299-4091-85aa-389d4ff4285f"
		contextEntityId         = "9eb4cd48-404b-dfe7-7193-78834bea969a"
		contextEntityType       = "test"
		userIdString            = "00000000-0000-4000-a000-00000000000c"
		platform                = "Win64"
		deployment              = "Client"
		configuration           = "Test"
		event                   = "test"
	)

	tests := []struct {
		name    string
		user    model.User
		request model.IndexAnalyticEventRequest
	}{
		{
			"events (user)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			model.IndexAnalyticEventRequest{
				Offset: &offset,
				Limit:  &limit,
			},
		},
		{
			"events (admin)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			model.IndexAnalyticEventRequest{
				Offset: &offset,
				Limit:  &limit,
				Event:  &event,
			},
		},
		{
			"events (admin)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			model.IndexAnalyticEventRequest{
				Offset:            &offset,
				Limit:             &limit,
				AppId:             &appId,
				ContextEntityId:   &contextEntityId,
				ContextEntityType: &contextEntityType,
				UserId:            &userIdString,
				Platform:          &platform,
				Deployment:        &deployment,
				Configuration:     &configuration,
			},
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	ctx, err = GetClickhouseContext(ctx)
	if err != nil {
		t.Fatalf("failed to get clickhouse context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := model.IndexAnalyticEvent(ctx, &tt.user, tt.request)
			if err != nil {
				if err == model.ErrNoPermission && !tt.user.IsAdmin {
					return
				}
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(entities, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal entities: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}

func TestIndexUsers(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	var (
		offset int64 = 0
		limit  int64 = 10
		search       = ""
	)

	tests := []struct {
		name    string
		user    model.User
		request model.IndexUserRequest
	}{
		{
			"users (user)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			model.IndexUserRequest{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
			},
		},
		{
			"users (admin)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: true,
			},
			model.IndexUserRequest{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
			},
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := model.IndexUser(ctx, &tt.user, tt.request)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(entities, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal entities: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}

func TestIndexWorlds(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	var (
		offset int64 = 0
		limit  int64 = 35
		search       = "d"
	)

	tests := []struct {
		name    string
		user    model.User
		request model.IndexWorldRequest
	}{
		{
			"worlds (user)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			model.IndexWorldRequest{
				Offset: &offset,
				Limit:  &limit,
				Search: &search,
				Options: &model.WorldRequestOptions{
					Pak: true,
					PakOptions: &model.WorldRequestPakOptions{
						Platform:   "Win64",
						Deployment: "Client",
					},
					Preview: true,
					Likes:   true,
					Owner:   true,
				},
				Sort: []model.IndexRequestSort{
					{
						Column:    "pakFile",
						Direction: "asc",
					},
					{
						Column:    "previewFile",
						Direction: "asc",
					},
					{
						Column:    "likes",
						Direction: "desc",
					},
				},
			},
		},
		//{
		//	"worlds (admin)",
		//	model.User{
		//		Entity: model.Entity{
		//			Identifier: model.Identifier{Id: userId},
		//		},
		//		IsAdmin: true,
		//	},
		//	model.IndexWorldRequest{
		//		Offset: &offset,
		//		Limit:  &limit,
		//		Search: &search,
		//	},
		//},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := model.IndexWorld(ctx, &tt.user, tt.request)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(entities, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal entities: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}

func TestGetWorld(t *testing.T) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	var (
		id = uuid.FromStringOrNil("6d790807-2ccf-4c0e-bba5-47da33540f69")
	)

	tests := []struct {
		name    string
		user    model.User
		request model.GetWorldRequest
	}{
		{
			"get world (user)",
			model.User{
				Entity: model.Entity{
					Identifier: model.Identifier{Id: userId},
				},
				IsAdmin: false,
			},
			model.GetWorldRequest{
				Id: id,
				Options: &model.WorldRequestOptions{
					Pak: true,
					PakOptions: &model.WorldRequestPakOptions{
						Platform:   "Win64",
						Deployment: "Client",
					},
					Preview: true,
					Likes:   true,
					Owner:   true,
				},
			},
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities, err := model.GetWorld(ctx, &tt.user, tt.request)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			j, err := json.MarshalIndent(entities, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal entities: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}

func TestGetLatestReleaseV2Public(t *testing.T) {
	var (
		appId = uuid.FromStringOrNil("0207e030-cc69-4fe0-9408-3b0ce5fc124d")
	)

	tests := []struct {
		name    string
		request model.GetLatestReleaseRequest
	}{
		{
			"get world (user)",
			model.GetLatestReleaseRequest{
				AppId: appId,
				Options: &model.LatestReleaseRequestOptions{
					Files: true,
					FileOptions: &model.LatestReleaseRequestFileOptions{
						Platform: "Win64",
						Target:   "Client",
					},
					Owner: true,
				},
			},
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, err := model.GetLatestReleaseV2Public(ctx, tt.request)
			if err != nil {
				t.Errorf("error = %+v", err)
				return
			}

			if release == nil {
				t.Errorf("release is nil")
				return
			}

			if release.Files == nil {
				t.Errorf("release files is nil")
				return
			}

			if release.Files.Entities == nil {
				t.Errorf("release files entities is nil")
				return
			}

			if len(release.Files.Entities) == 0 {
				t.Errorf("release files entities is empty")
				return
			}

			j, err := json.MarshalIndent(release, "", "  ")
			if err != nil {
				t.Errorf("failed to marshal release: %v", err)
				return
			}

			fmt.Println(string(j))
		})
	}
}
