package graphtest

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph"
)

// TestLinkUpsert verifies the link upsert logic.
func (s *BaseSuite) TestLinkUpsert(c *check.C) {
	// Create a new link
	initial := &graph.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Add(-10 * time.Hour),
	}

	err := s.g.UpsertLink(initial)

	c.Assert(err, check.IsNil)
	// Expect a new ID to be assigned to the new Link.
	c.Assert(initial.ID, check.Not(check.Equals), uuid.Nil,
		check.Commentf("Expected an ID to be assigned to the new link."),
	)

	l, err := s.g.FindLink(initial.ID)

	c.Assert(err, check.IsNil)
	// Assert that the link was successfully created and stored.
	c.Assert(
		l.ID, check.Equals, initial.ID,
		check.Commentf("New link was never created and stored"),
	)

	// Attempt to upsert a link with same ID and URL as an existing link but
	// with a new RetrievedAt timestamp. This should update the existing link
	// with a new RetrievedAt timestamp.
	accessedAt := time.Now().Truncate(time.Second).UTC()
	updated := &graph.Link{
		ID:          initial.ID,
		URL:         initial.URL,
		RetrievedAt: accessedAt,
	}

	err = s.g.UpsertLink(updated)

	c.Assert(err, check.IsNil)
	// Assert that the updated link has not been added as a new link, but
	// instead updated. [ID still points to the same link].
	c.Assert(
		updated.ID, check.Equals, initial.ID,
		check.Commentf("ID changed during upsert"),
	)

	l, err = s.g.FindLink(updated.ID)

	c.Assert(err, check.IsNil)
	// Assert that the link's RetrievedAt field was updated.
	c.Assert(
		l.RetrievedAt, check.Equals, accessedAt,
		check.Commentf("RetrievedAt timestamp was never updated during upsert"),
	)

	// Attempt to insert a link whose URL matches an existing Link,
	// but with an older RetrievedAt value. The update to the RetrievedAt
	// field should not happen.
	oldRetrievedAt := time.Now().Add(-10 * time.Hour).UTC()
	sameURL := &graph.Link{
		URL:         updated.URL,
		RetrievedAt: oldRetrievedAt,
	}

	err = s.g.UpsertLink(sameURL)
	c.Assert(err, check.IsNil)
	c.Assert(sameURL.ID, check.Equals, updated.ID)

	l, err = s.g.FindLink(updated.ID)
	c.Assert(err, check.IsNil)
	// Assert that the RetrievedAt field was not updated during the upsert.
	c.Assert(l.RetrievedAt, check.Equals, accessedAt)
	c.Assert(l.RetrievedAt, check.Not(check.Equals), oldRetrievedAt)
}

// TestFindLink verifies the link lookup logic.
func (s *BaseSuite) TestFindLink(c *check.C) {
	// Create a new link
	newLink := &graph.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Truncate(time.Second),
	}

	err := s.g.UpsertLink(newLink)
	c.Assert(err, check.IsNil)
	// Expect a new ID to be assigned to the new Link.
	c.Assert(newLink.ID, check.Not(check.Equals), uuid.Nil,
		check.Commentf("Expected an ID to be assigned to the new link."),
	)

	// Lookup link by ID.
	l, err := s.g.FindLink(newLink.ID)
	c.Assert(err, check.IsNil)
	c.Assert(
		l, check.DeepEquals, newLink,
		check.Commentf("Lookup by ID returned wrong link"),
	)

	// Lookup link by unknown ID.
	_, err = s.g.FindLink(uuid.Nil)
	c.Assert(errors.Is(err, graph.ErrNotFound), check.Equals, true)
}

// TestConcurrentLinkIterators ensures that multiple clients can concurrently
// access the store without causing data races.
func (s *BaseSuite) TestConcurrentLinkIterators(c *check.C) {
	var (
		wg             sync.WaitGroup
		numOfIterators = 10
		numOfLinks     = 100
	)

	// Upsert 100 links into the graph store.
	for i := 0; i < numOfLinks; i++ {
		l := &graph.Link{URL: fmt.Sprint(i)}
		err := s.g.UpsertLink(l)
		c.Assert(err, check.IsNil)
	}

	wg.Add(numOfIterators)

	for i := 0; i < numOfIterators; i++ {
		go func(id int) {
			defer wg.Done()

			errComment := check.Commentf("Iterator %d", id)

			iterated := make(map[string]bool)

			it, err := s.partitionedLinkIterator(c, 0, 1, time.Now())
			c.Assert(err, check.IsNil)

			defer func() {
				c.Assert(it.Close(), check.IsNil, errComment)
			}()

			// Iterate over links for a specific partition.
			for it.Next() {
				link := it.Link()
				linkID := link.ID.String()

				c.Assert(
					iterated[linkID], check.Equals, false,
					check.Commentf("Iterator %d iterated the same link twice", id),
				)

				iterated[linkID] = true
			}

			c.Assert(iterated, check.HasLen, numOfLinks, errComment)
			c.Assert(it.Error(), check.IsNil, errComment)
			c.Assert(it.Close(), check.IsNil, errComment)
		}(i)
	}

	doneCh := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh: // Test completed successfully.
	case <-time.After(10 * time.Second):
		c.Fatal("Exceeded set test execution time: timed out!")
	}
}

