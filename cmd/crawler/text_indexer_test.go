package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/mycok/uSearch/cmd/crawler/mocks"
	check "gopkg.in/check.v1"
)

// Initialize and register an instance of TextExtractorTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(TextIndexerTestSuite))

type TextIndexerTestSuite struct {
	indexer *mocks.MockMiniIndexer
}

func (s *TextIndexerTestSuite) TestTextIndexer(c *check.C) {
	ctrl := gomock.NewController(c)
	s.indexer = mocks.NewMockMiniIndexer(ctrl)

	payload := &crawlerPayload{
		LinkID:      uuid.New(),
		URL:         "http://example.com",
		Title:       "test title",
		TextContent: "Lorem ipsum rolor",
	}

	s.indexer.EXPECT().Index(
		docMatcher{
			linkID:    payload.LinkID,
			url:       payload.URL,
			title:     payload.Title,
			content:   payload.TextContent,
			notBefore: time.Now(),
		},
	).Return(nil)

	p := s.updateIndex(c, payload)
	c.Assert(p, check.NotNil)
}

func (s *TextIndexerTestSuite) updateIndex(c *check.C, p *crawlerPayload) *crawlerPayload {
	result, err := newTextIndexer(s.indexer).Process(context.TODO(), p)
	c.Assert(err, check.IsNil)

	if result != nil {
		c.Assert(result, check.FitsTypeOf, p)

		return result.(*crawlerPayload)
	}

	return nil
}

type docMatcher struct {
	linkID    uuid.UUID
	url       string
	title     string
	content   string
	notBefore time.Time
}

func (dm docMatcher) Matches(x interface{}) bool {
	doc := x.(*index.Document)
	return dm.linkID == doc.LinKID &&
		dm.url == doc.URL &&
		dm.title == doc.Title &&
		dm.content == doc.Content &&
		!doc.IndexedAt.Before(dm.notBefore)
}

func (dm docMatcher) String() string {
	return fmt.Sprintf("has LinkID=%q, URL=%q, Title=%q, Content=%q and IndexedAt not before %v", dm.linkID, dm.url, dm.title, dm.content, dm.notBefore)
}
