package indextests

import (
	"errors"
	"fmt"
	"time"

	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"
)

// BaseSuite defines a set of re-usable indexer related tests that can
// be executed against any concrete type that implements the index.Indexer interface.
type BaseSuite struct {
	idx index.Indexer
}

// SetGraph configures the test-suite to run all tests against an instance
// of index.Indexer.
func (s *BaseSuite) SetIndexer(idx index.Indexer) {
	s.idx = idx
}

// TestIndexDocument verifies the indexing logic for new and existing documents.
func (s *BaseSuite)  TestIndexDocument(c *check.C) {
	// Insert a new document
	doc := &index.Document{
		LinKID: uuid.New(),
		URL: "https://example.com",
		Title: "test document title",
		Content: "This should be the body text of the document",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	c.Assert(err, check.IsNil)

	// Update existing document
	updatedDoc := &index.Document{
		LinKID: doc.LinKID,
		URL: "https://example.com",
		Title: "This is an updated document title",
		Content: "This is an updated document body",
		IndexedAt: time.Now().UTC(),
	}

	err = s.idx.Index(updatedDoc)
	c.Assert(err, check.IsNil)

	// Insert a document without an ID
	docWithoutID := &index.Document{
		URL: "https://example.com",
	}

	err = s.idx.Index(docWithoutID)
	c.Assert(errors.Is(err, index.ErrMissingLinkID), check.Equals, true)
}

// TestIndexDoesNotOverridePageRank verifies the indexing logic for new documents and
// UpdateScore logic for existing documents.
func (s *BaseSuite)  TestIndexDoesNotOverridePageRank(c *check.C){
		// Insert a new document.
		doc := &index.Document{
			LinKID: uuid.New(),
			URL: "https://example.com",
			Title: "test document title",
			Content: "This should be the body text of the document",
			IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
		}

		err := s.idx.Index(doc)
		c.Assert(err, check.IsNil)

		// Update the inserted doc's score.
		updatedScore := 0.5
		err = s.idx.UpdateScore(doc.LinKID, updatedScore)
		c.Assert(err, check.IsNil)

		// Update existing document.
		updatedDoc := &index.Document{
			LinKID: doc.LinKID,
			URL: "https://example.com",
			Title: "This is an updated document title",
			Content: "This is an updated document body",
			IndexedAt: time.Now().UTC(),
		}

		err = s.idx.Index(updatedDoc)
		c.Assert(err, check.IsNil)

		// Perform a doc lookup to verify the updates without changing the
		// assigned PageRank / score.
		retrievedDoc, err := s.idx.FindByID(doc.LinKID)
		c.Assert(err, check.IsNil)
		c.Assert(retrievedDoc.PageRank, check.Equals, updatedScore)
}

// TestFindByID verifies the document lookup logic.
func (s *BaseSuite) TestFindByID(c *check.C) {
	// Insert a new document.
	doc := &index.Document{
		LinKID: uuid.New(),
		URL: "https://example.com",
		Title: "test document title",
		Content: "This should be the body text of the document",
		IndexedAt: time.Now().Add(-12 * time.Hour).UTC(),
	}

	err := s.idx.Index(doc)
	c.Assert(err, check.IsNil)

	// Perform a doc lookup to verify the insert logic.
	retrievedDoc, err := s.idx.FindByID(doc.LinKID)
	c.Assert(err, check.IsNil)
	c.Assert(retrievedDoc, check.DeepEquals, doc, check.Commentf("document returned by FindByID does not match the inserted document"))

	// Perform a doc lookup for a non existing id.
	_, err = s.idx.FindByID(uuid.New())
	c.Assert(errors.Is(err, index.ErrNotFound), check.Equals, true)
}

// TestPhraseSearch verifies the document search logic when searching for
// exact phrases.
func (s *BaseSuite) TestPhraseSearch(c *check.C) {
	var (
		numOfDocs = 50
		expectedIDs []uuid.UUID
	)

	// Insert and assign scrores / page ranks to 50 documents.
	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		doc := &index.Document{
			LinKID: id,
			Title: fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		if i % 5 == 0 {
			doc.Content = "Updated Document Body"
			expectedIDs = append(expectedIDs, id)
		}

		err := s.idx.Index(doc)
		c.Assert(err, check.IsNil)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(err, check.IsNil)
	}

	it, err := s.idx.Search(index.Query{
		Type: index.QueryTypePhrase,
		Expression: "updated document body",
	})

	c.Assert(err, check.IsNil)
	c.Assert(IterateDocs(c, it), check.DeepEquals, expectedIDs)

}

// TestMatchSearch verifies the document search logic when searching for
// keyword matches.
func (s *BaseSuite) TestMatchSearch(c *check.C) {
	var (
		numOfDocs = 50
		expectedIDs []uuid.UUID
	)

	// Insert and assign scrores / page ranks to 50 documents.
	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		doc := &index.Document{
			LinKID: id,
			Title: fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		if i % 5 == 0 {
			doc.Content = "Updated Document Body"
			expectedIDs = append(expectedIDs, id)
		}

		err := s.idx.Index(doc)
		c.Assert(err, check.IsNil)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(err, check.IsNil)
	}

	it, err := s.idx.Search(index.Query{
		Type: index.QueryTypeMatch,
		Expression: "updated body",
	})

	c.Assert(err, check.IsNil)
	c.Assert(IterateDocs(c, it), check.DeepEquals, expectedIDs)
}

// TestMatchSearchWithOffset verifies the document search logic when searching
// for keyword matches and skipping some results.
func (s *BaseSuite) TestMatchSearchWithOffset(c *check.C) {
	var (
		numOfDocs = 50
		expectedIDs []uuid.UUID
	)

	// Insert and assign scrores / page ranks to 50 documents.
	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		expectedIDs = append(expectedIDs, id)

		doc := &index.Document{
			LinKID: id,
			Title: fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		err := s.idx.Index(doc)
		c.Assert(err, check.IsNil)

		err = s.idx.UpdateScore(id, float64(numOfDocs-i))
		c.Assert(err, check.IsNil)
	}

	it, err := s.idx.Search(index.Query{
		Type: index.QueryTypeMatch,
		Expression: "body",
		Offset: 20,
	})

	c.Assert(err, check.IsNil)
	c.Assert(IterateDocs(c, it), check.DeepEquals, expectedIDs[20:])

	// Search with offset above the total number of results
	it, err = s.idx.Search(index.Query{
		Type: index.QueryTypeMatch,
		Expression: "body",
		Offset: 200,
	})

	c.Assert(err, check.IsNil)
	c.Assert(IterateDocs(c, it), check.HasLen, 0)
}

// TestUpdateScore checks that PageRank score updates work as expected.
func (s *BaseSuite) TestUpdateScore(c *check.C) {
	var (
		numOfDocs = 50
		expectedIDs []uuid.UUID
	)

	for i := 0; i < numOfDocs; i++ {
		id := uuid.New()
		expectedIDs = append(expectedIDs, id)

		doc := &index.Document{
			LinKID: id,
			Title: fmt.Sprintf("doc with ID %s", id.String()),
			Content: "This should be the body text of the document",
		}

		err := s.idx.Index(doc)
		c.Assert(err, check.IsNil)

		err = s.idx.UpdateScore(id, float64(numOfDocs - i))
		c.Assert(err, check.IsNil)
	}

	it, err := s.idx.Search(index.Query{
		Type: index.QueryTypeMatch,
		Expression: "body",
	})

	c.Assert(err, check.IsNil)
	c.Assert(IterateDocs(c, it), check.DeepEquals, expectedIDs)

	// Update the pagerank scores so that results are sorted in the
	// reverse order.
	for i := 0; i < numOfDocs; i++ {
		err = s.idx.UpdateScore(expectedIDs[i], float64(i))
		c.Assert(err, check.IsNil, check.Commentf(expectedIDs[i].String()))
	}

	it, err = s.idx.Search(index.Query{
		Type: index.QueryTypeMatch,
		Expression: "body",
	})

	c.Assert(err, check.IsNil)
	c.Assert(IterateDocs(c, it), check.DeepEquals, reverse(expectedIDs))
}

// TestUpdateScoreForUnknownDocument checks that a placeholder document will
// be created when setting the PageRank score for an unknown document.
func (s *BaseSuite) TestUpdateScoreForUnknownDocument(c *check.C) {
	linkID := uuid.New()

	err := s.idx.UpdateScore(linkID, 0.5)
	c.Assert(err, check.IsNil)

	doc, err := s.idx.FindByID(linkID)
	c.Assert(err, check.IsNil)

	c.Assert(doc.URL, check.Equals, "")
	c.Assert(doc.Title, check.Equals, "")
	c.Assert(doc.Content, check.Equals, "")
	c.Assert(doc.IndexedAt.IsZero(), check.Equals, true)
	c.Assert(doc.PageRank, check.Equals, 0.5)
}

func IterateDocs(c *check.C, it index.Iterator) []uuid.UUID {
	var docIDs []uuid.UUID
	for it.Next() {
		docIDs = append(docIDs, it.Document().LinKID)
	}

	c.Assert(it.Error(), check.IsNil)
	c.Assert(it.Close(), check.IsNil)

	return docIDs
}

func reverse(in []uuid.UUID) []uuid.UUID {
	for left, right := 0, len(in) - 1; left < right; left, right = left + 1, right - 1 {
		in[left], in[right] = in[right], in[left]
	}

	return in
}