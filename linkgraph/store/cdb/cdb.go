package cdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/mycok/uSearch/linkgraph/graph"
)

var (
	upsertLinkQuery = `
					INSERT INTO links (url, retrieved_at)
					VALUES ($1, $2)
					ON CONFLICT (url) 
					DO UPDATE SET retrieved_at=GREATEST(links.retrieved_at, $2)
					RETURNING id, retrieved_at
					`
	findLinkQuery = "SELECT id, url, retrieved_at FROM links WHERE id=$1"

	partitionedLinksQuery = `
							SELECT id, url, retrieved_at FROM links
							WHERE id >= $1 AND id < $2 AND retrieved_at < $3
							`

	upsertEdgeQuery = `
					INSERT INTO edges (src, dest, updated_at)
					VALUES ($1, $2, NOW())
					ON CONFLICT (src, dest)
					DO UPDATE SET updated_at=NOW()
					RETURNING id, updated_at 
					`
	partitionedEdgesQuery = `
							SELECT id, src, dest, updated_at FROM edges
							WHERE src >= $1 AND src < $2 AND updated_at < $3
							`

	removeStaleEdgesQuery = "DELETE FROM edges WHERE src=$1 AND updated_at < $2"
)

// Static and compile-time check to ensure CockroachDBGraph implements
// Graph interface.
var _ graph.Graph = (*CockroachDBGraph)(nil)

// CockroachDBGraph implements a persistent link and edge graph using
// CockroachDB instance.
type CockroachDBGraph struct {
	db *sql.DB
}

// NewCockroachDBGraph returns a CockroachDBGraph instance.
func NewCockroachDBGraph(dsn string) (*CockroachDBGraph, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &CockroachDBGraph{db}, nil
}

// Close terminates the connection to the cockroachDB instance.
func (s *CockroachDBGraph) Close() error {
	return s.db.Close()
}

// UpsertLink creates a new or updates an existing link.
func (s *CockroachDBGraph) UpsertLink(link *graph.Link) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx, upsertLinkQuery, link.URL, link.RetrievedAt,
	).Scan(&link.ID, &link.RetrievedAt)
	if err != nil {
		return fmt.Errorf("upsert link: %w", err)
	}

	return nil
}

// FindLink performs a link lookup by id.
func (s *CockroachDBGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	l := new(graph.Link)

	err := s.db.QueryRowContext(ctx, findLinkQuery, id).Scan(&l.ID, &l.URL, &l.RetrievedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("find link: %w", graph.ErrNotFound)
		}

		return nil, fmt.Errorf("find link: %w", err)
	}

	return l, nil
}

// Links returns an iterator for a set of links whose id's belong
// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
// time.
func (s *CockroachDBGraph) Links(
	fromID, toID uuid.UUID, retrievedBefore time.Time,
) (graph.LinkIterator, error) {

	rows, err := s.db.Query(
		partitionedLinksQuery, fromID, toID, retrievedBefore.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("links: %w", err)
	}

	return &linkIterator{rows: rows}, nil
}

// UpsertEdge creates a new or updates an existing edge.
func (s *CockroachDBGraph) UpsertEdge(edge *graph.Edge) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx, upsertEdgeQuery, edge.Src, edge.Dest,
	).Scan(&edge.ID, &edge.UpdatedAt)
	if err != nil {
		if isForeignKeyViolationError(err) {
			err = graph.ErrUnknownEdgeLinks
		}

		return fmt.Errorf("upsert edge: %w", err)
	}

	return nil
}

// RemoveStaleEdges removes any edge that originates from a specific link ID
// and was updated before the specified [updatedBefore] time.
func (s *CockroachDBGraph) RemoveStaleEdges(
	fromID uuid.UUID, updatedBefore time.Time,
) error {

	_, err := s.db.Exec(removeStaleEdgesQuery, fromID, updatedBefore)
	if err != nil {
		return fmt.Errorf("remove stale edges: %w", err)
	}

	return nil
}

// Edges returns an iterator for a set of edges whose source link / vertex id's
// belong to the [fromID, toID] range and were updated before the
// [updatedBefore] time.
func (s CockroachDBGraph) Edges(
	fromID, toID uuid.UUID, updatedBefore time.Time,
) (graph.EdgeIterator, error) {

	rows, err := s.db.Query(
		partitionedEdgesQuery, fromID, toID, updatedBefore.UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("edges: %w", err)
	}

	return &edgeIterator{rows: rows}, nil
}

// isForeignKeyViolationError returns true if error is a foreign key
// constraint violation error.
func isForeignKeyViolationError(err error) bool {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return false
	}

	return pqErr.Code.Name() == "foreign_key_violation"
}
