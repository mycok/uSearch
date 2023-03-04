package rpc_test

import (
	"context"
	"io"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc/mocks"
	"github.com/mycok/uSearch/linkgraph/store/api/rpc/proto"
)

var _ = check.Suite(new(ClientTestSuite))

type ClientTestSuite struct{}

func (s *ClientTestSuite) TestUpsertLink(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockLinkGraphClient(ctrl)

	now := time.Now().Truncate(time.Second).UTC()

	link := &graph.Link{
		URL:         "http://www.example.com",
		RetrievedAt: now,
	}

	assignedID := uuid.New()

	rpcClient.EXPECT().UpsertLink(gomock.AssignableToTypeOf(context.TODO()),
		&proto.Link{
			Uuid:        uuid.Nil[:],
			Url:         link.URL,
			RetrievedAt: encodeTimestamp(link.RetrievedAt),
		},
	).Return(&proto.Link{
		Uuid:        assignedID[:],
		Url:         link.URL,
		RetrievedAt: encodeTimestamp(link.RetrievedAt),
	}, nil)

	client := rpc.NewLinkGraphClient(context.TODO(), rpcClient)
	err := client.UpsertLink(link)
	c.Assert(err, check.IsNil)
	c.Assert(link.ID, check.DeepEquals, assignedID)
	c.Assert(link.RetrievedAt, check.Equals, now)
}

func (s *ClientTestSuite) TestUpsertEdge(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockLinkGraphClient(ctrl)

	edge := &graph.Edge{
		Src:  uuid.New(),
		Dest: uuid.New(),
	}

	assignedID := uuid.New()

	rpcClient.EXPECT().UpsertEdge(
		gomock.AssignableToTypeOf(context.TODO()),
		&proto.Edge{
			Uuid:     uuid.Nil[:],
			SrcUuid:  edge.Src[:],
			DestUuid: edge.Dest[:],
		},
	).Return(
		&proto.Edge{
			Uuid:      assignedID[:],
			SrcUuid:   edge.Src[:],
			DestUuid:  edge.Dest[:],
			UpdatedAt: timestamppb.Now(),
		},
		nil,
	)

	client := rpc.NewLinkGraphClient(context.TODO(), rpcClient)

	err := client.UpsertEdge(edge)
	c.Assert(err, check.IsNil)
	c.Assert(edge.ID, check.DeepEquals, assignedID)
}

func (s *ClientTestSuite) TestLinks(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockLinkGraphClient(ctrl)
	linkStream := mocks.NewMockLinkGraph_LinksClient(ctrl)

	ctxWithCancel, cancelFn := context.WithCancel(context.TODO())
	defer cancelFn()

	now := time.Now().Truncate(time.Second).UTC()

	rpcClient.EXPECT().Links(
		gomock.AssignableToTypeOf(ctxWithCancel),
		&proto.Range{
			FromUuid:   minUUID[:],
			ToUuid:     maxUUID[:],
			TimeFilter: encodeTimestamp(now),
		},
	).Return(linkStream, nil)

	uuid1 := uuid.New()
	uuid2 := uuid.New()
	lastAccessed := encodeTimestamp(now)

	returns := [][]interface{}{
		{&proto.Link{
			Uuid:        uuid1[:],
			Url:         "http://example.com",
			RetrievedAt: lastAccessed,
		}, nil},
		{&proto.Link{
			Uuid:        uuid2[:],
			Url:         "http://example.com",
			RetrievedAt: lastAccessed,
		}, nil},
		{nil, io.EOF},
	}

	linkStream.EXPECT().Recv().DoAndReturn(
		func() (interface{}, interface{}) {
			next := returns[0]
			returns = returns[1:]
			return next[0], next[1]
		},
	).Times(len(returns))

	client := rpc.NewLinkGraphClient(context.TODO(), rpcClient)
	it, err := client.Links(minUUID, maxUUID, now)
	c.Assert(err, check.IsNil)

	var linkCount int
	for it.Next() {
		linkCount++
		link := it.Link()

		if link.ID != uuid1 && link.ID != uuid2 {
			c.Fatalf("unexpected link with ID %q", link.ID)
		}
		c.Assert(link.URL, check.Equals, "http://example.com")
		c.Assert(link.RetrievedAt, check.Equals, now)
	}

	err = it.Error()
	if err != nil && err != io.EOF {
		c.Errorf("Unexpected error: %v", err)
	}
	c.Assert(it.Close(), check.IsNil)
	c.Assert(linkCount, check.Equals, 2)
}

func (s *ClientTestSuite) TestEdges(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockLinkGraphClient(ctrl)
	edgeStream := mocks.NewMockLinkGraph_EdgesClient(ctrl)

	ctxWithCancel, cancelFn := context.WithCancel(context.TODO())
	defer cancelFn()

	now := time.Now().Truncate(time.Second).UTC()

	rpcClient.EXPECT().Edges(
		gomock.AssignableToTypeOf(ctxWithCancel),
		&proto.Range{
			FromUuid:   minUUID[:],
			ToUuid:     maxUUID[:],
			TimeFilter: encodeTimestamp(now),
		},
	).Return(edgeStream, nil)

	uuid1 := uuid.New()
	uuid2 := uuid.New()
	srcID := uuid.New()
	destID := uuid.New()
	updatedAt := time.Now().UTC()

	returns := [][]interface{}{
		{&proto.Edge{
			Uuid:      uuid1[:],
			SrcUuid:   srcID[:],
			DestUuid:  destID[:],
			UpdatedAt: encodeTimestamp(updatedAt),
		}, nil},
		{&proto.Edge{
			Uuid:      uuid2[:],
			SrcUuid:   srcID[:],
			DestUuid:  destID[:],
			UpdatedAt: encodeTimestamp(updatedAt),
		}, nil},
		{nil, io.EOF},
	}

	edgeStream.EXPECT().Recv().DoAndReturn(
		func() (interface{}, interface{}) {
			next := returns[0]
			returns = returns[1:]
			return next[0], next[1]
		},
	).Times(len(returns))

	client := rpc.NewLinkGraphClient(context.TODO(), rpcClient)
	it, err := client.Edges(minUUID, maxUUID, now)
	c.Assert(err, check.IsNil)

	var edgeCount int
	for it.Next() {
		edgeCount++
		next := it.Edge()

		if next.ID != uuid1 && next.ID != uuid2 {
			c.Fatalf("unexpected edge with ID %q", next.ID)
		}
		c.Assert(next.Src, check.Equals, srcID)
		c.Assert(next.Dest, check.Equals, destID)
		c.Assert(next.UpdatedAt, check.Equals, updatedAt)
	}

	err = it.Error()
	if err != nil && err != io.EOF {
		c.Errorf("Unexpected error: %v", err)
	}
	c.Assert(it.Close(), check.IsNil)
	c.Assert(edgeCount, check.Equals, 2)
}

func (s *ClientTestSuite) TestRetainVersionedEdges(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockLinkGraphClient(ctrl)
	from := uuid.New()
	now := time.Now()

	rpcClient.EXPECT().RemoveStaleEdges(
		gomock.AssignableToTypeOf(context.TODO()),
		&proto.RemoveStaleEdgesQuery{
			FromUuid:      from[:],
			UpdatedBefore: encodeTimestamp(now),
		},
	).Return(new(emptypb.Empty), nil)

	client := rpc.NewLinkGraphClient(context.TODO(), rpcClient)
	err := client.RemoveStaleEdges(from, now)
	c.Assert(err, check.IsNil)
}
