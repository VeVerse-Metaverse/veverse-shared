package model

import (
	"github.com/gofrs/uuid"
	"time"
)

type IdRequestMetadata struct {
	Id uuid.UUID `json:"id"` // Entity ID
}

type IdRequestMetadataWithValidation struct {
	Id uuid.UUID `json:"id"` // Entity ID
}

type BatchRequestMetadata struct {
	Offset int64  `json:"offset"` // Start index
	Limit  int64  `json:"limit"`  // Number of elements to fetch
	Query  string `json:"query"`  // Search query string
}

type KeyRequestMetadata struct {
	Key string `json:"key"`
}

type HttpRequestMetadata struct {
	Id        uuid.UUID         `json:"id"`
	IPv4      string            `json:"ipv4"`
	IPv6      string            `json:"ipv6"`
	UserId    uuid.UUID         `json:"userId"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Status    uint16            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Source    string            `json:"source"`
}

type IndexRequestSort struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}
