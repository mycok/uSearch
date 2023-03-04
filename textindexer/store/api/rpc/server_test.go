package rpc_test

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/textindexer/index"
	"github.com/mycok/uSearch/textindexer/store/api/rpc"
	"github.com/mycok/uSearch/textindexer/store/api/rpc/proto"
	"github.com/mycok/uSearch/textindexer/store/memory"
)

var _ = check.Suite(new(ServerTestSuite))

type indexerCloser interface {
	io.Closer
	index.Indexer
}

type ServerTestSuite struct {
	idx         indexerCloser
	netListener *bufconn.Listener
	grpcSrv     *grpc.Server
	clientConn  *grpc.ClientConn
	client      proto.TextIndexerClient
}

func (s *ServerTestSuite) SetUpTest(c *check.C) {
	var err error

	s.idx, err = memory.NewInMemoryIndex()
	c.Assert(err, check.IsNil)

	s.netListener = bufconn.Listen(1024)
	s.grpcSrv = grpc.NewServer()
	proto.RegisterTextIndexerServer(s.grpcSrv, rpc.NewTextIndexerServer(s.idx))

	// Launch a grpc server
	go func() {
		err := s.grpcSrv.Serve(s.netListener)
		c.Assert(err, check.IsNil)
	}()

	// Create a grpc client connection.
	s.clientConn, err = grpc.Dial(
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return s.netListener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	c.Assert(err, check.IsNil)

	s.client = proto.NewTextIndexerClient(s.clientConn)
}

func (s *ServerTestSuite) TearDownTest(c *check.C) {
	_ = s.clientConn.Close()
	s.grpcSrv.Stop()
	_ = s.netListener.Close()
	_ = s.idx.Close()
}

func (s *ServerTestSuite) TestIndex(c *check.C) {
	linkID := uuid.New()
	doc := &proto.Document{
		LinkId:  linkID[:],
		Url:     "http://example.com",
		Title:   "Test",
		Content: "Lorem Ipsum",
	}
	res, err := s.client.Index(context.TODO(), doc)
	c.Assert(err, check.IsNil)
	c.Assert(res.Url, check.Equals, doc.Url)
	c.Assert(res.Title, check.Equals, doc.Title)
	c.Assert(res.Content, check.Equals, doc.Content)

	// Check that document has been correctly indexed
	indexedDoc, err := s.idx.FindByID(linkID)
	c.Assert(err, check.IsNil)
	c.Assert(indexedDoc.URL, check.Equals, doc.Url)
	c.Assert(indexedDoc.Title, check.Equals, doc.Title)
	c.Assert(indexedDoc.Content, check.Equals, doc.Content)
	c.Assert(indexedDoc.IndexedAt.Unix(), check.Not(check.Equals), 0)
}

func (s *ServerTestSuite) TestReIndex(c *check.C) {
	// Manually index test document
	linkID := uuid.New()
	doc := &index.Document{
		LinkID:  linkID,
		URL:     "http://example.com",
		Title:   "Test",
		Content: "Lorem Ipsum",
	}
	c.Assert(s.idx.Index(doc), check.IsNil)

	// Re-index existing document
	req := &proto.Document{
		LinkId:  doc.LinkID[:],
		Url:     "http://foo.com",
		Title:   "Bar",
		Content: "Baz",
	}

	res, err := s.client.Index(context.TODO(), req)
	c.Assert(err, check.IsNil)
	c.Assert(res.LinkId, check.DeepEquals, doc.LinkID[:])
	c.Assert(res.Url, check.Equals, req.Url)
	c.Assert(res.Title, check.Equals, req.Title)
	c.Assert(res.Content, check.Equals, req.Content)

	// Check that document has been correctly re-indexed
	indexedDoc, err := s.idx.FindByID(linkID)
	c.Assert(err, check.IsNil)
	c.Assert(indexedDoc.URL, check.Equals, res.Url)
	c.Assert(indexedDoc.Title, check.Equals, res.Title)
	c.Assert(indexedDoc.Content, check.Equals, res.Content)
}

func (s *ServerTestSuite) TestUpdateScore(c *check.C) {
	// Manually index test document
	linkID := uuid.New()
	doc := &index.Document{
		LinkID:  linkID,
		URL:     "http://example.com",
		Title:   "Test",
		Content: "Lorem Ipsum",
	}
	c.Assert(s.idx.Index(doc), check.IsNil)

	// Update PageRank score and check that the document has been updated
	req := &proto.UpdateScoreRequest{
		LinkId:        linkID[:],
		PageRankScore: 0.5,
	}
	_, err := s.client.UpdateScore(context.TODO(), req)
	c.Assert(err, check.IsNil)

	indexedDoc, err := s.idx.FindByID(linkID)
	c.Assert(err, check.IsNil)
	c.Assert(indexedDoc.PageRank, check.Equals, 0.5)
}

func (s *ServerTestSuite) TestSearch(c *check.C) {
	idList := s.indexDocs(c, 100)

	stream, err := s.client.Search(context.TODO(), &proto.Query{
		Type:       proto.Query_MATCH,
		Expression: "Test",
	})
	c.Assert(err, check.IsNil)

	s.assertSearchResultsMatchList(c, stream, 100, idList)
}

func (s *ServerTestSuite) TestSearchWithOffset(c *check.C) {
	idList := s.indexDocs(c, 100)

	stream, err := s.client.Search(context.TODO(), &proto.Query{
		Type:       proto.Query_MATCH,
		Expression: "Test",
		Offset:     50,
	})
	c.Assert(err, check.IsNil)

	s.assertSearchResultsMatchList(c, stream, 100, idList[50:])
}

func (s *ServerTestSuite) TestSearchWithOffsetAfterEndOfResultSet(c *check.C) {
	_ = s.indexDocs(c, 100)

	stream, err := s.client.Search(context.TODO(), &proto.Query{
		Type:       proto.Query_MATCH,
		Expression: "Test",
		Offset:     101,
	})
	c.Assert(err, check.IsNil)

	s.assertSearchResultsMatchList(c, stream, 100, nil)
}

func (s *ServerTestSuite) assertSearchResultsMatchList(
	c *check.C, stream proto.TextIndexer_SearchClient,
	expTotalCount int, expIDList []uuid.UUID,
) {

	// First message should be the result count
	next, err := stream.Recv()
	c.Assert(err, check.IsNil)
	c.Assert(next.GetDoc(), check.IsNil, check.Commentf("expected first message should contain result count only"))
	c.Assert(int(next.GetDocCount()), check.Equals, expTotalCount)

	var docCount int
	for {
		result, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			c.Fatal(err)
		}

		doc := result.GetDoc()
		linkID, err := uuid.FromBytes(doc.LinkId)
		c.Assert(err, check.IsNil)
		c.Assert(expIDList[docCount], check.Equals, linkID)

		docCount++
	}

	c.Assert(docCount, check.Equals, len(expIDList))
}

func (s *ServerTestSuite) indexDocs(c *check.C, count int) []uuid.UUID {
	idList := make([]uuid.UUID, count)
	for i := 0; i < count; i++ {
		linkID := uuid.New()
		idList[i] = linkID
		err := s.idx.Index(&index.Document{
			LinkID:  linkID,
			URL:     fmt.Sprintf("http://example.com/%d", i),
			Title:   fmt.Sprintf("Test-%d", i),
			Content: "Lorem Ipsum",
		})
		c.Assert(err, check.IsNil)

		// Assign decending scores so documents sort in correct order
		// in search results.
		c.Assert(s.idx.UpdateScore(linkID, float64(count-i)), check.IsNil)
	}

	return idList
}
