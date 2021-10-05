package bspgraph_test

import (
	"context"
	// "errors"
	"fmt"
	"testing"

	"github.com/mycok/uSearch/internal/bspgraph"
	"github.com/mycok/uSearch/internal/bspgraph/aggregator"
	"github.com/mycok/uSearch/internal/bspgraph/message"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(new(GraphTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type GraphTestSuite struct{}

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
	defer func() { c.Assert(g.Close(), check.IsNil) }()

	g.AddVertex("0", 0)
	g.AddVertex("1", 0)

	err = executeFixedSteps(g, 2)
	c.Assert(err, check.IsNil)

	for id, v := range g.Vertices() {
		c.Assert(v.Value(), check.Equals, 11, check.Commentf("vertex %v", id))
	}
}

func (s *GraphTestSuite) TestMessageBroadcast(c *check.C) {
	g, err := bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeFunc: func(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
			if err := g.BroadcastToNeighbors(v, &intMsg{value: 11}); err != nil {
				return nil
			}

			for msgIt.Next() {
				v.SetValue(msgIt.Message().(*intMsg).value)
			}

			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g.Close(), check.IsNil) }()

	g.AddVertex("0", 11)
	g.AddVertex("1", 0)
	g.AddVertex("2", 0)
	g.AddVertex("3", 0)

	// Add edges to a single vertex.
	c.Assert(g.AddEdge("0", "1", nil), check.IsNil)
	c.Assert(g.AddEdge("0", "2", nil), check.IsNil)
	c.Assert(g.AddEdge("0", "3", nil), check.IsNil)

	err = executeFixedSteps(g, 2)
	c.Assert(err, check.IsNil)

	for id, v := range g.Vertices() {
		c.Assert(v.Value(), check.Equals, 11, check.Commentf("vertex %v", id))
	}
}

func (s *GraphTestSuite) TestGraphAggregator(c *check.C) {
	g, err := bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeWorkers: 4,
		ComputeFunc: func(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
			g.Aggregator("counter").Aggregate(1)

			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g.Close(), check.IsNil) }()

	offset := 5
	g.RegisterAggregator("counter", new(aggregator.IntAccumulator))
	g.Aggregator("counter").Aggregate(offset)

	numOfVertices := 1000
	for i := 0; i < numOfVertices; i++ {
		g.AddVertex(fmt.Sprint(i), nil)
	}

	err = executeFixedSteps(g, 1)
	c.Assert(err, check.IsNil)
	c.Assert(g.Aggregators()["counter"].Get(), check.Equals, numOfVertices+offset)
}

func (s *GraphTestSuite) TestGraphMessageRelay(c *check.C) {
	g1, err := bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeFunc: func(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
			if g.Superstep() == 0 {
				for _, e := range v.Edges() {
					_ = g.SendMessage(e.DestID(), &intMsg{value: 42})
				}
				return nil
			}

			for msgIt.Next() {
				v.SetValue(msgIt.Message().(*intMsg).value)
			}
			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g1.Close(), check.IsNil) }()

	g2, err := bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeFunc: func(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
			for msgIt.Next() {
				m := msgIt.Message().(*intMsg)
				v.SetValue(m.value)
				_ = g.SendMessage("graph1.vertex", m)
			}
			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g2.Close(), check.IsNil) }()

	g1.AddVertex("graph1.vertex", nil)
	c.Assert(g1.AddEdge("graph1.vertex", "graph2.vertex", nil), check.IsNil)
	g1.RegisterRelayer(localRelayer{to: g2})

	g2.AddVertex("graph2.vertex", nil)
	g2.RegisterRelayer(localRelayer{to: g1})

	// Exec both graphs in lockstep for 3 steps.
	// Step 0: g1 sends message to g2.
	// Step 1: g2 receives the message, updates its value and sends message
	//         back to g1.
	// Step 2: g1 receives message and updates its value.
	syncCh := make(chan struct{})
	ex1 := bspgraph.NewExecutor(g1, bspgraph.ExecutorCallbacks{
		PreStep: func(context.Context, *bspgraph.Graph) error {
			syncCh <- struct{}{}
			return nil
		},
		PostStep: func(context.Context, *bspgraph.Graph, int) error {
			syncCh <- struct{}{}
			return nil
		},
	})
	ex2 := bspgraph.NewExecutor(g2, bspgraph.ExecutorCallbacks{
		PreStep: func(context.Context, *bspgraph.Graph) error {
			<-syncCh
			return nil
		},
		PostStep: func(context.Context, *bspgraph.Graph, int) error {
			<-syncCh
			return nil
		},
	})

	ex1DoneCh := make(chan struct{})
	go func() {
		err := ex1.RunSteps(context.TODO(), 3)
		c.Assert(err, check.IsNil)
		close(ex1DoneCh)
	}()

	err = ex2.RunSteps(context.TODO(), 3)
	c.Assert(err, check.IsNil)
	<-ex1DoneCh

	c.Assert(g1.Vertices()["graph1.vertex"].Value(), check.Equals, 42)
	c.Assert(g2.Vertices()["graph2.vertex"].Value(), check.Equals, 42)
}

// .......
type intMsg struct {
	value int
}

func (m intMsg) Type() string { return "intMsg" }

type localRelayer struct {
	relayErr error
	to       *bspgraph.Graph
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
