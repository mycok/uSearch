/*
Package graphtests contains re-usable test suites that can be,
imported and run against any object that implements the Graph
interface.
*/
package graphtests

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/mycok/uSearch/internal/graphlink/graph"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"
)

// BaseSuite defines a set of re-usable graph-related tests that can
// be executed against any concrete type that implements the graph.Graph interface.
type BaseSuite struct {
	g graph.Graph
}

// SetGraph configures the test-suite to run all tests against an instance
// of graph.Graph.
func (s *BaseSuite) SetGraph(g graph.Graph) {
	s.g = g
}

// TestUpsertLink verifies the link upsert logic.
func (s *BaseSuite) TestUpsertLink(c *check.C) {
	// Create a new link.
	newLink := &graph.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Add(-10 * time.Hour),
	}

	err := s.g.UpsertLink(newLink)
	c.Assert(err, check.IsNil)
	// Expected a new ID to be assigned to the new Link.
	c.Assert(newLink.ID, check.Not(check.Equals), uuid.Nil)

	// Update newly created link with a new RetrievedAt timestamp.
	accessedAt := time.Now().Truncate(time.Second).UTC()
	updatedLink := &graph.Link{
		ID:          newLink.ID,
		URL:         "https://example.com",
		RetrievedAt: accessedAt,
	}

	err = s.g.UpsertLink(updatedLink)
	c.Assert(err, check.IsNil)
	// Link ID changed as a result of a successful upsert / update.
	c.Assert(updatedLink.ID, check.Equals, newLink.ID)

	stored, err := s.g.FindLink(updatedLink.ID)
	c.Assert(err, check.IsNil)
	// RetrievedAt timestamp was not updated.
	c.Assert(stored.RetrievedAt, check.Equals, accessedAt)

	// Attempt to insert a new Link whose URL matches an existing Link
	// and provide an older accessedAt value.
	sameURL := &graph.Link{
		URL:         updatedLink.URL,
		RetrievedAt: time.Now().Add(-10 * time.Hour),
	}
	err = s.g.UpsertLink(sameURL)
	c.Assert(err, check.IsNil)
	c.Assert(sameURL.ID, check.Equals, updatedLink.ID)

	stored, err = s.g.FindLink(updatedLink.ID)
	c.Assert(err, check.IsNil)
	// RetrievedAt timestamp was over-written by an older value.
	c.Assert(stored.RetrievedAt, check.Equals, accessedAt)

	// Create a new link and then attempt to update its URL to the same as
	// an existing link.
	dup := &graph.Link{
		URL: "foo",
	}
	err = s.g.UpsertLink(dup)
	c.Assert(err, check.IsNil)
	// Expected a linkID to be assigned to the new link.
	c.Assert(dup.ID, check.Not(check.Equals), uuid.Nil)
}

// TestFindLink verifies the link lookup logic.
func (s *BaseSuite) TestFindLink(c *check.C) {
	// Create a new link
	link := &graph.Link{
		URL:         "https://example.com",
		RetrievedAt: time.Now().Truncate(time.Second).UTC(),
	}

	err := s.g.UpsertLink(link)
	c.Assert(err, check.IsNil)
	c.Assert(link.ID, check.Not(check.Equals), uuid.Nil, check.Commentf("expected a linkID to be assigned to the new link"))

	// Lookup link by ID
	other, err := s.g.FindLink(link.ID)
	c.Assert(err, check.IsNil)
	c.Assert(other, check.DeepEquals, link, check.Commentf("lookup by ID returned the wrong link"))

	// Lookup link by unknown ID
	_, err = s.g.FindLink(uuid.Nil)
	c.Assert(errors.Is(err, graph.ErrNotFound), check.Equals, true)
}

// TestConcurrentLinkIterators verifies that multiple clients can concurrently
// access the store.
func (s *BaseSuite) TestConcurrentLinkIterators(c *check.C) {
	var (
		wg           sync.WaitGroup
		numIterators = 10
		numLinks     = 100
	)

	for i := 0; i < numLinks; i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
	}

	wg.Add(numIterators)
	for i := 0; i < numIterators; i++ {
		go func(id int) {
			defer wg.Done()

			cc := check.Commentf("iterator %d", id)
			seen := make(map[string]bool)
			it, err := s.partitionedLinkIterator(c, 0, 1, time.Now())
			c.Assert(err, check.IsNil, cc)
			defer func() {
				c.Assert(it.Close(), check.IsNil, cc)
			}()

			for i := 0; it.Next(); i++ {
				link := it.Link()
				linkID := link.ID.String()
				c.Assert(seen[linkID], check.Equals, false, check.Commentf("iterator %d saw same link twice", id))
				seen[linkID] = true
			}

			c.Assert(seen, check.HasLen, numLinks, cc)
			c.Assert(it.Error(), check.IsNil, cc)
			c.Assert(it.Close(), check.IsNil, cc)
		}(i)
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	// test completed successfully
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for test to complete")
	}
}

