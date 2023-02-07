package color_test

import (
	"context"
	"math/rand"
	"testing"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/color"
)

var _ = check.Suite(new(colorTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type colorTestSuite struct {
	assigner *color.Assigner
}

func (s *colorTestSuite) SetUpTest(c *check.C) {
	assigner, err := color.NewColorAssigner(16)
	c.Assert(err, check.IsNil)

	s.assigner = assigner
}

func (s *colorTestSuite) TearDownTest(c *check.C) {
	c.Assert(s.assigner.Close(), check.IsNil)
}

func (s *colorTestSuite) TestUncoloredGraph(c *check.C) {
	// Ensure to use the same seed to make the test deterministic.
	rand.Seed(42)
	vertexToEdges := map[string][]string{
		"0": {"1", "2"},
		"1": {"2", "3"},
		"2": {"3"},
		"3": {"4"},
	}

	maxEdges := s.populateGraph(c, vertexToEdges, nil)
	vertexToColor := make(map[string]int)

	totalAssignedColors, err := s.assigner.AssignColors(context.TODO(), func(vertexID string, color int) {
		vertexToColor[vertexID] = color
	})
	c.Assert(err, check.IsNil)

	maxAssignedColors := maxEdges + 1

	c.Assert(totalAssignedColors <= maxAssignedColors, check.Equals, true)
	assertNoColorConflictsBetweenVertices(c, vertexToEdges, vertexToColor)
}

func (s *colorTestSuite) TestPartiallyPreColoredGraph(c *check.C) {
	// Ensure to use the same seed to make the test deterministic.
	rand.Seed(101)

	preColoredVertices := map[string]int{
		"0": 1,
		"3": 1,
	}

	vertexToEdges := map[string][]string{
		"0": {"1", "2"},
		"1": {"2", "3"},
		"2": {"3"},
		"3": {"4"},
	}

	maxEdges := s.populateGraph(c, vertexToEdges, preColoredVertices)
	vertexToColor := make(map[string]int)

	totalAssignedColors, err := s.assigner.AssignColors(context.TODO(), func(vertexID string, color int) {
		vertexToColor[vertexID] = color
		if vertexColor := vertexToColor[vertexID]; vertexColor != 0 {
			c.Assert(
				vertexColor, check.Equals, color,
				check.Commentf(
					"pre-colored vertex %v color was overwritten from %d to %d",
					vertexID, vertexColor, color,
				),
			)
		}
	})
	c.Assert(err, check.IsNil)

	maxAssignedColors := maxEdges + 1

	c.Assert(totalAssignedColors <= maxAssignedColors, check.Equals, true)
	assertNoColorConflictsBetweenVertices(c, vertexToEdges, vertexToColor)
}

func (s *colorTestSuite) populateGraph(
	c *check.C, vertexToEdges map[string][]string,
	preColoredVertices map[string]int,
) int {

	uniqueVertices := make(map[string]struct{})
	for src, destinations := range vertexToEdges {
		uniqueVertices[src] = struct{}{}
		for _, dest := range destinations {
			uniqueVertices[dest] = struct{}{}
		}
	}

	if preColoredVertices == nil {
		preColoredVertices = make(map[string]int)
	}

	// Add vertices to the graph.
	for id := range uniqueVertices {
		if vertexColor := preColoredVertices[id]; vertexColor != 0 {
			s.assigner.AddPreColoredVertex(id, vertexColor)
		} else {
			s.assigner.AddVertex(id)
		}
	}

	// Add vertex edges to the graph.
	for src, destinations := range vertexToEdges {
		for _, dest := range destinations {
			c.Assert(s.assigner.AddEdge(src, dest), check.IsNil)
		}
	}

	var maxEdges int
	for _, v := range s.assigner.Graph().Vertices() {
		if edges := len(v.Edges()); edges > maxEdges {
			maxEdges = edges
		}
	}

	return maxEdges
}

func assertNoColorConflictsBetweenVertices(
	c *check.C, vertexToEdges map[string][]string,
	vertexToColor map[string]int,
) {

	for srcID, srcColor := range vertexToColor {
		c.Assert(
			srcColor, check.Not(check.Equals), 0,
			check.Commentf("no color assigned to vertex %v", srcID),
		)

		for _, destID := range vertexToEdges[srcID] {
			c.Assert(
				vertexToColor[destID], check.Not(check.Equals), srcColor,
				check.Commentf(
					"neighbor vertex %d assigned same color %d as src vertex %d",
					destID, srcColor, srcID,
				),
			)
		}
	}
}
