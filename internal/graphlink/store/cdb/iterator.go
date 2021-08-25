package cdb

import (
	"database/sql"
	"fmt"

	"github.com/mycok/uSearch/internal/graphlink/graph"
)

// LinkIterator servers as a graph.LinkIterator interface concrete
// implementation for the in-memory graph store.
type LinkIterator struct {
	rows    *sql.Rows
	lastErr error
	link    *graph.Link
}

// Next implements graph.LinkIterator.Next()
func (i *LinkIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	l := new(graph.Link)

	i.lastErr = i.rows.Scan(&l.ID, &l.URL, &l.RetrievedAt)
	if i.lastErr != nil {
		return false
	}

	l.RetrievedAt = l.RetrievedAt.UTC()
	i.link = l

	return true
}

// Error implements graph.LinkIterator.Error()
func (i *LinkIterator) Error() error {
	return i.lastErr
}

// Close implements graph.LinkIterator.Close()
func (i *LinkIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return fmt.Errorf("link iterator: %w", err)
	}

	return nil
}

// Link implements graph.LinkIterator.Link()
func (i *LinkIterator) Link() *graph.Link {
	return i.link
}
