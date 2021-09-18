package crawler

import (
	"context"
	"time"

	"github.com/mycok/uSearch/internal/graphlink/graph"
	"github.com/mycok/uSearch/internal/pipeline"
)

// Compile-time check for ensuring that graphUpdater implements pipeline.Processor.
var _ pipeline.Processor = (*graphUpdater)(nil)

type graphUpdater struct {
	updater MiniGraph
}

func newGraphUpdater(updater MiniGraph) *graphUpdater {
	return &graphUpdater{
		updater: updater,
	}
}

func (gu *graphUpdater) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	// We perform an update since it's the same link retrieved from the graphLink store by
	// the linkSource / source object.
	srcLink := &graph.Link{
		ID:          payload.LinkID,
		URL:         payload.URL,
		RetrievedAt: time.Now(),
	}

	if err := gu.updater.UpsertLink(srcLink); err != nil {
		return nil, err
	}

	// Upsert discovered no-follow links without creating an edge.
	for _, destLink := range payload.NoFollowLinks {
		dest := &graph.Link{URL: destLink}

		if err := gu.updater.UpsertLink(dest); err != nil {
			return nil, err
		}
	}

	// Upsert discovered links and create edges for them. Keep track of
	// the current time so we can drop stale edges that have not been
	// updated after this loop.
	edgesOlderThan := time.Now()
	for _, destLink := range payload.Links {
		dest := &graph.Link{URL: destLink}

		if err := gu.updater.UpsertLink(dest); err != nil {
			return nil, err
		}

		if err := gu.updater.UpsertEdge(&graph.Edge{Src: srcLink.ID, Dest: dest.ID}); err != nil {
			return nil, err
		}
	}

	// Drop any stale edges that were not updated during the edge upserting operation.
	if err := gu.updater.RemoveStaleEdges(srcLink.ID, edgesOlderThan); err != nil {
		return nil, err
	}

	return p, nil
}
