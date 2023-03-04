package rpc

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc/proto"
)

// LinkGraphClient provides an API that wraps the graph.Graph interface
// for accessing graph data store instances exposed by a remote gRPC server.
type LinkGraphClient struct {
	ctx       context.Context
	rpcClient proto.LinkGraphClient
}

// NewLinkGraphClient configures and returns a LinkGraphClient instance.
func NewLinkGraphClient(
	ctx context.Context, rpcClient proto.LinkGraphClient,
) *LinkGraphClient {

	return &LinkGraphClient{
		ctx:       ctx,
		rpcClient: rpcClient,
	}
}

// UpsertLink creates a new or updates an existing link.
func (c *LinkGraphClient) UpsertLink(link *graph.Link) error {
	req := &proto.Link{
		Uuid:        link.ID[:],
		Url:         link.URL,
		RetrievedAt: timeToProto(link.RetrievedAt),
	}

	result, err := c.rpcClient.UpsertLink(c.ctx, req)
	if err != nil {
		return err
	}

	link.ID = uuidFromBytes(result.Uuid)
	link.URL = result.Url
	link.RetrievedAt = result.RetrievedAt.AsTime()

	return nil
}

// UpsertEdge creates a new or updates an existing edge.
func (c *LinkGraphClient) UpsertEdge(edge *graph.Edge) error {
	req := &proto.Edge{
		Uuid:     edge.ID[:],
		SrcUuid:  edge.Src[:],
		DestUuid: edge.Dest[:],
	}
	result, err := c.rpcClient.UpsertEdge(c.ctx, req)
	if err != nil {
		return err
	}

	edge.ID = uuidFromBytes(result.Uuid)
	edge.UpdatedAt = result.UpdatedAt.AsTime()

	return nil
}

// Links returns an iterator for a set of links whose id's belong
// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
// time.
func (c *LinkGraphClient) Links(
	fromID, toID uuid.UUID, retrievedBefore time.Time,
) (graph.LinkIterator, error) {

	filter := timeToProto(retrievedBefore)

	req := &proto.Range{
		FromUuid:   fromID[:],
		ToUuid:     toID[:],
		TimeFilter: filter,
	}

	ctx, cancel := context.WithCancel(c.ctx)
	stream, err := c.rpcClient.Links(ctx, req)
	if err != nil {
		// Cancel the context
		cancel()

		return nil, err
	}

	return &linkIterator{
		stream:   stream,
		cancelFn: cancel,
	}, nil
}

// Edges returns an iterator for a set of edges whose source vertex id's
// belong to the [fromID, toID] range and were updated before the
// [updatedBefore] time.
func (c *LinkGraphClient) Edges(
	fromID, toID uuid.UUID, updatedBefore time.Time,
) (graph.EdgeIterator, error) {

	filter := timeToProto(updatedBefore)

	req := &proto.Range{
		FromUuid:   fromID[:],
		ToUuid:     toID[:],
		TimeFilter: filter,
	}

	ctx, cancel := context.WithCancel(c.ctx)
	stream, err := c.rpcClient.Edges(ctx, req)
	if err != nil {
		// Cancel the context
		cancel()
		return nil, err
	}

	return &edgeIterator{
		stream:   stream,
		cancelFn: cancel,
	}, nil
}

// RemoveStaleEdges removes any edge that originates from a specific link ID
// and was updated before the specified [updatedBefore] time.
func (c *LinkGraphClient) RemoveStaleEdges(
	fromID uuid.UUID, updatedBefore time.Time,
) error {

	req := &proto.RemoveStaleEdgesQuery{
		FromUuid:      fromID[:],
		UpdatedBefore: timeToProto(updatedBefore),
	}

	_, err := c.rpcClient.RemoveStaleEdges(c.ctx, req)

	return err
}

type linkIterator struct {
	stream   proto.LinkGraph_LinksClient
	link     *graph.Link
	lastErr  error
	cancelFn func()
}

// Next loads the next item, returns false when no more links
// are available or when an error occurs.
func (i *linkIterator) Next() bool {
	result, err := i.stream.Recv()
	if err != nil {
		if err == io.EOF {
			i.lastErr = err
		}

		// Cancel the provided context.
		i.cancelFn()

		return false
	}

	i.link = &graph.Link{
		ID:          uuidFromBytes(result.Uuid),
		URL:         result.Url,
		RetrievedAt: result.RetrievedAt.AsTime(),
	}

	return true
}

// Error returns the last error encountered by the iterator.
func (i *linkIterator) Error() error {
	return i.lastErr
}

// Close releases any resources allocated to the iterator.
func (i *linkIterator) Close() error {
	// Cancel the context.
	i.cancelFn()

	return nil
}

// Link returns the currently fetched link object.
func (i *linkIterator) Link() *graph.Link {
	return i.link
}

// edgeIterator is a graph.EdgeIterator implementation for the in-memory graph.
type edgeIterator struct {
	stream   proto.LinkGraph_EdgesClient
	lastErr  error
	edge     *graph.Edge
	cancelFn func()
}

// Next advances the iterator. When no edges are available or when an
// error occurs, calls to Next() return false.
func (i *edgeIterator) Next() bool {
	result, err := i.stream.Recv()
	if err != nil {
		if err == io.EOF {
			i.lastErr = err
		}

		// Cancel the provided context.
		i.cancelFn()

		return false
	}

	i.edge = &graph.Edge{
		ID:        uuidFromBytes(result.Uuid),
		Src:       uuidFromBytes(result.SrcUuid),
		Dest:      uuidFromBytes(result.DestUuid),
		UpdatedAt: result.UpdatedAt.AsTime(),
	}

	return true
}

// Error returns the last error recorded by the iterator.
func (i *edgeIterator) Error() error {
	return i.lastErr
}

// Close releases any resources linked to the iterator.
func (i *edgeIterator) Close() error {
	i.cancelFn()

	return nil
}

// Edge returns the currently fetched edge object.
func (i *edgeIterator) Edge() *graph.Edge {
	return i.edge
}
