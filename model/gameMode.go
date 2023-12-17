package model

type GameMode struct {
	Entity
	Name string `json:"name"`
	Path string `json:"path"`
}
