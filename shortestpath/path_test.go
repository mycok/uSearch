package shortestpath_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mycok/uSearch/shortestpath"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(shortestPathTestSuite))

type shortestPathTestSuite struct{}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *shortestPathTestSuite) TestShortestPathCalculation(c *check.C) {
	calc, err := shortestpath.NewCalculator(4)
	c.Assert(err, check.IsNil)

	for i := 0; i < 9; i++ {
		calc.AddVertex(fmt.Sprint(i))
	}

	// costMatrix[i][j] is the cost of an edge (if non zero) from i -> j. The
	// matrix is symmetric as the edges are un-directed.
	costMatrix := [][]int{
		{0, 4, 0, 0, 0, 0, 0, 8, 0},
		{4, 0, 8, 0, 0, 0, 0, 11, 0},
		{0, 8, 0, 7, 0, 4, 0, 0, 2},
		{0, 0, 7, 0, 9, 14, 0, 0, 0},
		{0, 0, 0, 9, 0, 10, 0, 0, 0},
		{0, 0, 4, 0, 10, 0, 2, 0, 0},
		{0, 0, 0, 14, 0, 2, 0, 1, 6},
		{8, 11, 0, 0, 0, 0, 1, 0, 7},
		{0, 0, 2, 0, 0, 0, 6, 7, 0},
	}

	for srcIdx, costs := range costMatrix {
		for destIdx, cost := range costs {
			if cost == 0 {
				continue
			}

			err := calc.AddEdge(fmt.Sprint(srcIdx), fmt.Sprint(destIdx), cost)
			c.Assert(err, check.IsNil)
		}
	}

	pathSrc := 0

	err = calc.CalculateShortestPaths(context.TODO(), fmt.Sprint(pathSrc))
	c.Assert(err, check.IsNil)

	expectedPaths := []struct {
		path []string
		cost int
	}{
		{
			path: []string{"0"},
			cost: 0,
		},
		{
			path: []string{"0", "1"},
			cost: 4,
		},
		{
			path: []string{"0", "1", "2"},
			cost: 12,
		},
		{
			path: []string{"0", "1", "2", "3"},
			cost: 19,
		},
		{
			path: []string{"0", "7", "6", "5", "4"},
			cost: 21,
		},
		{
			path: []string{"0", "7", "6", "5"},
			cost: 11,
		},
		{
			path: []string{"0", "7", "6"},
			cost: 9,
		},
		{
			path: []string{"0", "7"},
			cost: 8,
		},
		{
			path: []string{"0", "1", "2", "8"},
			cost: 14,
		},
	}

	for dest, exp := range expectedPaths {
		builtPath, cost, err := calc.BuildShortestPathTo(fmt.Sprint(dest))
		c.Assert(err, check.IsNil)
		c.Assert(builtPath, check.DeepEquals, exp.path)
		c.Assert(
			cost, check.Equals, exp.cost,
			check.Commentf("path from %d -> %d", pathSrc, dest),
		)
	}
}
