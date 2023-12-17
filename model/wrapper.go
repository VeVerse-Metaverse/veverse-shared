package model

type Wrapper[T any] struct {
	Payload T      `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}
