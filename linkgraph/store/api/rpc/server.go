package rpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc/proto"
)

var _ proto.LinkGraphServer = (*LinkGraphServer)(nil)

// LinkGraphServer provides a gRPC wrapper for accessing a link graph.
type LinkGraphServer struct {
	// Any concrete type that satisfies the graph.Graph interface.
	g graph.Graph
	proto.UnimplementedLinkGraphServer
}

// NewLinkGraphServer returns a new server instance that uses the provided
// graph as its backing store.
func NewLinkGraphServer(g graph.Graph) *LinkGraphServer {
	return &LinkGraphServer{g: g}
}

// UpsertLink creates a new or updates an existing link.
func (s *LinkGraphServer) UpsertLink(
	_ context.Context, req *proto.Link,
) (*proto.Link, error) {

	var err error
	link := graph.Link{
		ID:  uuidFromBytes(req.Uuid),
		URL: req.Url,
	}

	if err = req.RetrievedAt.CheckValid(); err != nil {
		return nil, err
	}

	link.RetrievedAt = req.RetrievedAt.AsTime()

	if err = s.g.UpsertLink(&link); err != nil {
		return nil, err
	}

	req.RetrievedAt = timeToProto(link.RetrievedAt)
	req.Url = link.URL
	req.Uuid = link.ID[:]

	return req, nil
}

// UpsertEdge creates a new or updates an existing edge.
func (s *LinkGraphServer) UpsertEdge(
	_ context.Context, req *proto.Edge,
) (*proto.Edge, error) {

	edge := graph.Edge{
		ID:   uuidFromBytes(req.Uuid),
		Src:  uuidFromBytes(req.SrcUuid),
		Dest: uuidFromBytes(req.DestUuid),
	}

	if err := s.g.UpsertEdge(&edge); err != nil {
		return nil, err
	}

	req.Uuid = edge.ID[:]
	req.SrcUuid = edge.Src[:]
	req.DestUuid = edge.Dest[:]
	req.UpdatedAt = timeToProto(edge.UpdatedAt)

	return req, nil
}

// Links streams the set of links in the specified ID range.
func (s *LinkGraphServer) Links(
	idRange *proto.Range, stream proto.LinkGraph_LinksServer,
) error {

	if err := idRange.TimeFilter.CheckValid(); err != nil {
		return err
	}

	fromID, err := uuid.FromBytes(idRange.FromUuid)
	if err != nil {
		return err
	}

	toID, err := uuid.FromBytes(idRange.ToUuid)
	if err != nil {
		return err
	}

	it, err := s.g.Links(fromID, toID, idRange.TimeFilter.AsTime())
	if err != nil {
		return err
	}
	defer func() { _ = it.Close() }()

	for it.Next() {
		link := it.Link()
		msg := proto.Link{
			Uuid:        link.ID[:],
			Url:         link.URL,
			RetrievedAt: timeToProto(link.RetrievedAt),
		}

		if err := stream.Send(&msg); err != nil {
			_ = it.Close()

			return err
		}
	}

	if err = it.Error(); err != nil {
		return err
	}

	return it.Close()
}

// Edges streams the set of edges in the specified ID range.
func (s *LinkGraphServer) Edges(
	idRange *proto.Range, stream proto.LinkGraph_EdgesServer,
) error {

	if err := idRange.TimeFilter.CheckValid(); err != nil {
		return err
	}

	fromID, err := uuid.FromBytes(idRange.FromUuid)
	if err != nil {
		return err
	}

	toID, err := uuid.FromBytes(idRange.ToUuid)
	if err != nil {
		return err
	}

	it, err := s.g.Edges(fromID, toID, idRange.TimeFilter.AsTime())
	if err != nil {
		return err
	}
	defer func() { _ = it.Close() }()

	for it.Next() {
		edge := it.Edge()
		msg := proto.Edge{
			Uuid:      edge.ID[:],
			SrcUuid:   edge.Src[:],
			DestUuid:  edge.Dest[:],
			UpdatedAt: timeToProto(edge.UpdatedAt),
		}

		if err := stream.Send(&msg); err != nil {
			_ = it.Close()

			return err
		}
	}

	if err = it.Error(); err != nil {
		return err
	}

	return it.Close()
}

// RemoveStaleEdges removes any edge that originates from the specified
// link ID and was updated before the specified timestamp.
func (s *LinkGraphServer) RemoveStaleEdges(
	_ context.Context, req *proto.RemoveStaleEdgesQuery,
) (*emptypb.Empty, error) {

	var err error

	if err = req.UpdatedBefore.CheckValid(); err != nil {
		return nil, err
	}

	err = s.g.RemoveStaleEdges(
		uuidFromBytes(req.FromUuid), req.UpdatedBefore.AsTime(),
	)

	return new(emptypb.Empty), err
}

func uuidFromBytes(b []byte) uuid.UUID {
	if len(b) != 16 {
		return uuid.Nil
	}

	var dest uuid.UUID
	_ = copy(dest[:], b)

	return dest
}

func timeToProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
