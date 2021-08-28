package cdb

import (
	"database/sql"
	"fmt"

	"github.com/mycok/uSearch/internal/graphlink/graph"
)

// LinkIterator type serves as a graph.LinkIterator interface concrete
// implementation for the cockroachDB graph store.
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

// EdgeIterator type servers as a graph.EdgeIterator interface concrete
// implementation for the cockroachDB graph store.
type EdgeIterator struct {
	rows    *sql.Rows
	lastErr error
	edge    *graph.Edge
}

// Next implements graph.EdgeIterator.Next()
func (i *EdgeIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	e := new(graph.Edge)

	i.lastErr = i.rows.Scan(&e.ID, &e.Src, &e.Dest, &e.UpdatedAt)
	if i.lastErr != nil {
		return false
	}

	e.UpdatedAt = e.UpdatedAt.UTC()

	i.edge = e

	return true
}

// Error implements graph.EdgeIterator.Error()
func (i *EdgeIterator) Error() error {
	return i.lastErr
}

// Close implements graph.EdgeIterator.Close()
func (i *EdgeIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return fmt.Errorf("edge iterator: %w", err)
	}

	return nil
}

// Edge implements graph.EdgeIterator.Edge()
func (i *EdgeIterator) Edge() *graph.Edge {
	return i.edge
}
