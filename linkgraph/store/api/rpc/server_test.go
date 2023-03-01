package rpc_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc/proto"
	"github.com/mycok/uSearch/linkgraph/store/memory"
)

var _ = check.Suite(new(ServerTestSuite))

var minUUID = uuid.Nil
var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

type ServerTestSuite struct {
	g           graph.Graph
	netListener *bufconn.Listener
	grpcSrv     *grpc.Server
	clientConn  *grpc.ClientConn
	client      proto.LinkGraphClient
}

func (s *ServerTestSuite) SetUpTest(c *check.C) {
	s.g = memory.NewInMemoryGraph()

	s.netListener = bufconn.Listen(1024)
	s.grpcSrv = grpc.NewServer()
	proto.RegisterLinkGraphServer(s.grpcSrv, rpc.NewLinkGraphServer(s.g))

	// Launch a grpc server
	go func() {
		err := s.grpcSrv.Serve(s.netListener)
		c.Assert(err, check.IsNil)
	}()

	var err error
	// Create a grpc client connection.
	s.clientConn, err = grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.netListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	c.Assert(err, check.IsNil)

	s.client = proto.NewLinkGraphClient(s.clientConn)
}

func (s *ServerTestSuite) TearDownTest(c *check.C) {
	_ = s.clientConn.Close()
	s.grpcSrv.Stop()
	_ = s.netListener.Close()
}

func (s *ServerTestSuite) TestInsertLink(c *check.C) {
	now := time.Now().Truncate(time.Second).UTC()
	req := &proto.Link{
		Url:         "http://example.com",
		RetrievedAt: encodeTimestamp(now),
	}

	result, err := s.client.UpsertLink(context.TODO(), req)
	c.Assert(err, check.IsNil)
	c.Assert(
		result.Uuid, check.Not(check.Equals), req.Uuid,
		check.Commentf("new link was not assigned a new UUID"),
	)
	c.Assert(decodeTimestamp(c, result.RetrievedAt), check.Equals, now)
}

func (s *ServerTestSuite) TestUpdateLink(c *check.C) {
	// Add a link to the graph.
	link := &graph.Link{URL: "http://example.com"}
	c.Assert(s.g.UpsertLink(link), check.IsNil)

	// Update the link.
	now := time.Now().Truncate(time.Second).UTC()
	req := &proto.Link{
		Uuid:        link.ID[:],
		Url:         "http://example.com",
		RetrievedAt: encodeTimestamp(now),
	}

	result, err := s.client.UpsertLink(context.TODO(), req)
	c.Assert(err, check.IsNil)
	c.Assert(
		result.Uuid, check.DeepEquals, link.ID[:],
		check.Commentf("ID of existing link modified during update"),
	)
	c.Assert(decodeTimestamp(c, result.RetrievedAt), check.Equals, now)
}

func (s *ServerTestSuite) TestInsertEdge(c *check.C) {
	// Add two links to the graph
	src := &graph.Link{URL: "http://example.com"}
	dest := &graph.Link{URL: "http://foo.com"}
	c.Assert(s.g.UpsertLink(src), check.IsNil)
	c.Assert(s.g.UpsertLink(dest), check.IsNil)

	// Create new edge
	req := &proto.Edge{
		SrcUuid:  src.ID[:],
		DestUuid: dest.ID[:],
	}

	res, err := s.client.UpsertEdge(context.TODO(), req)
	c.Assert(err, check.IsNil)
	c.Assert(
		res.Uuid, check.Not(check.DeepEquals), req.Uuid,
		check.Commentf("UUID not assigned to new edge"),
	)
	c.Assert(res.SrcUuid[:], check.DeepEquals, req.SrcUuid[:])
	c.Assert(res.DestUuid[:], check.DeepEquals, req.DestUuid[:])
	c.Assert(res.UpdatedAt.Seconds, check.Not(check.Equals), 0)
}

func (s *ServerTestSuite) TestUpdateEdge(c *check.C) {
	// Add two links and an edge to the graph
	src := &graph.Link{URL: "http://example.com"}
	dest := &graph.Link{URL: "http://foo.com"}
	c.Assert(s.g.UpsertLink(src), check.IsNil)
	c.Assert(s.g.UpsertLink(dest), check.IsNil)

	edge := &graph.Edge{Src: src.ID, Dest: dest.ID}
	c.Assert(s.g.UpsertEdge(edge), check.IsNil)

	// Touch the edge
	req := &proto.Edge{
		Uuid:     edge.ID[:],
		SrcUuid:  src.ID[:],
		DestUuid: dest.ID[:],
	}

	res, err := s.client.UpsertEdge(context.TODO(), req)
	c.Assert(err, check.IsNil)
	c.Assert(
		res.Uuid, check.DeepEquals, req.Uuid,
		check.Commentf("UUID for existing edge modified"),
	)
	c.Assert(res.SrcUuid[:], check.DeepEquals, req.SrcUuid[:])
	c.Assert(res.DestUuid[:], check.DeepEquals, req.DestUuid[:])
	c.Assert(
		decodeTimestamp(c, res.UpdatedAt).After(edge.UpdatedAt), check.Equals, true,
		check.Commentf(
			"expected UpdatedAt field to be set to a newer timestamp after updating edge"),
	)
}

