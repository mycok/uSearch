package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/mycok/uSearch/cmd/crawler/mocks"
	"github.com/mycok/uSearch/internal/graphlink/graph"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the GraphUpdaterTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(GraphUpdaterTestSuite))

type GraphUpdaterTestSuite struct {
	graph *mocks.MockMiniGraph
}

func (s *GraphUpdaterTestSuite) SetUpSuite(c *check.C) {
	ctl := gomock.NewController(c)
	s.graph = mocks.NewMockMiniGraph(ctl)
}

func (s *GraphUpdaterTestSuite) TestGraphUpdater(c *check.C) {
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

	expected := s.graph.EXPECT()

	// We expect the original link to be upserted / updated with a new timestamp and
	// two additional insert calls for the discovered links.
	expected.UpsertLink(linkMatcher{
		id:        payload.LinkID,
		url:       payload.URL,
		notBefore: time.Now(),
	}).Return(nil)

	id0, id1, id2 := uuid.New(), uuid.New(), uuid.New()
	expected.UpsertLink(linkMatcher{
		url:       "http://exampletest.com",
		notBefore: time.Time{},
	}).DoAndReturn(setLinkID(id0))

	expected.UpsertLink(linkMatcher{
		url:       "http://examplelinks.com",
		notBefore: time.Time{},
	}).DoAndReturn(setLinkID(id1))

	expected.UpsertLink(linkMatcher{
		url:       "http://examplelinks1.com",
		notBefore: time.Time{},
	}).DoAndReturn(setLinkID(id2))

	// We then expect two edges to be created from the original link to the
	// two links we just created.
	expected.UpsertEdge(edgeMatcher{src: payload.LinkID, dest: id0}).Return(nil)
	expected.UpsertEdge(edgeMatcher{src: payload.LinkID, dest: id1}).Return(nil)

	// Finally we expect a call to drop stale edges whose source is the origin link.
	expected.RemoveStaleEdges(payload.LinkID, gomock.Any()).Return(nil)

	// test graphUpdater implementation by passing a mock updater object
	// that implements the MiniGraph interface.
	p := s.updateGraph(c, payload)
	c.Assert(p, check.NotNil)
}

func (s *GraphUpdaterTestSuite) updateGraph(c *check.C, p *crawlerPayload) *crawlerPayload {
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

type linkMatcher struct {
	id        uuid.UUID
	url       string
	notBefore time.Time
}

func (lm *linkMatcher) Matches(x interface{}) bool {
	link := x.(*graph.Link)

	return lm.id == link.ID && lm.url == link.URL && !link.RetrievedAt.Before(lm.notBefore)
}

func (lm linkMatcher) String() string {
	return fmt.Sprintf("has ID=%q, URL=%q and LastAccessed not before %v", lm.id, lm.url, lm.notBefore)
}

type edgeMatcher struct {
	src  uuid.UUID
	dest uuid.UUID
}

func (em *edgeMatcher) Matches(x interface{}) bool {
	edge := x.(*graph.Edge)
	return em.src == edge.Src && em.dest == edge.Dest
}

func (em *edgeMatcher) String() string {
	return fmt.Sprintf("has src=%q and dest=%q", em.src, em.dest)
}
