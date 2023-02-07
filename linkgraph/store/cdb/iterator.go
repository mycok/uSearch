package cdb

import (
	"database/sql"
	"fmt"

	"github.com/mycok/uSearch/linkgraph/graph"
)

// Static and compile-time check to ensure linkIterator implements
// graph.Iterator interface.
var _ graph.Iterator = (*linkIterator)(nil)

// linkIterator is a graph.LinkIterator implementation for the in-memory graph.
// It wraps the [database/sql] Rows type that serves as an iterator for the
// returned query data.
type linkIterator struct {
	rows    *sql.Rows
	lastErr error
	link    *graph.Link
}

// Next loads the next item, returns false when no more documents
// are available or when an error occurs.
func (i *linkIterator) Next() bool {
	// Check if an error occurred during the most recent [rows.Scan]
	// operation or if there are no more rows data to return.
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	l := new(graph.Link)
	if i.lastErr = i.rows.Scan(&l.ID, &l.URL, &l.RetrievedAt); i.lastErr != nil {
		return false
	}

	// Re-assign this field to a .UTC time value to cater for cases
	// where the retrieved time for the field is reverted back to a non
	// UTC value during the Scan / parsing process.
	l.RetrievedAt = l.RetrievedAt.UTC()
	i.link = l

	return true
}

// Error returns the last error encountered by the iterator.
func (i *linkIterator) Error() error {
	return i.lastErr
}

// Close releases any resources allocated to the iterator.
func (i *linkIterator) Close() error {
	if err := i.rows.Close(); err != nil {
		return fmt.Errorf("link iterator: %w", err)
	}

	return nil
}

// Link returns the currently fetched link object.
func (i *linkIterator) Link() *graph.Link {
	return i.link
}

// edgeIterator is a graph.EdgeIterator implementation for the in-memory graph.
type edgeIterator struct {
	rows    *sql.Rows
	lastErr error
	edge    *graph.Edge
}

// Next advances the iterator. When no items are available or when an
// error occurs, calls to Next() return false.
func (i *edgeIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	e := new(graph.Edge)
	if i.lastErr = i.rows.Scan(
		&e.ID, &e.Src, &e.Dest, &e.UpdatedAt,
	); i.lastErr != nil {

		return false
	}

	e.UpdatedAt = e.UpdatedAt.UTC()
	i.edge = e

	return true
}

// Error returns the last error recorded by the iterator.
func (i *edgeIterator) Error() error {
	return i.lastErr
}

// Close releases any resources linked to the iterator.
func (i *edgeIterator) Close() error {
	if err := i.rows.Close(); err != nil {
		return fmt.Errorf("edge iterator: %w", err)
	}

	return nil
}

// Edge returns the currently fetched edge object.
func (i *edgeIterator) Edge() *graph.Edge {
	return i.edge
}
