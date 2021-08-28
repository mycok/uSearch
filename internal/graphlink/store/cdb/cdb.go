package cdb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mycok/uSearch/internal/graphlink/graph"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	upsertLinkQuery = `
					INSERT INTO links (url, retrieved_at)
					VALUES ($1, $2)
					ON CONFLICT (url) DO UPDATE SET retrieved_at=GREATEST(links.retrieved_at, $2)
					RETURNING id, retrieved_at
					`

	findLinkQuery = "SELECT id, url, retrieved_at FROM links WHERE id=$1"

	linksInPartitionsQuery = `
							SELECT id, url, retrieved_at FROM links 
							WHERE id >= $1
							AND id < $2
							AND retrieved_at < $3
							`

	upsertEdgeQuery = `
					INSERT INTO edges (src, dest, updated_at)
					VALUES ($1, $2, NOW())
					ON CONFLICT (src, dest) DO UPDATE SET updated_at=NOW()
					RETURNING id, updated_at
					`

	edgesInPartionsQuery = `
							SELECT id, src, dest, updated_at FROM edges
							WHERE src >= $1
							AND src < $2
							AND updated_at < $3
							`

	removeStaleEdgesQuery = "DELETE FROM edges WHERE src=$1 AND updated_at < $2"
)

// Compile-time check for ensuring CockroachDbGraph implements Graph.
var _ graph.Graph = (*CockroachDBGraph)(nil)

// CockroachDBGraph implements a Graph concrete type that persists its links
//  and edges to a cockroachdb instance.
type CockroachDBGraph struct {
	db *sql.DB
}

// NewCockroachDbGraph returns a CockroachDbGraph instance that connects to the cockroachdb
// instance specified by dsn.
func NewCockroachDbGraph(dsn string) (*CockroachDBGraph, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return &CockroachDBGraph{db: db}, nil
}

// Close terminates the connection to the cockroachdb instance.
func (c *CockroachDBGraph) Close() error {
	return c.db.Close()
}

// UpsertLink creates a new or updates an existing link.
func (c *CockroachDBGraph) UpsertLink(link *graph.Link) error {
	err := c.db.QueryRow(upsertLinkQuery, link.URL, link.RetrievedAt.UTC()).Scan(&link.ID, &link.RetrievedAt)
	if err != nil {
		return fmt.Errorf("upsert link: %w", err)
	}

	link.RetrievedAt = link.RetrievedAt.UTC()

	return nil
}

// FindLink performs a Link lookup by ID and returns an error if no
// match is found.
func (c *CockroachDBGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	link := &graph.Link{ID: id}

	err := c.db.QueryRow(findLinkQuery, id).Scan(&link.URL, &link.RetrievedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("find link: %w", graph.ErrNotFound)
		}

		return nil, fmt.Errorf("find link: %w", err)
	}

	return link, nil
}

// Links returns an alterator for a set of links whose id's belong
// to the [fromID, toID] range and were retrieved before the [retrievedBefore] time.
func (c *CockroachDBGraph) Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error) {
	rows, err := c.db.Query(linksInPartitionsQuery, fromID, toID, retrievedBefore.UTC())
	if err != nil {
		return nil, fmt.Errorf("links: %w", err)
	}

	return &LinkIterator{rows: rows}, nil
}

// UpsertEdge creates a new or updates an existing edge.
func (c *CockroachDBGraph) UpsertEdge(edge *graph.Edge) error {
	err := c.db.QueryRow(upsertEdgeQuery, edge.Src, edge.Dest).Scan(&edge.ID, &edge.UpdatedAt)
	if err != nil {
		if isForeignKeyViolationError(err) {
			err = graph.ErrUnknownEdgeLinks
		}

		return fmt.Errorf("upsert edge: %w", err)
	}

	edge.UpdatedAt = edge.UpdatedAt.UTC()

	return nil
}

// Edges returns an iterator for a set of edges whose source vertex id's
// belong to the [fromID, toID] range and were updated before the [updatedBefore] time.
func (c *CockroachDBGraph) Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (graph.EdgeIterator, error) {
	rows, err := c.db.Query(edgesInPartionsQuery, fromID, toID, updatedBefore.UTC())
	if err != nil {
		return nil, fmt.Errorf("edges: %w", err)
	}

	return &EdgeIterator{rows: rows}, nil
}

// RemoveStaleEdges removes edges that belong to the provided link id [fromID]
// and were updated before the [updatedBefore] time
func (c *CockroachDBGraph) RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error {
	_, err := c.db.Exec(removeStaleEdgesQuery, fromID, updatedBefore.UTC())
	if err != nil {
		return fmt.Errorf("removeStaleEdges: %w", err)
	}

	return nil
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
