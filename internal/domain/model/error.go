package model

import "errors"

var (
	ErrSessionNotFound   = errors.New("session not found")
	ErrFileTooLarge      = errors.New("file too large")
	ErrUnsupportedFormat = errors.New("unsupported format")
	ErrParseFailed       = errors.New("parse failed")
	ErrFileReadError     = errors.New("file read error")
)
