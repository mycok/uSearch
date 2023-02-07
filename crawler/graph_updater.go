package crawler

import (
	"context"
	"time"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/pipeline"
)

// Static and compile-time check to ensure graphUpdater implements
// pipeline.Processor interface.
var _ pipeline.Processor = (*graphUpdater)(nil)

type graphUpdater struct {
	graph MiniGraph
}

func newGraphUpdater(graph MiniGraph) *graphUpdater {
	return &graphUpdater{graph}
}

// Process updates the payload's retrievedAt timestamp,
// inserts / updates the payload's newly discovered links and no-follow links,
// inserts / updates the relevant edges originating from the payload and also
// removes stale edges originating from the payload.
func (p *graphUpdater) Process(
	ctx context.Context, payload pipeline.Payload,
) (pipeline.Payload, error) {

	cPayload, ok := payload.(*crawlerPayload)
	if !ok {
		return nil, nil
	}

	// Update the source link's retrievedAt field with the current time.
	srcLink := &graph.Link{
		ID:          cPayload.LinkID,
		URL:         cPayload.URL,
		RetrievedAt: time.Now(),
	}

	if err := p.graph.UpsertLink(srcLink); err != nil {
		return nil, err
	}

	// Upsert the discovered no-follow links without creating an edge that links
	// them with the source link.
	for _, url := range cPayload.NoFollowLinks {
		link := &graph.Link{URL: url}

		if err := p.graph.UpsertLink(link); err != nil {
			return nil, err
		}
	}

	// Upsert discovered links and create edges for them. Keep track of
	// the current time so we can drop stale edges that have not been
	// updated after this loop.
	updatedBefore := time.Now()
	for _, url := range cPayload.Links {
		link := &graph.Link{URL: url}

		// Upsert new link
		if err := p.graph.UpsertLink(link); err != nil {
			return nil, err
		}

		// Upsert an edge for the source link with the newly created link as
		// the destination.
		if err := p.graph.UpsertEdge(&graph.Edge{
			Src:  srcLink.ID,
			Dest: link.ID,
		}); err != nil {
			return nil, err
		}
	}

	// Drop any stale edges that were not updated during the edge upsert
	//  operation.
	if err := p.graph.RemoveStaleEdges(srcLink.ID, updatedBefore); err != nil {
		return nil, err
	}

	return cPayload, nil
}
