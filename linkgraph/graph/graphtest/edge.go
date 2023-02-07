package graphtest

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph"
)

// TestEdgeUpsert verifies the edge upsert logic.
func (s *BaseSuite) TestEdgeUpsert(c *check.C) {
	linkIDs := make([]uuid.UUID, 3)
	for i := 0; i < len(linkIDs); i++ {
		l := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(l), check.IsNil)

		linkIDs[i] = l.ID
	}

	e := &graph.Edge{
		Src:  linkIDs[0],
		Dest: linkIDs[1],
	}

	err := s.g.UpsertEdge(e)
	c.Assert(err, check.IsNil)
	c.Assert(e.ID, check.Not(check.Equals), uuid.Nil, check.Commentf(
		"expected an ID to be assigned to the new edge",
	))
	c.Assert(e.UpdatedAt.IsZero(), check.Equals, false, check.Commentf(
		"UpdatedAt field not set",
	))

	// Update existing edge
	forUpdate := &graph.Edge{
		ID:   e.ID,
		Src:  linkIDs[0],
		Dest: linkIDs[1],
	}
	err = s.g.UpsertEdge(forUpdate)
	c.Assert(err, check.IsNil)
	c.Assert(forUpdate.ID, check.Equals, e.ID, check.Commentf("edge ID changed while upserting"))
	c.Assert(forUpdate.UpdatedAt, check.Not(check.Equals), e.UpdatedAt, check.Commentf("UpdatedAt field not modified"))

	// Create edge with unknown link IDs
	invalid := &graph.Edge{
		Src:  linkIDs[0],
		Dest: uuid.New(),
	}
	err = s.g.UpsertEdge(invalid)
	c.Assert(errors.Is(err, graph.ErrUnknownEdgeLinks), check.Equals, true)
}

// TestConcurrentEdgeIterators ensures that multiple clients can concurrently
// access the store without causing data races.
func (s *BaseSuite) TestConcurrentEdgeIterators(c *check.C) {
	var (
		wg           sync.WaitGroup
		numIterators = 10
		numEdges     = 100
		linkIDs      = make([]uuid.UUID, numEdges*2)
	)

	// Upsert links
	for i := 0; i < numEdges*2; i++ {
		l := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(l), check.IsNil)

		linkIDs[i] = l.ID
	}

	// Upsert Edges
	for i := 0; i < numEdges; i++ {
		e := &graph.Edge{
			Src:  linkIDs[0],
			Dest: linkIDs[i],
		}

		c.Assert(s.g.UpsertEdge(e), check.IsNil)
	}

	wg.Add(numIterators)

	for i := 0; i < numIterators; i++ {
		go func(id int) {
			defer wg.Done()

			comment := check.Commentf("iterator %d", id)
			seen := make(map[string]bool)

			it, err := s.partitionedEdgeIterator(c, 0, 1, time.Now())
			c.Assert(err, check.IsNil)

			defer func() {
				c.Assert(it.Close(), check.IsNil, comment)
			}()

			for it.Next() {
				e := it.Edge()
				id := e.ID.String()
				c.Assert(seen[id], check.Equals, false, check.Commentf(
					"Iterator %d iterated the same edge twice", id,
				))
				seen[id] = true
			}

			c.Assert(seen, check.HasLen, numEdges, comment)
			c.Assert(it.Error(), check.IsNil, comment)
		}(i)
	}

	doneCh := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh: // Test completed successfully
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out while waiting for the tests to complete")
	}
}

// TestEdgeIteratorTimeFilter ensures that the time-based filtering of the
// edge iterator works as expected.
func (s *BaseSuite) TestEdgeIteratorTimeFilter(c *check.C) {
	linkIDs := make([]uuid.UUID, 3)
	linkInsertTimes := make([]time.Time, len(linkIDs))

	for i := 0; i < len(linkIDs); i++ {
		l := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(l), check.IsNil)

		linkIDs[i] = l.ID
		linkInsertTimes[i] = time.Now()
	}

	edgeIDs := make([]uuid.UUID, len(linkIDs))
	edgeInsertTimes := make([]time.Time, len(linkIDs))

	for i := 0; i < len(linkIDs); i++ {
		e := &graph.Edge{
			Src:  linkIDs[0],
			Dest: linkIDs[i],
		}

		c.Assert(s.g.UpsertEdge(e), check.IsNil)

		edgeIDs[i] = e.ID
		edgeInsertTimes[i] = time.Now()
	}

	for i, t := range edgeInsertTimes {
		c.Logf("Fetching edges created before edge %d", i)
		s.assertIteratedEdgeIDsMatch(c, t, edgeIDs[:i+1])
	}
}

