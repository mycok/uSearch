package rpc_test

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/textindexer/index"
	"github.com/mycok/uSearch/textindexer/store/api/rpc"
	"github.com/mycok/uSearch/textindexer/store/api/rpc/mocks"
	"github.com/mycok/uSearch/textindexer/store/api/rpc/proto"
)

var _ = check.Suite(new(ClientTestSuite))

type ClientTestSuite struct{}

func (s *ClientTestSuite) TestIndex(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockTextIndexerClient(ctrl)

	now := time.Now().Truncate(time.Second).UTC()
	doc := &index.Document{
		LinkID:  uuid.New(),
		URL:     "http://example.com",
		Title:   "Title",
		Content: "Lorem Ipsum",
	}

	rpcClient.EXPECT().Index(
		gomock.AssignableToTypeOf(context.TODO()),
		&proto.Document{
			LinkId:  doc.LinkID[:],
			Url:     doc.URL,
			Title:   doc.Title,
			Content: doc.Content,
		},
	).Return(
		&proto.Document{
			LinkId:    doc.LinkID[:],
			Url:       doc.URL,
			Title:     doc.Title,
			Content:   doc.Content,
			IndexedAt: encodeTimestamp(now),
		},
		nil,
	)

	client := rpc.NewTextIndexerClient(context.TODO(), rpcClient)
	err := client.Index(doc)
	c.Assert(err, check.IsNil)
	c.Assert(doc.IndexedAt, check.Equals, now)
}

func (s *ClientTestSuite) TestUpdateScore(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockTextIndexerClient(ctrl)

	linkID := uuid.New()

	rpcClient.EXPECT().UpdateScore(
		gomock.AssignableToTypeOf(context.TODO()),
		&proto.UpdateScoreRequest{
			LinkId:        linkID[:],
			PageRankScore: 0.5,
		},
	).Return(new(emptypb.Empty), nil)

	cli := rpc.NewTextIndexerClient(context.TODO(), rpcClient)
	err := cli.UpdateScore(linkID, 0.5)
	c.Assert(err, check.IsNil)
}

func (s *ClientTestSuite) TestSearch(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	rpcClient := mocks.NewMockTextIndexerClient(ctrl)
	docStream := mocks.NewMockTextIndexer_SearchClient(ctrl)

	ctxWithCancel, cancelFn := context.WithCancel(context.TODO())
	defer cancelFn()

	rpcClient.EXPECT().Search(
		gomock.AssignableToTypeOf(ctxWithCancel),
		&proto.Query{Type: proto.Query_MATCH, Expression: "foo"},
	).Return(docStream, nil)

	now := time.Now().Truncate(time.Second).UTC()
	linkIDs := [2]uuid.UUID{uuid.New(), uuid.New()}

	returns := [][]interface{}{
		{&proto.QueryResult{Result: &proto.QueryResult_DocCount{DocCount: 2}}, nil},
		{&proto.QueryResult{Result: &proto.QueryResult_Doc{
			Doc: &proto.Document{
				LinkId:    linkIDs[0][:],
				Url:       "url-0",
				Title:     "title-0",
				Content:   "content-0",
				IndexedAt: encodeTimestamp(now),
			},
		}}, nil},
		{&proto.QueryResult{Result: &proto.QueryResult_Doc{
			Doc: &proto.Document{
				LinkId:    linkIDs[1][:],
				Url:       "url-1",
				Title:     "title-1",
				Content:   "content-1",
				IndexedAt: encodeTimestamp(now),
			},
		}}, nil},
		{nil, io.EOF},
	}

	docStream.EXPECT().Recv().DoAndReturn(
		func() (interface{}, interface{}) {
			next := returns[0]
			returns = returns[1:]
			return next[0], next[1]
		},
	).Times(len(returns))

	client := rpc.NewTextIndexerClient(context.TODO(), rpcClient)
	it, err := client.Search(index.Query{Type: index.QueryTypeMatch, Expression: "foo"})
	c.Assert(err, check.IsNil)
	c.Assert(it.TotalCount(), check.Equals, uint64(2))

	var docCount int
	for it.Next() {
		next := it.Document()
		c.Assert(next.LinkID, check.DeepEquals, linkIDs[docCount])
		c.Assert(next.URL, check.Equals, fmt.Sprintf("url-%d", docCount))
		c.Assert(next.Title, check.Equals, fmt.Sprintf("title-%d", docCount))
		c.Assert(next.Content, check.Equals, fmt.Sprintf("content-%d", docCount))
		c.Assert(next.IndexedAt, check.Equals, now)

		docCount++
	}

	err = it.Error()
	if err != nil && err != io.EOF {
		c.Errorf("Unexpected error: %v", err)
	}
	c.Assert(it.Close(), check.IsNil)
	c.Assert(docCount, check.Equals, 2)
}