// TestLinkIteratorTimeFilter verifies that the time-based filtering of the
// link iterator works as expected.
func (s *BaseSuite) TestLinkIteratorTimeFilter(c *check.C) {
	linkUUIDs := make([]uuid.UUID, 3)
	linkInsertTimes := make([]time.Time, len(linkUUIDs))
	for i := 0; i < len(linkUUIDs); i++ {
		link := &graph.Link{URL: fmt.Sprint(i), RetrievedAt: time.Now()}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		linkUUIDs[i] = link.ID
		linkInsertTimes[i] = time.Now()
	}

	for i, t := range linkInsertTimes {
		c.Logf("fetching links created before edge %d", i)
		s.assertIteratedLinkIDsMatch(c, t, linkUUIDs[:i+1])
	}
}

func (s *BaseSuite) assertIteratedLinkIDsMatch(c *check.C, updatedBefore time.Time, exp []uuid.UUID) {
	it, err := s.partitionedLinkIterator(c, 0, 1, updatedBefore)
	c.Assert(err, check.IsNil)

	var got []uuid.UUID
	for it.Next() {
		got = append(got, it.Link().ID)
	}
	c.Assert(it.Error(), check.IsNil)
	c.Assert(it.Close(), check.IsNil)

	sort.Slice(got, func(l, r int) bool { return got[l].String() < got[r].String() })
	sort.Slice(exp, func(l, r int) bool { return exp[l].String() < exp[r].String() })
	c.Assert(got, check.DeepEquals, exp)
}

// TestPartitionedLinkIterators verifies that the graph partitioning logic
// works as expected even when partitions contain an uneven number of items.
func (s *BaseSuite) TestPartitionedLinkIterators(c *check.C) {
	numLinks := 100
	numPartitions := 10
	for i := 0; i < numLinks; i++ {
		c.Assert(s.g.UpsertLink(&graph.Link{URL: fmt.Sprint(i)}), check.IsNil)
	}

	// Check with both odd and even partition counts to check for rounding-related bugs.
	c.Assert(s.iteratePartitionedLinks(c, numPartitions), check.Equals, numLinks)
	c.Assert(s.iteratePartitionedLinks(c, numPartitions+1), check.Equals, numLinks)
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
			link := it.Link()
			linkID := link.ID.String()
			c.Assert(seen[linkID], check.Equals, false, check.Commentf("iterator returned same link in different partitions"))
			seen[linkID] = true
		}

		c.Assert(it.Error(), check.IsNil)
		c.Assert(it.Close(), check.IsNil)
	}

	return len(seen)
}

// TestUpsertEdge verifies the edge upsert logic.
func (s *BaseSuite) TestUpsertEdge(c *check.C) {
	// Create links
	linkUUIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		linkUUIDs[i] = link.ID
	}

	// Create a edge
	edge := &graph.Edge{
		Src:  linkUUIDs[0],
		Dest: linkUUIDs[1],
	}

	err := s.g.UpsertEdge(edge)
	c.Assert(err, check.IsNil)
	c.Assert(edge.ID, check.Not(check.Equals), uuid.Nil, check.Commentf("expected an edgeID to be assigned to the new edge"))
	c.Assert(edge.UpdatedAt.IsZero(), check.Equals, false, check.Commentf("UpdatedAt field not set"))

	// Update existing edge
	other := &graph.Edge{
		ID:   edge.ID,
		Src:  linkUUIDs[0],
		Dest: linkUUIDs[1],
	}
	err = s.g.UpsertEdge(other)
	c.Assert(err, check.IsNil)
	c.Assert(other.ID, check.Equals, edge.ID, check.Commentf("edge ID changed while upserting"))
	c.Assert(other.UpdatedAt, check.Not(check.Equals), edge.UpdatedAt, check.Commentf("UpdatedAt field not modified"))

	// Create edge with unknown link IDs
	bogus := &graph.Edge{
		Src:  linkUUIDs[0],
		Dest: uuid.New(),
	}
	err = s.g.UpsertEdge(bogus)
	c.Assert(errors.Is(err, graph.ErrUnknownEdgeLinks), check.Equals, true)
}

