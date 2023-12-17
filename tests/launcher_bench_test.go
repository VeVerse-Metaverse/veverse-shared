package tests

import (
	"context"
	"testing"

	"dev.hackerman.me/artheon/veverse-shared/model"
	"github.com/gofrs/uuid"
)

func BenchmarkGetLauncher(b *testing.B) {
	userId := uuid.FromStringOrNil("1578BA66-3334-496E-8BB8-1A0696B42C68")

	tests := []struct {
		name     string
		user     model.User
		launcher uuid.UUID
		platform string
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
		},
	}

	ctx, err := GetDatabaseContext(context.Background())
	if err != nil {
		b.Fatalf("failed to get database context: %v", err)
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := model.GetLauncherV2(ctx, &tt.user, tt.launcher, tt.platform)
				if err != nil {
					b.Errorf("GetLauncherV2() error = %+v", err)
					return
				}
			}
		})
	}
}
