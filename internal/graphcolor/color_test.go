package graphcolor_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/mycok/uSearch/internal/graphcolor"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(GraphColorTestSuite))

type GraphColorTestSuite struct {
	assigner *graphcolor.Assigner
}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *GraphColorTestSuite) SetUpTest(c *check.C) {
	assigner, err := graphcolor.NewColorAssigner(16)
	c.Assert(err, check.IsNil)

	s.assigner = assigner
}

func (s *GraphColorTestSuite) TearDownTest(c *check.C) {
	c.Assert(s.assigner.Close(), check.IsNil)
}

func (s *GraphColorTestSuite) TestUncoloredGraph(c *check.C) {
	// Ensure to use the same seed to make the test deterministic.
	rand.Seed(42)
	adjMap := map[string][]string{
		"0": {"1", "2"},
		"1": {"2", "3"},
		"2": {"3"},
		"3": {"4"},
	}

	outDeg := s.setupGraph(c, adjMap, nil)

	colorMap := make(map[string]int)
	numOfColors, err := s.assigner.AssignColors(context.TODO(), func(vertexID string, color int) {
		colorMap[vertexID] = color
	})
	c.Assert(err, check.IsNil)

	maxColors := outDeg + 1
	c.Assert(numOfColors <= maxColors, check.Equals, true, check.Commentf("number of colors should not exceed (max vertex out degree + 1"))
	assertNoColorConflictWithNeibours(c, adjMap, colorMap)
}

func (s *GraphColorTestSuite) TestPartiallyPrecoloredColoredGraph(c *check.C) {
	// Ensure to use the same seed to make the test deterministic.
	rand.Seed(101)

	preColoredVertices := map[string]int{
		"0": 1,
		"3": 1,
	}
	adjMap := map[string][]string{
		"0": {"1", "2"},
		"1": {"2", "3"},
		"2": {"3"},
		"3": {"4"},
	}
	outDeg := s.setupGraph(c, adjMap, preColoredVertices)

	colorMap := make(map[string]int)
	numOfColors, err := s.assigner.AssignColors(context.TODO(), func(vertexID string, color int) {
		colorMap[vertexID] = color
		if fixedColor := preColoredVertices[vertexID]; fixedColor != 0 {
			c.Assert(color, check.Equals, fixedColor, check.Commentf("pre-colored vertex %v color was overwritten from %d to %d", vertexID, fixedColor, color))
		}
	})
	c.Assert(err, check.IsNil)

	maxColors := outDeg + 1
	c.Assert(numOfColors <= maxColors, check.Equals, true, check.Commentf("number of colors should not exceed (max vertex out degree + 1)"))
	assertNoColorConflictWithNeibours(c, adjMap, colorMap)
}

func (s *GraphColorTestSuite) setupGraph(c *check.C, adjMap map[string][]string, precoloredVertices map[string]int) int {
	uniqueVertices := make(map[string]struct{})
	for src, destinations := range adjMap {
		uniqueVertices[src] = struct{}{}
		for _, dest := range destinations {
			uniqueVertices[dest] = struct{}{}
		}
	}

	if precoloredVertices == nil {
		precoloredVertices = make(map[string]int)
	}

	for id := range uniqueVertices {
		if fixedColor := precoloredVertices[id]; fixedColor != 0 {
			s.assigner.AddPrecoloredVertex(id, fixedColor)
		} else {
			s.assigner.AddVertex(id)
		}
	}

	for src, destinations := range adjMap {
		for _, dest := range destinations {
			c.Assert(s.assigner.AddUndirectedEdge(src, dest), check.IsNil)
		}
	}

	var maxOutDeg int
	for _, v := range s.assigner.Graph().Vertices() {
		if deg := len(v.Edges()); deg > maxOutDeg {
			maxOutDeg = deg
		}
	}

	return maxOutDeg
}

func assertNoColorConflictWithNeibours(c *check.C, adjMap map[string][]string, colorMap map[string]int) {
	for srcID, srcColor := range colorMap {
		c.Assert(srcColor, check.Not(check.Equals), 0, check.Commentf("no color assigned to vertex %v", srcID))

		for _, destID := range adjMap[srcID] {
			c.Assert(colorMap[destID], check.Not(check.Equals), srcColor, check.Commentf("neighbor vertex %d assigned same color %d as vertex %d", destID, srcColor, srcID))
		}
	}
}
