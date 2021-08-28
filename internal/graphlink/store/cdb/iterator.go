package cdb

import (
	"database/sql"
	"fmt"

	"github.com/mycok/uSearch/internal/graphlink/graph"
)

// linkIterator type serves as a graph.LinkIterator interface concrete
// implementation for the cockroachDB graph store.
type linkIterator struct {
	rows    *sql.Rows
	lastErr error
	link    *graph.Link
}

// Next implements graph.LinkIterator.Next() which returns true to indicate
// the presence of a valid link object and false otherwise.
func (i *linkIterator) Next() bool {
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

// Error implements graph.LinkIterator.Error() and returns the last error
// returned by calling graph.LinkIterator.Next().
func (i *linkIterator) Error() error {
	return i.lastErr
}

// Close implements graph.LinkIterator.Close() and releases any allocated resources
// still being used by the graph.LinkIterator.
func (i *linkIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return fmt.Errorf("link iterator: %w", err)
	}

	return nil
}

// Link implements graph.LinkIterator.Link()
func (i *linkIterator) Link() *graph.Link {
	return i.link
}

// edgeIterator type servers as a graph.EdgeIterator interface concrete
// implementation for the cockroachDB graph store.
type edgeIterator struct {
	rows    *sql.Rows
	lastErr error
	edge    *graph.Edge
}

// Next implements graph.EdgeIterator.Next() which returns true to indicate
// the presence of a valid edge object and false otherwise.
func (i *edgeIterator) Next() bool {
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

// Error implements graph.EdgeIterator.Error() and returns the last error
// returned by calling graph.EdgeIterator.Next().
func (i *edgeIterator) Error() error {
	return i.lastErr
}

// Close implements graph.EdgeIterator.Close() and releases any allocated resources
// still being used by the graph.EdgeIterator.
func (i *edgeIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return fmt.Errorf("edge iterator: %w", err)
	}

	return nil
}

// Edge implements graph.EdgeIterator.Edge() which returns an edge object
// whenever a call to graph.EdgeIterator.Next() returns true.
func (i *edgeIterator) Edge() *graph.Edge {
	return i.edge
}
