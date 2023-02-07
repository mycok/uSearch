package indextest

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/textindexer/index"
)

// BaseSuite defines a set of re-usable index related tests that can
// be executed against any concrete type that implements the index.Indexer interface.
type BaseSuite struct {
	idx index.Indexer
}

// SetIndex sets BaseSuite's index field.
func (s *BaseSuite) SetIndex(index index.Indexer) {
	s.idx = index
}

// TestIndexingDocument verifies the indexing logic for new and existing documents.
func (s *BaseSuite) TestIndexingDocument(c *check.C) {
	// Upsert new document.
	doc := &index.Document{
		LinkID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "test document title",
		Content:   "This should be the body text of the document",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Index insert++++: %v", err),
	)

	// Update existing document
	updatedDoc := &index.Document{
		LinkID:    doc.LinkID,
		URL:       doc.URL,
		Title:     "This is an updated document title",
		Content:   "This is an updated document body",
		IndexedAt: time.Now().UTC(),
	}

	err = s.idx.Index(updatedDoc)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Index update++++: %v", err),
	)

	// Query the index to verify the update process.
	d, err := s.idx.FindByID(updatedDoc.LinkID)
	c.Assert(err, check.IsNil)
	c.Assert(d, check.DeepEquals, updatedDoc)

	// Insert a document without an ID
	docWithoutID := &index.Document{
		URL: "https://example.com",
	}

	err = s.idx.Index(docWithoutID)
	c.Assert(
		errors.Is(err, index.ErrMissingLinkID), check.Equals, true,
		check.Commentf("++++Index insert++++: %v", err),
	)
}

// TestIndexingDoesNotOverridePageRank verifies the indexing logic for new documents and
// UpdateScore logic for existing documents.
func (s *BaseSuite) TestIndexingDoesNotOverridePageRank(c *check.C) {
	// Upsert new document.
	doc := &index.Document{
		LinkID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "test document title",
		Content:   "This should be the body text of the document",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Index insert++++: %v", err),
	)

	// Update the inserted doc's score.
	updatedPageRank := 0.5

	// Update existing document.
	updatedDoc := &index.Document{
		LinkID:    doc.LinkID,
		URL:       doc.URL,
		Title:     "This is an updated document title",
		Content:   "This is an updated document body",
		IndexedAt: time.Now().UTC(),
		PageRank:  updatedPageRank,
	}

	err = s.idx.Index(updatedDoc)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Index update++++: %v", err),
	)

	// Verify that the indexing logic doesn't override the PageRank value
	// during an update to an existing document.
	d, err := s.idx.FindByID(updatedDoc.LinkID)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Find doc by id++++: %v", err),
	)
	c.Assert(d.PageRank, check.Not(check.Equals), updatedPageRank)
	c.Assert(d.Title, check.Not(check.Equals), doc.Title)
	c.Assert(d.Content, check.Not(check.Equals), doc.Content)

	// Update the document page rank.
	err = s.idx.UpdateScore(doc.LinkID, updatedPageRank)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Update score++++: %v", err),
	)

	d, err = s.idx.FindByID(doc.LinkID)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Find doc by id++++: %v", err),
	)
	c.Assert(d.PageRank, check.Equals, updatedPageRank)
}

// TestFindByID verifies the document lookup logic.
func (s *BaseSuite) TestFindByID(c *check.C) {
	// Upsert new document.
	doc := &index.Document{
		LinkID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "test document title",
		Content:   "This should be the body text of the document",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Index insert++++: %v", err),
	)

	// Perform a doc lookup to verify the insert logic.
	retrievedDoc, err := s.idx.FindByID(doc.LinkID)
	c.Assert(err, check.IsNil)
	c.Assert(retrievedDoc, check.DeepEquals, doc, check.Commentf("document returned by FindByID does not match the inserted document"))

	// Perform a doc lookup for a non existing id.
	_, err = s.idx.FindByID(uuid.New())
	c.Assert(errors.Is(err, index.ErrNotFound), check.Equals, true)
}

// TestFullTextSearch verifies the document search logic when searching for
// exact phrases.
func (s *BaseSuite) TestFullTextSearch(c *check.C) {
	var (
		numOfDocs   = 50
		expectedIDs = make([]uuid.UUID, 0)
	)

	// Insert and assign scores / page ranks to 50 documents.
	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		doc := &index.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		if i%5 == 0 {
			doc.Content = "Updated Document Body"
			expectedIDs = append(expectedIDs, id)
		}

		err := s.idx.Index(doc)
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Index insert++++: %v", err),
		)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Update score++++: %v", err),
		)
	}

	// Perform phrase / full-text-search
	it, err := s.idx.Search(index.Query{
		Type:       index.QueryTypePhrase,
		Expression: "Updated Document Body",
	})
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Search full-text / phrase++++: %v", err),
	)
	c.Assert(iterateDocs(c, it), check.DeepEquals, expectedIDs)
}

