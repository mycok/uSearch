package bspgraph_test

import (
	"context"
	// "errors"
	// "fmt"
	"testing"

	// "github.com/mycok/uSearch/internal/bspgraph/aggregator"
	"github.com/mycok/uSearch/internal/bspgraph/message"
	"github.com/mycok/uSearch/internal/bspgraph"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(GraphTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type GraphTestSuite struct {}

func (s *GraphTestSuite) TestMessageExchange(c *check.C) {
	g, err := bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeFunc: func(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
			v.Freeze()
			if g.Superstep() == 0 {
				var dest string
				switch v.ID() {
				case "0":
					dest = "1"
				case "1":
					dest = "0"
				}

				return g.SendMessage(dest, &intMsg{value: 11})
			}

			for msgIt.Next() {
				v.SetValue(msgIt.Message().(*intMsg).value)
			}

			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func () { c.Assert(g.Close(), check.IsNil) }()

	g.AddVertex("0", 0)
	g.AddVertex("1", 0)

	err = executeFixedSteps(g, 2)
	c.Assert(err, check.IsNil)

	for id, v := range g.Vertices() {
		c.Assert(v.Value(), check.Equals, 11, check.Commentf("vertex %v", id))
	}
}

// .......
type intMsg struct {
	value int
}

func (m intMsg) Type() string { return "intMsg"}


type localRelayer struct {
	relayErr error
	to *bspgraph.Graph
}

func (r localRelayer) Relay(destID string, msg message.Message) error {
	if r.relayErr != nil {
		return r.relayErr
	}

	return r.to.SendMessage(destID, msg)
}

func executeFixedSteps(g *bspgraph.Graph, numOfSteps int) error {
	exec := bspgraph.NewExecutor(g, bspgraph.ExecutorCallbacks{})

	return exec.RunSteps(context.TODO(), numOfSteps)
}