// TestConcurrentEdgeIterators verifies that multiple clients can concurrently
// access the store.
func (s *BaseSuite) TestConcurrentEdgeIterators(c *check.C) {
	var (
		wg           sync.WaitGroup
		numIterators = 10
		numEdges     = 100
		linkUUIDs    = make([]uuid.UUID, numEdges*2)
	)

	for i := 0; i < numEdges*2; i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		linkUUIDs[i] = link.ID
	}
	for i := 0; i < numEdges; i++ {
		c.Assert(s.g.UpsertEdge(&graph.Edge{
			Src:  linkUUIDs[0],
			Dest: linkUUIDs[i],
		}), check.IsNil)
	}

	wg.Add(numIterators)
	for i := 0; i < numIterators; i++ {
		go func(id int) {
			defer wg.Done()

			cc := check.Commentf("iterator %d", id)
			seen := make(map[string]bool)
			it, err := s.partitionedEdgeIterator(c, 0, 1, time.Now())
			c.Assert(err, check.IsNil, cc)
			defer func() {
				c.Assert(it.Close(), check.IsNil, cc)
			}()

			for i := 0; it.Next(); i++ {
				edge := it.Edge()
				edgeID := edge.ID.String()
				c.Assert(seen[edgeID], check.Equals, false, check.Commentf("iterator %d saw same edge twice", id))
				seen[edgeID] = true
			}

			c.Assert(seen, check.HasLen, numEdges, cc)
			c.Assert(it.Error(), check.IsNil, cc)
			c.Assert(it.Close(), check.IsNil, cc)
		}(i)
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	// test completed successfully
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for test to complete")
	}
}

// TestEdgeIteratorTimeFilter verifies that the time-based filtering of the
// edge iterator works as expected.
func (s *BaseSuite) TestEdgeIteratorTimeFilter(c *check.C) {
	linkUUIDs := make([]uuid.UUID, 3)
	linkInsertTimes := make([]time.Time, len(linkUUIDs))
	for i := 0; i < len(linkUUIDs); i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		linkUUIDs[i] = link.ID
		linkInsertTimes[i] = time.Now()
	}

	edgeUUIDs := make([]uuid.UUID, len(linkUUIDs))
	edgeInsertTimes := make([]time.Time, len(linkUUIDs))
	for i := 0; i < len(linkUUIDs); i++ {
		edge := &graph.Edge{Src: linkUUIDs[0], Dest: linkUUIDs[i]}
		c.Assert(s.g.UpsertEdge(edge), check.IsNil)
		edgeUUIDs[i] = edge.ID
		edgeInsertTimes[i] = time.Now()
	}

	for i, t := range edgeInsertTimes {
		c.Logf("fetching edges created before edge %d", i)
		s.assertIteratedEdgeIDsMatch(c, t, edgeUUIDs[:i+1])
	}
}

func (s *BaseSuite) assertIteratedEdgeIDsMatch(c *check.C, updatedBefore time.Time, exp []uuid.UUID) {
	it, err := s.partitionedEdgeIterator(c, 0, 1, updatedBefore)
	c.Assert(err, check.IsNil)

	var got []uuid.UUID
	for it.Next() {
		got = append(got, it.Edge().ID)
	}
	c.Assert(it.Error(), check.IsNil)
	c.Assert(it.Close(), check.IsNil)

	sort.Slice(got, func(l, r int) bool { return got[l].String() < got[r].String() })
	sort.Slice(exp, func(l, r int) bool { return exp[l].String() < exp[r].String() })
	c.Assert(got, check.DeepEquals, exp)
}

// TestPartitionedEdgeIterators verifies that the graph partitioning logic
// works as expected even when partitions contain an uneven number of items.
func (s *BaseSuite) TestPartitionedEdgeIterators(c *check.C) {
	numEdges := 100
	numPartitions := 10
	linkUUIDs := make([]uuid.UUID, numEdges*2)
	for i := 0; i < numEdges*2; i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		linkUUIDs[i] = link.ID
	}
	for i := 0; i < numEdges; i++ {
		c.Assert(s.g.UpsertEdge(&graph.Edge{
			Src:  linkUUIDs[0],
			Dest: linkUUIDs[i],
		}), check.IsNil)
	}

	// Check with both odd and even partition counts to check for rounding-related bugs.
	c.Assert(s.iteratePartitionedEdges(c, numPartitions), check.Equals, numEdges)
	c.Assert(s.iteratePartitionedEdges(c, numPartitions+1), check.Equals, numEdges)
}