// TestMatchKeywordSearch verifies the document search logic when searching for
// keyword matches.
func (s *BaseSuite) TestMatchKeywordSearch(c *check.C) {
	var (
		numOfDocs   = 50
		expectedIDs = make([]uuid.UUID, 0)
	)

	// Insert and assign scores / page ranks to 50 documents.
	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		doc := &index.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		if i%5 == 0 {
			doc.Content = "Updated Document Body"
			expectedIDs = append(expectedIDs, id)
		}

		err := s.idx.Index(doc)
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Index insert++++: %v", err),
		)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Update score++++: %v", err),
		)
	}

	// Perform phrase / full-text-search
	it, err := s.idx.Search(index.Query{
		Type:       index.QueryTypeMatch,
		Expression: "updated",
	})
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Search full-text / phrase++++: %v", err),
	)
	c.Assert(iterateDocs(c, it), check.DeepEquals, expectedIDs)
}

// TestMatchKeywordSearchWithOffset verifies the document search logic when searching
// for keyword matches and skipping some results.
func (s *BaseSuite) TestMatchKeywordSearchWithOffset(c *check.C) {
	var (
		numOfDocs   = 50
		expectedIDs = make([]uuid.UUID, 0)
	)

	// Insert and assign scores / page ranks to 50 documents.
	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		expectedIDs = append(expectedIDs, id)

		doc := &index.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		err := s.idx.Index(doc)
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Index insert++++: %v", err),
		)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Update score++++: %v", err),
		)
	}

	// Perform phrase / full-text-search
	it, err := s.idx.Search(index.Query{
		Type:       index.QueryTypeMatch,
		Expression: "body",
		Offset:     20,
	})
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Search full-text / phrase++++: %v", err),
	)
	c.Assert(iterateDocs(c, it), check.DeepEquals, expectedIDs[20:])

	// Search with offset above the total number of results
	it, err = s.idx.Search(index.Query{
		Type:       index.QueryTypeMatch,
		Expression: "body",
		Offset:     200,
	})

	c.Assert(err, check.IsNil)
	c.Assert(iterateDocs(c, it), check.HasLen, 0)
}

// TestUpdateScore checks that PageRank score updates work as expected.
func (s *BaseSuite) TestUpdateScore(c *check.C) {
	var (
		numOfDocs   = 50
		expectedIDs = make([]uuid.UUID, 0)
	)

	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		expectedIDs = append(expectedIDs, id)

		doc := &index.Document{
			LinkID:  id,
			Title:   fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		err := s.idx.Index(doc)
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Index insert++++: %v", err),
		)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Update score++++: %v", err),
		)
	}

	it, err := s.idx.Search(index.Query{
		Type:       index.QueryTypeMatch,
		Expression: "body",
	})
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Search full-text / phrase++++: %v", err),
	)
	c.Assert(iterateDocs(c, it), check.DeepEquals, expectedIDs)

	// Update the PageRank scores so that results are sorted in the
	// reverse order.
	for i := 0; i < numOfDocs; i++ {
		err := s.idx.UpdateScore(expectedIDs[i], float64(i))
		c.Assert(
			err, check.IsNil,
			check.Commentf("++++Update score++++: %v", err),
		)
	}

	it, err = s.idx.Search(index.Query{
		Type:       index.QueryTypeMatch,
		Expression: "body",
	})
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Search full-text / phrase++++: %v", err),
	)
	c.Assert(iterateDocs(c, it), check.DeepEquals, reverse(expectedIDs))
}

// TestUpdateScoreForUnknownDocument checks that a placeholder document will
// be created when setting the PageRank score for an unknown document.
func (s *BaseSuite) TestUpdateScoreForUnknownDocument(c *check.C) {
	linkID := uuid.New()

	err := s.idx.UpdateScore(linkID, 0.5)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Update score++++: %v", err),
	)

	doc, err := s.idx.FindByID(linkID)
	c.Assert(
		err, check.IsNil,
		check.Commentf("++++Find doc by id++++: %v", err),
	)
	c.Assert(doc.URL, check.Equals, "")
	c.Assert(doc.Title, check.Equals, "")
	c.Assert(doc.Content, check.Equals, "")
	c.Assert(doc.IndexedAt.IsZero(), check.Equals, true)
	c.Assert(doc.PageRank, check.Equals, 0.5)
}

func iterateDocs(c *check.C, it index.Iterator) []uuid.UUID {
	var docIDs []uuid.UUID
	for it.Next() {
		docIDs = append(docIDs, it.Document().LinkID)
	}

	c.Assert(it.Error(), check.IsNil)
	c.Assert(it.Close(), check.IsNil)

	return docIDs
}

func reverse(data []uuid.UUID) []uuid.UUID {
	for left, right := 0, len(data)-1; left < right; left, right = left+1, right-1 {
		data[left], data[right] = data[right], data[left]
	}

	return data
}
