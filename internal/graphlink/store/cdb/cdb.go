package cdb

import (
	"database/sql"
	"fmt"

	"github.com/mycok/uSearch/internal/graphlink/graph"
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
// var _ graph.Graph = (*CockroachDBGraph)(nil)

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
	row := c.db.QueryRow(upsertLinkQuery, link.URL, link.RetrievedAt.UTC())
	if err := row.Scan(&link.ID, &link.RetrievedAt); err != nil {
		return fmt.Errorf("upsert link: %w", err)
	}

	link.RetrievedAt = link.RetrievedAt.UTC()

	return nil
}
