package graph

import "errors"

var (
	// ErrNotFound is returned when a Link or Edge object lookup fails
	ErrNotFound = errors.New("not found")

	// ErrUnknownEdgeLinks is returned when we attempt to create
	// an Edge object with an invalid source & / or destination ID
	ErrUnknownEdgeLinks = errors.New("unknown source and / or destination")
)
