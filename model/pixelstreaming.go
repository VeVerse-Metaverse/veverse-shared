package model

import (
	"github.com/gofrs/uuid"
)

type PixelStreamingInstance struct {
	Identifier
	ReleaseId    *uuid.UUID `json:"releaseId,omitempty"`
	RegionId     *uuid.UUID `json:"regionId,omitempty"`
	Host         string     `json:"host,omitempty"`
	Port         uint16     `json:"port,omitempty"`
	Status       string     `json:"status,omitempty"`
	InstanceType string     `json:"instanceType,omitempty"`
}

type PixelStreamingSession struct {
	Identifier

	InstanceId *uuid.UUID `json:"instanceId"`
	AppId      *uuid.UUID `json:"appId"`
	WorldId    *uuid.UUID `json:"worldId"`
	Status     string     `json:"status"`

	Timestamps
}

type PixelStreamingSessionData struct {
	Id           *uuid.UUID `json:"id,omitempty"`
	InstanceType string     `json:"instanceType,omitempty"`
	AppId        *uuid.UUID `json:"appId,omitempty"`
	WorldId      *uuid.UUID `json:"worldId,omitempty"`
	Status       string     `json:"status,omitempty"`
}