func (s *BaseSuite) iteratePartitionedEdges(c *check.C, numPartitions int) int {
	seen := make(map[string]bool)
	for partition := 0; partition < numPartitions; partition++ {
		// Build list of expected edges per partition. An edge belongs to a
		// partition if its origin link also belongs to the same partition.
		linksInPartition := make(map[uuid.UUID]struct{})
		linkIt, err := s.partitionedLinkIterator(c, partition, numPartitions, time.Now())
		c.Assert(err, check.IsNil)
		for linkIt.Next() {
			linkID := linkIt.Link().ID
			linksInPartition[linkID] = struct{}{}
		}

		it, err := s.partitionedEdgeIterator(c, partition, numPartitions, time.Now())
		c.Assert(err, check.IsNil)
		defer func() {
			c.Assert(it.Close(), check.IsNil)
		}()

		for it.Next() {
			edge := it.Edge()
			edgeID := edge.ID.String()
			c.Assert(seen[edgeID], check.Equals, false, check.Commentf("iterator returned same edge in different partitions"))
			seen[edgeID] = true

			_, srcInPartition := linksInPartition[edge.Src]
			c.Assert(srcInPartition, check.Equals, true, check.Commentf("iterator returned an edge whose source link belongs to a different partition"))
		}

		c.Assert(it.Error(), check.IsNil)
		c.Assert(it.Close(), check.IsNil)
	}

	return len(seen)
}

// TestRemoveStaleEdges verifies that the edge deletion logic works as expected.
func (s *BaseSuite) TestRemoveStaleEdges(c *check.C) {
	numEdges := 100
	linkUUIDs := make([]uuid.UUID, numEdges*4)
	goneUUIDs := make(map[uuid.UUID]struct{})
	for i := 0; i < numEdges*4; i++ {
		link := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(link), check.IsNil)
		linkUUIDs[i] = link.ID
	}

	var lastTs time.Time
	for i := 0; i < numEdges; i++ {
		e1 := &graph.Edge{
			Src:  linkUUIDs[0],
			Dest: linkUUIDs[i],
		}
		c.Assert(s.g.UpsertEdge(e1), check.IsNil)
		goneUUIDs[e1.ID] = struct{}{}
		lastTs = e1.UpdatedAt
	}

	deleteBefore := lastTs.Add(time.Millisecond)
	time.Sleep(250 * time.Millisecond)

	// The following edges will have an updated at value > lastTs
	for i := 0; i < numEdges; i++ {
		e2 := &graph.Edge{
			Src:  linkUUIDs[0],
			Dest: linkUUIDs[numEdges+i+1],
		}
		c.Assert(s.g.UpsertEdge(e2), check.IsNil)
	}
	c.Assert(s.g.RemoveStaleEdges(linkUUIDs[0], deleteBefore), check.IsNil)

	it, err := s.partitionedEdgeIterator(c, 0, 1, time.Now())
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(it.Close(), check.IsNil) }()

	var seen int
	for it.Next() {
		id := it.Edge().ID
		_, found := goneUUIDs[id]
		c.Assert(found, check.Equals, false, check.Commentf("expected edge %s to be removed from the edge list", id.String()))
		seen++
	}

	c.Assert(seen, check.Equals, numEdges)
}

func (s *BaseSuite) partitionedLinkIterator(c *check.C, partition, numPartitions int, accessedBefore time.Time) (graph.LinkIterator, error) {
	from, to := s.partitionRange(c, partition, numPartitions)
	return s.g.Links(from, to, accessedBefore)
}

func (s *BaseSuite) partitionedEdgeIterator(c *check.C, partition, numPartitions int, updatedBefore time.Time) (graph.EdgeIterator, error) {
	from, to := s.partitionRange(c, partition, numPartitions)
	return s.g.Edges(from, to, updatedBefore)
}

func (s *BaseSuite) partitionRange(c *check.C, partition, numPartitions int) (from, to uuid.UUID) {
	if partition < 0 || partition >= numPartitions {
		c.Fatal("invalid partition")
	}

	var minUUID = uuid.Nil
	var maxUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	var err error

	// Calculate the size of each partition as: (2^128 / numPartitions)
	tokenRange := big.NewInt(0)
	partSize := big.NewInt(0)
	partSize.SetBytes(maxUUID[:])
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