// TestPartitionedEdgeIterators verifies that the graph partitioning logic
// works as expected even when partitions contain an uneven number of items.
func (s *BaseSuite) TestPartitionedEdgeIterators(c *check.C) {
	numEdges := 100
	numPartitions := 10
	linkIDs := make([]uuid.UUID, numEdges*2)

	for i := 0; i < numEdges*2; i++ {
		l := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(l), check.IsNil)
		linkIDs[i] = l.ID
	}

	for i := 0; i < numEdges; i++ {
		e := &graph.Edge{
			Src:  linkIDs[0],
			Dest: linkIDs[i],
		}
		c.Assert(s.g.UpsertEdge(e), check.IsNil)
	}

	// Check with both odd and even partition counts to check for rounding-related bugs.
	c.Assert(s.iteratePartitionedEdges(c, numPartitions), check.Equals, numEdges)
	c.Assert(s.iteratePartitionedEdges(c, numPartitions+1), check.Equals, numEdges)
}

// TestRemoveStaleEdges verifies that the edge deletion logic works as expected.
func (s *BaseSuite) TestRemoveStaleEdges(c *check.C) {
	numEdges := 100
	linkIDs := make([]uuid.UUID, numEdges*4)
	goneIDs := make(map[uuid.UUID]struct{})

	for i := 0; i < numEdges*4; i++ {
		l := &graph.Link{URL: fmt.Sprint(i)}
		c.Assert(s.g.UpsertLink(l), check.IsNil)

		linkIDs[i] = l.ID
	}

	var lastTimeStamp time.Time

	for i := 0; i < numEdges; i++ {
		e1 := &graph.Edge{
			Src:  linkIDs[0],
			Dest: linkIDs[i],
		}
		c.Assert(s.g.UpsertEdge(e1), check.IsNil)

		goneIDs[e1.ID] = struct{}{}
		lastTimeStamp = e1.UpdatedAt
	}

	deleteBefore := lastTimeStamp.Add(time.Millisecond)
	time.Sleep(250 * time.Millisecond)

	// The following edges will have an updatedAt value greater than lastTimeStamp
	for i := 0; i < numEdges; i++ {
		e2 := &graph.Edge{
			Src:  linkIDs[0],
			Dest: linkIDs[numEdges+i+1],
		}
		c.Assert(s.g.UpsertEdge(e2), check.IsNil)
	}

	c.Assert(s.g.RemoveStaleEdges(linkIDs[0], deleteBefore), check.IsNil)

	it, err := s.partitionedEdgeIterator(c, 0, 1, time.Now())
	c.Assert(err, check.IsNil)

	defer func() {
		c.Assert(it.Close(), check.IsNil)
	}()

	var seen int
	for it.Next() {
		id := it.Edge().ID
		_, found := goneIDs[id]
		c.Assert(found, check.Equals, false, check.Commentf("expected edge %s to be removed from the edge list", id.String()))
		seen++
	}

	c.Assert(seen, check.Equals, numEdges)
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

		defer func() {
			c.Assert(linkIt.Close(), check.IsNil)
		}()

		it, err := s.partitionedEdgeIterator(c, partition, numPartitions, time.Now())
		c.Assert(err, check.IsNil)

		defer func() {
			c.Assert(it.Close(), check.IsNil)
		}()

		for it.Next() {
			e := it.Edge()
			id := e.ID.String()
			c.Assert(seen[id], check.Equals, false, check.Commentf("Iterator returned same edge in different partitions"))
			seen[id] = true

			_, srcInPartition := linksInPartition[e.Src]
			c.Assert(srcInPartition, check.Equals, true, check.Commentf("Iterator returned an edge whose source link belongs to a different partition"))
		}

		c.Assert(it.Error(), check.IsNil)
		c.Assert(it.Close(), check.IsNil)
	}

	return len(seen)
}

func (s *BaseSuite) assertIteratedEdgeIDsMatch(c *check.C, updatedBefore time.Time, expected []uuid.UUID) {
	it, err := s.partitionedEdgeIterator(c, 0, 1, updatedBefore)
	c.Assert(err, check.IsNil)

	var got []uuid.UUID

	for it.Next() {
		got = append(got, it.Edge().ID)
	}
	c.Assert(it.Error(), check.IsNil)
	c.Assert(it.Close(), check.IsNil)

	sort.Slice(got, func(l, r int) bool { return got[l].String() < got[r].String() })
	sort.Slice(expected, func(l, r int) bool { return expected[l].String() < expected[r].String() })
	c.Assert(got, check.DeepEquals, expected)
}

func (s *BaseSuite) partitionedEdgeIterator(c *check.C, partition, numPartitions int, updatedBefore time.Time) (graph.EdgeIterator, error) {
	from, to := s.partitionRange(c, partition, numPartitions)
	return s.g.Edges(from, to, updatedBefore)
}