// TestLinkIteratorTimeFilter ensures that the time-based filtering of the
// link iterator works as expected.
func (s *BaseSuite) TestLinkIteratorTimeFilter(c *check.C) {
	linkIDs := make([]uuid.UUID, 3)
	linkInsertTimes := make([]time.Time, len(linkIDs))

	for i := 0; i < len(linkIDs); i++ {
		l := &graph.Link{
			URL:         fmt.Sprint(i),
			RetrievedAt: time.Now(),
		}
		c.Assert(s.g.UpsertLink(l), check.IsNil)

		linkIDs[i] = l.ID
		linkInsertTimes[i] = time.Now()
	}

	for i, t := range linkInsertTimes {
		c.Logf("Fetching links created before link %d", i)
		s.assertIteratedLinkIDsMatch(c, t, linkIDs[:i+1])
	}
}

// TestPartitionedLinkIterators ensures that the graph partitioning logic
// works as expected even when partitions contain an uneven number of items.
func (s *BaseSuite) TestPartitionedLinkIterators(c *check.C) {
	numLinks := 100
	numPartitions := 10

	// Upsert 100 links.
	for i := 0; i < numLinks; i++ {
		c.Assert(s.g.UpsertLink(&graph.Link{URL: fmt.Sprint(i)}), check.IsNil)
	}

	// Check with both odd and even partition counts to check for rounding-related bugs.
	c.Assert(s.iteratePartitionedLinks(c, numPartitions), check.Equals, numLinks)
	c.Assert(s.iteratePartitionedLinks(c, numPartitions+1), check.Equals, numLinks)
	c.Assert(s.iteratePartitionedLinks(c, numPartitions-8), check.Equals, numLinks)
	c.Assert(s.iteratePartitionedLinks(c, numPartitions+9), check.Equals, numLinks)
}

func (s *BaseSuite) iteratePartitionedLinks(c *check.C, numPartitions int) int {
	seen := make(map[string]bool)

	for partition := 0; partition < numPartitions; partition++ {
		it, err := s.partitionedLinkIterator(c, partition, numPartitions, time.Now())
		c.Assert(err, check.IsNil)

		defer func() {
			c.Assert(it.Close(), check.IsNil)
		}()

		for it.Next() {
			l := it.Link()
			linkID := l.ID.String()
			c.Assert(seen[linkID], check.Equals, false, check.Commentf(
				"Iterator returned same link in different partitions",
			))
			seen[linkID] = true
		}
	}

	return len(seen)
}

func (s *BaseSuite) assertIteratedLinkIDsMatch(
	c *check.C, retrievedBefore time.Time, expected []uuid.UUID) {

	it, err := s.partitionedLinkIterator(c, 0, 1, retrievedBefore)
	c.Assert(err, check.IsNil)

	var got []uuid.UUID

	for it.Next() {
		got = append(got, it.Link().ID)
	}

	c.Assert(it.Error(), check.IsNil)
	c.Assert(it.Close(), check.IsNil)

	sort.Slice(got, func(i, j int) bool {
		return got[i].String() < got[j].String()
	})

	sort.Slice(expected, func(i, j int) bool {
		return expected[i].String() < expected[j].String()
	})

	c.Assert(got, check.DeepEquals, expected)
}

func (s *BaseSuite) partitionedLinkIterator(
	c *check.C, partition, numPartitions int, accessedBefore time.Time) (graph.LinkIterator, error) {

	from, to := s.partitionRange(c, partition, numPartitions)
	return s.g.Links(from, to, accessedBefore)
}

// Compute the UUID range for which to query the links and or edges by providing
// the start partition value and the desired number of partitions.
func (s *BaseSuite) partitionRange(
	c *check.C, partition, numPartitions int) (from, to uuid.UUID) {
	if partition < 0 || partition >= numPartitions {
		c.Fatal("invalid partition")
	}
	// A 16-byte array filled with 0's
	var minUUID = uuid.Nil
	// Returns a 16-byte array as the value of the maxUUID
	var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	var err error

	// Calculate the size of each partition as: (2^128 / numPartitions)
	tokenRange := big.NewInt(0)
	partSize := big.NewInt(0)
	partSize.SetBytes(maxUUID[:])
	// Calculate the size of each partition.
	partSize = partSize.Div(partSize, big.NewInt(int64(numPartitions)))

	// We model the partitions as a segment that begins at minUUID (all
	// bits set to zero) and ends at maxUUID (all bits set to 1). By
	// setting the end range for the *last* partition to maxUUID we ensure
	// that we always cover the full range of UUIDs even if the range
	// itself is not evenly divisible by numPartitions.
	if partition == 0 {
		from = minUUID
	} else {
		tokenRange.Mul(partSize, big.NewInt(int64(partition)))
		from, err = uuid.FromBytes(tokenRange.Bytes())
		c.Assert(err, check.IsNil)
	}

	if partition == numPartitions-1 {
		to = maxUUID
	} else {
		tokenRange.Mul(partSize, big.NewInt(int64(partition+1)))
		to, err = uuid.FromBytes(tokenRange.Bytes())
		c.Assert(err, check.IsNil)
	}

	return from, to
}