func (s *ServerTestSuite) TestLinks(c *check.C) {
	// Add links to the graph
	seenLinks := make(map[uuid.UUID]bool)
	for i := 0; i < 100; i++ {
		link := &graph.Link{
			URL: fmt.Sprintf("http://example.com/%d", i),
		}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		seenLinks[link.ID] = false
	}

	filter := encodeTimestamp(time.Now().Add(time.Hour))
	stream, err := s.client.Links(
		context.TODO(), &proto.Range{FromUuid: minUUID[:],
			ToUuid: maxUUID[:], TimeFilter: filter},
	)
	c.Assert(err, check.IsNil)
	for {
		nextLink, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Fatal(err)
		}

		linkID, err := uuid.FromBytes(nextLink.Uuid)
		c.Assert(err, check.IsNil)

		seenID, exists := seenLinks[linkID]
		if !exists {
			c.Fatalf("saw unexpected link with ID %q", linkID)
		} else if seenID {
			c.Fatalf("saw duplicate link with ID %q", linkID)
		}
		seenLinks[linkID] = true
	}

	for linkID, seen := range seenLinks {
		if !seen {
			c.Fatalf("expected to see link with ID %q", linkID)
		}
	}
}

func (s *ServerTestSuite) TestEdges(c *check.C) {
	// Add links and edges to the graph
	links := make([]uuid.UUID, 100)
	for i := 0; i < len(links); i++ {
		link := &graph.Link{
			URL: fmt.Sprintf("http://example.com/%d", i),
		}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		links[i] = link.ID
	}
	seenEdges := make(map[uuid.UUID]bool)
	for i := 0; i < len(links); i++ {
		edge := &graph.Edge{
			Src:  links[0],
			Dest: links[i],
		}
		c.Assert(s.g.UpsertEdge(edge), check.IsNil)
		seenEdges[edge.ID] = false
	}

	filter := encodeTimestamp(time.Now().Add(time.Hour))
	stream, err := s.client.Edges(
		context.TODO(), &proto.Range{FromUuid: minUUID[:],
			ToUuid: maxUUID[:], TimeFilter: filter},
	)
	c.Assert(err, check.IsNil)
	for {
		nextEdge, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			c.Fatal(err)
		}

		edgeID, err := uuid.FromBytes(nextEdge.Uuid)
		c.Assert(err, check.IsNil)

		seenID, exists := seenEdges[edgeID]
		if !exists {
			c.Fatalf("saw unexpected edges with ID %q", edgeID)
		} else if seenID {
			c.Fatalf("saw duplicate edges with ID %q", edgeID)
		}
		seenEdges[edgeID] = true
	}

	for edgeID, seen := range seenEdges {
		if !seen {
			c.Fatalf("expected to see edge with ID %q", edgeID)
		}
	}
}

func (s *ServerTestSuite) TestRemoveStaleEdges(c *check.C) {
	// Add three links and and two edges to the graph with different versions
	src := &graph.Link{URL: "http://example.com"}
	dest1 := &graph.Link{URL: "http://foo.com"}
	dest2 := &graph.Link{URL: "http://bar.com"}
	c.Assert(s.g.UpsertLink(src), check.IsNil)
	c.Assert(s.g.UpsertLink(dest1), check.IsNil)
	c.Assert(s.g.UpsertLink(dest2), check.IsNil)

	edge1 := &graph.Edge{Src: src.ID, Dest: dest1.ID}
	c.Assert(s.g.UpsertEdge(edge1), check.IsNil)

	time.Sleep(100 * time.Millisecond)
	t1 := time.Now()
	edge2 := &graph.Edge{Src: src.ID, Dest: dest2.ID}
	c.Assert(s.g.UpsertEdge(edge2), check.IsNil)

	req := &proto.RemoveStaleEdgesQuery{
		FromUuid:      src.ID[:],
		UpdatedBefore: encodeTimestamp(t1),
	}
	_, err := s.client.RemoveStaleEdges(context.TODO(), req)
	c.Assert(err, check.IsNil)

	// Check that the (src, dst1) edge has been removed from the backing graph.
	it, err := s.g.Edges(minUUID, maxUUID, time.Now())
	c.Assert(err, check.IsNil)

	var edgeCount int
	for it.Next() {
		edgeCount++
		edge := it.Edge()
		c.Assert(
			edge.ID, check.Not(check.DeepEquals), edge1.ID,
			check.Commentf("expected edge1 to be dropper"),
		)
	}
	c.Assert(it.Error(), check.IsNil)
	c.Assert(edgeCount, check.Equals, 1)
}
