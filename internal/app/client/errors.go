package client

import "errors"

// ErrSessionExpired indicates that the server rejected the token.
var ErrSessionExpired = errors.New("session expired")

// ErrConflict indicates version conflict on update/delete.
var ErrConflict = errors.New("conflict")
