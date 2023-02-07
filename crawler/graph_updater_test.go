package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	check "gopkg.in/check.v1"

	mock_crawler "github.com/mycok/uSearch/crawler/mocks"
	"github.com/mycok/uSearch/linkgraph/graph"
)

// Initialize and register a pointer instance of the graphUpdateTestSuite
// to be executed by check testing package.
var _ = check.Suite(new(graphUpdateTestSuite))

type graphUpdateTestSuite struct {
	graph *mock_crawler.MockMiniGraph
}

func (s *graphUpdateTestSuite) TestSuccessfulGraphUpdate(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.graph = mock_crawler.NewMockMiniGraph(ctrl)

	payload := &crawlerPayload{
		LinkID: uuid.New(),
		URL:    "http://example.com",
		NoFollowLinks: []string{
			"http://exampletest.com",
		},
		Links: []string{
			"http://examplelinks.com",
			"http://examplelinks1.com",
		},
	}

	// We expect the original link to be upsert with a new timestamp and three
	// additional upsert calls for the discovered links.
	expect := s.graph.EXPECT()
	expect.UpsertLink(linkMatcher{
		id:        payload.LinkID,
		url:       payload.URL,
		notBefore: time.Now(),
	}).Return(nil)

	id0, id1, id2 := uuid.New(), uuid.New(), uuid.New()
	expect.UpsertLink(linkMatcher{
		url:       "http://exampletest.com",
		notBefore: time.Time{},
	}).DoAndReturn(setLinkID(id0))

	expect.UpsertLink(linkMatcher{
		url:       "http://examplelinks.com",
		notBefore: time.Time{},
	}).DoAndReturn(setLinkID(id1))

	expect.UpsertLink(linkMatcher{
		url:       "http://examplelinks1.com",
		notBefore: time.Time{},
	}).DoAndReturn(setLinkID(id2))

	// We then expect two edges to be created from the original link to the
	// two links we just created.
	expect.UpsertEdge(edgeMatcher{src: payload.LinkID, dest: id1}).Return(nil)
	expect.UpsertEdge(edgeMatcher{src: payload.LinkID, dest: id2}).Return(nil)

	// Finally we expect a call to drop all stale edges originating from the
	// original link.
	expect.RemoveStaleEdges(payload.LinkID, gomock.Any()).Return(nil)

	p := s.updateGraph(c, payload)
	c.Assert(p, check.Not(check.IsNil))
}

func (s *graphUpdateTestSuite) updateGraph(
	c *check.C, p *crawlerPayload,
) *crawlerPayload {

	output, err := newGraphUpdater(s.graph).Process(context.TODO(), p)
	c.Assert(err, check.IsNil)

	if output != nil {
		c.Assert(output, check.FitsTypeOf, p)

		return output.(*crawlerPayload)
	}

	return nil
}

func setLinkID(id uuid.UUID) func(*graph.Link) error {
	return func(l *graph.Link) error {
		l.ID = id

		return nil
	}
}

// linkMatcher implements gomock.Matcher interface. it's used to compare whether
// the value(s) of the object passed to the mocked method during testing matches
// the value(s) of the object passed to the mocked method while executing the
// unit under test.
type linkMatcher struct {
	id        uuid.UUID
	url       string
	notBefore time.Time
}

func (m linkMatcher) Matches(x interface{}) bool {
	link := x.(*graph.Link)

	return m.id == link.ID && m.url == link.URL && !link.RetrievedAt.Before(m.notBefore)
}

func (m linkMatcher) String() string {
	return fmt.Sprintf("has ID=%q, URL=%q and LastAccessed not before %v", m.id, m.url, m.notBefore)
}

// edgeMatcher implements gomock.Matcher interface. it's used to compare whether
// the value(s) of the object passed to the mocked method during testing matches
// the value(s) of the object passed to the mocked method while executing the
// unit under test.
type edgeMatcher struct {
	src  uuid.UUID
	dest uuid.UUID
}

func (m edgeMatcher) Matches(x interface{}) bool {
	edge := x.(*graph.Edge)
	return m.src == edge.Src && m.dest == edge.Dest
}

func (m edgeMatcher) String() string {
	return fmt.Sprintf("has src=%q and dest=%q", m.src, m.dest)
}
