package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	check "gopkg.in/check.v1"

	mock_crawler "github.com/mycok/uSearch/crawler/mocks"
	"github.com/mycok/uSearch/textindexer/index"
)

// Initialize and register a pointer instance of the textIndexTestSuite
// to be executed by check testing package.
var _ = check.Suite(new(textIndexTestSuite))

type textIndexTestSuite struct {
	indexer *mock_crawler.MockMiniIndexer
}

func (s *textIndexTestSuite) TestSuccessfulTextIndex(c *check.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	s.indexer = mock_crawler.NewMockMiniIndexer(ctrl)

	payload := &crawlerPayload{
		LinkID:      uuid.New(),
		URL:         "http://example.com",
		Title:       "test title",
		TextContent: "Lorem ipsum rolor",
	}

	s.indexer.EXPECT().Index(docMatcher{
		linkID:    payload.LinkID,
		url:       payload.URL,
		title:     payload.Title,
		content:   payload.TextContent,
		notBefore: time.Now(),
	}).Return(nil)

	p := s.updateIndex(c, payload)
	c.Assert(p, check.Not(check.IsNil))
}

func (s *textIndexTestSuite) updateIndex(c *check.C, p *crawlerPayload) *crawlerPayload {
	result, err := newTextIndexer(s.indexer).Process(context.TODO(), p)
	c.Assert(err, check.IsNil)

	if result != nil {
		c.Assert(result, check.FitsTypeOf, p)

		return result.(*crawlerPayload)
	}

	return nil
}

// docMatcher implements gomock.Matcher interface. it's used to compare whether
// the value(s) of the object passed to the mocked method during testing matches
// the value(s) of the object passed to the mocked method while executing the
// unit under test.
type docMatcher struct {
	linkID    uuid.UUID
	url       string
	title     string
	content   string
	notBefore time.Time
}

func (m docMatcher) Matches(x interface{}) bool {
	doc := x.(*index.Document)
	return m.linkID == doc.LinkID &&
		m.url == doc.URL &&
		m.title == doc.Title &&
		m.content == doc.Content &&
		!doc.IndexedAt.Before(m.notBefore)
}

func (m docMatcher) String() string {
	return fmt.Sprintf("has LinkID=%q, URL=%q, Title=%q, Content=%q and IndexedAt not before %v", m.linkID, m.url, m.title, m.content, m.notBefore)
}
