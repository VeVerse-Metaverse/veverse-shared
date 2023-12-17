package model

import "errors"

var (
	ErrNoRequester            = errors.New("no requester provided")
	ErrNoDatabase             = errors.New("no database provided")
	ErrNoRows                 = errors.New("no rows returned")
	ErrNoPermission           = errors.New("no permission to perform this action")
	ErrInvalidServerStatus    = errors.New("invalid server status")
	ErrPlayerNotConnected     = errors.New("player not connected to server")
	ErrPlayerAlreadyConnected = errors.New("player already connected to server")
	ErrNoFreeSlots            = errors.New("no free slots on server")
)
