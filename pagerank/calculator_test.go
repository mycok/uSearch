package pagerank_test

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/pagerank"
)

var _ = check.Suite(new(CalculatorTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type edge struct {
	src, dest string
}

type spec struct {
	description string
	vertices    []string
	edges       []edge
	expScores   map[string]float64
}

type CalculatorTestSuite struct{}

func (s *CalculatorTestSuite) TestSimpleGraphCase1(c *check.C) {
	spec := spec{
		description: `
(A -> (B) -> (C)
 ^            |
 |            |
 +------------+
Expect the page rank score to be distributed evenly across the three nodes 
`,
		vertices: []string{"A", "B", "C"},
		edges: []edge{
			{src: "A", dest: "B"},
			{src: "B", dest: "C"},
			{src: "C", dest: "A"},
		},
		expScores: map[string]float64{
			"A": 1.0 / 3.0,
			"B": 1.0 / 3.0,
			"C": 1.0 / 3.0,
		},
	}

	s.assertOnPageRankScores(c, spec)
}

func (s *CalculatorTestSuite) TestSimpleGraphCase2(c *check.C) {
	spec := spec{
		description: `
  +--(A)<-+
  |       |
  V       |
 (B) <-> (C)

Expect B and C to get better score than A due to the back-link between them.
Also, B should get slightly better score than C as there are two links pointing
to it.
`,
		vertices: []string{"A", "B", "C"},
		edges: []edge{
			{"A", "B"},
			{"B", "C"},
			{"C", "A"},
			{"C", "B"},
		},
		expScores: map[string]float64{
			"A": 0.2145,
			"B": 0.3937,
			"C": 0.3879,
		},
	}

	s.assertOnPageRankScores(c, spec)
}

func (s *CalculatorTestSuite) TestSimpleGraphCase3(c *check.C) {
	spec := spec{
		description: `
 (A) <-> (B) <-> (C)

Expect A and C to get the same score and B to get the largest score since there 
are two links pointing to it.
`,
		vertices: []string{"A", "B", "C"},
		edges: []edge{
			{"A", "B"},
			{"B", "A"},
			{"B", "C"},
			{"C", "B"},
		},
		expScores: map[string]float64{
			"A": 0.2569,
			"B": 0.4860,
			"C": 0.2569,
		},
	}

	s.assertOnPageRankScores(c, spec)
}

func (s *CalculatorTestSuite) TestDeadEnd(c *check.C) {
	spec := spec{
		description: `
 (A) -> (B) -> (C)

Expect that S(C) < S(A) < S(B). C is a dead-end as it has no outgoing links.
The algorithm deals with such cases by transferring C's score to a random node
in the graph; essentially, it's like C is connected to all other nodes in the
graph. As a result, A and B get a backlink from C; B now has two links pointing
at it (from A and C's backlink) and hence has the biggest score. Due to the 
random teleportation from C, C will get a slightly lower score than A.
`,
		vertices: []string{"A", "B", "C"},
		edges: []edge{
			{"A", "B"},
			{"B", "C"},
		},
		expScores: map[string]float64{
			"A": 0.1842,
			"B": 0.3411,
			"C": 0.4745,
		},
	}

	s.assertOnPageRankScores(c, spec)
}

func (s *CalculatorTestSuite) TestConvergenceForLargeGraphs(c *check.C) {
	s.assertOnConvergence(c, 100000, 7)
}

func (s *CalculatorTestSuite) assertOnPageRankScores(c *check.C, spec spec) {
	c.Log(spec.description)

	// Ensure to use the same seed to make the test deterministic.
	rand.Seed(42)

	calc, err := pagerank.NewCalculator(pagerank.Config{
		ComputeWorkers: 2,
		DampingFactor:  0.85,
	})
	c.Assert(err, check.IsNil)
	defer func() {
		_ = calc.Close()
	}()

	// Add vertices to the graph.
	for _, id := range spec.vertices {
		calc.AddVertex(id)
	}

	// Add edges to the graph.
	for _, e := range spec.edges {
		c.Assert(calc.AddEdge(e.src, e.dest), check.IsNil)
	}

	err = calc.CalculatePageRanks(context.TODO())
	c.Assert(err, check.IsNil)
	c.Logf("****converged after %d steps****", calc.Graph().SuperStep())

	var pageRankSum float64
	err = calc.Scores(func(id string, score float64) error {
		pageRankSum += score
		absDelta := math.Abs(score - spec.expScores[id])

		c.Assert(
			absDelta <= 0.01, check.Equals, true,
			check.Commentf(
				"expected score for %v to be %f ± 0.01; got %f (abs. delta %f)",
				id, spec.expScores[id], score, absDelta,
			))

		return nil
	})
	c.Assert(err, check.IsNil)

	c.Assert(
		(1.0-pageRankSum) <= 0.001, check.Equals, true,
		check.Commentf(
			"expected all pagerank scores to add up to 1.0; got %f", pageRankSum,
		))
}

func (s *CalculatorTestSuite) assertOnConvergence(c *check.C, numOfLinks, maxOutLinks int) {
	calc, err := pagerank.NewCalculator(pagerank.Config{
		ComputeWorkers:       32,
		MinSADForConvergence: 0.001,
	})
	c.Assert(err, check.IsNil)
	defer func() {
		_ = calc.Close()
	}()

	// Ensure to use the same seed to make the test deterministic.
	rand.Seed(42)

	names := make([]string, numOfLinks)
	for i := 0; i < numOfLinks; i++ {
		names[i] = strconv.FormatInt(int64(i), 10)
	}

	start := time.Now()
	for i := 0; i < numOfLinks; i++ {
		calc.AddVertex(names[i])

		outLinks := rand.Intn(maxOutLinks)
		for j := 0; j < outLinks; j++ {
			dest := rand.Intn(numOfLinks)
			c.Assert(calc.AddEdge(names[i], names[dest]), check.IsNil)
		}
	}
	c.Logf(
		"constructed %d nodes in %v",
		numOfLinks, time.Since(start).Truncate(time.Millisecond).String(),
	)

	start = time.Now()
	err = calc.CalculatePageRanks(context.TODO())
	c.Assert(err, check.IsNil)
	c.Logf(
		"converged %d nodes after %d steps in %v",
		numOfLinks, calc.Graph().SuperStep,
		time.Since(start).Truncate(time.Millisecond).String(),
	)

	var pageRankSum float64
	err = calc.Scores(func(id string, score float64) error {
		pageRankSum += score

		return nil
	})
	c.Assert(err, check.IsNil)

	c.Assert(
		(1.0-pageRankSum) <= 0.001, check.Equals, true,
		check.Commentf("expected all pagerank scores to add up to 1.0; got %f", pageRankSum),
	)
}
