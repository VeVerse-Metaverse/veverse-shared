package model

import (
	"time"
)

type Package struct {
	Entity

	Name          string     `json:"name,omitempty"`
	Title         string     `json:"title,omitempty"`
	Summary       string     `json:"summary,omitempty"`
	Description   string     `json:"description,omitempty"`
	Map           string     `json:"map,omitempty"`
	Release       string     `json:"release,omitempty"`
	Price         *float64   `json:"price,omitempty"`
	Version       string     `json:"version,omitempty"`
	ReleasedAt    *time.Time `json:"releasedAt,omitempty"`
	Downloads     *int32     `json:"downloads,omitempty"`
	Liked         *int32     `json:"liked,omitempty"`
	TotalLikes    *int32     `json:"totalLikes,omitempty"`
	TotalDislikes *int32     `json:"totalDislikes,omitempty"`
}

// PackageBatchRequestMetadata Batch request metadata for requesting Package entities
type PackageBatchRequestMetadata struct {
	BatchRequestMetadata
	Platform   string `json:"platform,omitempty"`   // SupportedPlatform (OS) of the pak file (Win64, Mac, Linux, IOS, Android)
	Deployment string `json:"deployment,omitempty"` // SupportedDeployment for the pak file (Server or Client)
}

type PackageRequestMetadata struct {
	IdRequestMetadata
	Platform   string `json:"platform,omitempty"`   // SupportedPlatform (OS) of the destination pak file (Win64, Mac, Linux, IOS, Android)
	Deployment string `json:"deployment,omitempty"` // SupportedDeployment for the destination pak file (Server or Client)
}

type PackageCreateMetadata struct {
	Name        string  `json:"name,omitempty"`        // Name that used as package identifier
	Title       *string `json:"title,omitempty"`       // Title visible to users
	Public      *bool   `json:"public,omitempty"`      // Public or private
	Summary     *string `json:"summary,omitempty"`     // Short Summary
	Description *string `json:"description,omitempty"` // Full Description
	Release     string  `json:"releaseName,omitempty"` // Release
	Map         *string `json:"map,omitempty"`         // Map (list of maps included into the package)
	Version     *string `json:"version,omitempty"`     // Version of the package
}

type PackageUpdateMetadata struct {
	Name        *string `json:"name,omitempty"`        // Name that used as package identifier
	Title       *string `json:"title,omitempty"`       // Title visible to users
	Public      *bool   `json:"public,omitempty"`      // Public or private
	Summary     *string `json:"summary,omitempty"`     // Short Summary
	Description *string `json:"description,omitempty"` // Full Description
	Release     *string `json:"releaseName,omitempty"` // Release
	Map         *string `json:"map,omitempty"`         // Map (list of maps included into the package)
	Version     *string `json:"version,omitempty"`     // Version of the package
}
