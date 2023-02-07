package bsp_test

import (
	"context"
	"fmt"
	"time"

	"errors"
	"testing"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/bsp"
	"github.com/mycok/uSearch/bsp/aggregator"
	"github.com/mycok/uSearch/bsp/queue"
)

var _ = check.Suite(new(bspGraphTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

type bspGraphTestSuite struct{}

func (s *bspGraphTestSuite) TestMessageExchange(c *check.C) {
	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeFn: func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
			v.Freeze()
			if g.SuperStep() == 0 {
				var dest string

				switch v.ID() {
				case "0":
					dest = "1"
				case "1":
					dest = "0"
				}

				return g.SendMessage(dest, intMsg{value: 11})
			}

			for msgIt.Next() {
				v.SetValue(msgIt.Message().(intMsg).value)
			}

			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g.Close(), check.IsNil) }()

	g.AddVertex("0", 0)
	g.AddVertex("1", 1)

	err = executeFixedSteps(g, 2)
	c.Assert(err, check.IsNil)

	// Assert that the vertex values were updated with the message value.
	for id, vtx := range g.Vertices() {
		c.Assert(vtx.Value(), check.Equals, 11, check.Commentf("vertex %v", id))
	}
}

func (s *bspGraphTestSuite) TestMessageBroadcasting(c *check.C) {
	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeFn: func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
			if err := g.BroadcastToNeighbors(v, intMsg{value: 11}); err != nil {
				return err
			}

			for msgIt.Next() {
				v.SetValue(msgIt.Message().(intMsg).value)
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

	for id, vtx := range g.Vertices() {
		c.Assert(vtx.Value(), check.Equals, 11, check.Commentf("vertex %v", id))
	}
}

func (s *bspGraphTestSuite) TestAggregationWithComputeFunc(c *check.C) {
	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeWorkers: 4,
		ComputeFn: func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
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

	aggregatorMap := g.Aggregators()
	c.Assert(aggregatorMap["counter"].Get(), check.Equals, offset+numOfVertices)
}

func (s *bspGraphTestSuite) TestMessageRelay(c *check.C) {
	g1, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeFn: func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
			if g.SuperStep() == 0 {
				for _, e := range v.Edges() {
					_ = g.SendMessage(e.DestID(), intMsg{value: 11})
				}

				return nil
			}

			for msgIt.Next() {
				v.SetValue(msgIt.Message().(intMsg).value)
			}

			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g1.Close(), check.IsNil) }()

	g2, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeFn: func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
			for msgIt.Next() {
				m := msgIt.Message().(intMsg)
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

	// Execute both graphs in lockstep for 3 steps.
	// Step 0: g1 sends message to g2.
	// Step 1: g2 receives the message, updates its value and sends message
	//         back to g1.
	// Step 2: g1 receives message and updates its value.
	syncChan := make(chan struct{})
	exec1 := bsp.NewExecutor(g1, bsp.ExecutorCallbacks{
		PreStep: func(ctx context.Context, g *bsp.Graph) error {
			syncChan <- struct{}{}

			return nil
		},

		PostStep: func(ctx context.Context, g *bsp.Graph, activeInStep int) error {
			syncChan <- struct{}{}

			return nil
		},
	})

	exec2 := bsp.NewExecutor(g2, bsp.ExecutorCallbacks{
		PreStep: func(ctx context.Context, g *bsp.Graph) error {
			syncChan <- struct{}{}

			return nil
		},

		PostStep: func(ctx context.Context, g *bsp.Graph, activeInStep int) error {
			syncChan <- struct{}{}

			return nil
		},
	})

	go func() {
		for {
			select {
			case <-syncChan:
			case <-time.After(5 * time.Second):
			}
		}
	}()

	exec1DoneChan := make(chan struct{})
	go func() {
		err := exec1.RunSteps(context.TODO(), 3)
		c.Assert(err, check.IsNil)

		close(exec1DoneChan)
	}()

	err = exec2.RunSteps(context.TODO(), 3)
	c.Assert(err, check.IsNil)
	<-exec1DoneChan

	c.Assert(g1.Vertices()["graph1.vertex"].Value(), check.Equals, 11)
	c.Assert(g2.Vertices()["graph2.vertex"].Value(), check.Equals, 11)
}

func (s *bspGraphTestSuite) TestComputeFuncErrorHandling(c *check.C) {
	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeWorkers: 4,
		ComputeFn: func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
			if v.ID() == "50" {
				return errors.New("something went wrong")
			}
			return nil
		},
	})
	c.Assert(err, check.IsNil)
	defer func() { c.Assert(g.Close(), check.IsNil) }()

	numOfVertices := 1000
	for i := 0; i < numOfVertices; i++ {
		g.AddVertex(fmt.Sprint(i), nil)
	}

	err = executeFixedSteps(g, 1)
	c.Assert(err, check.ErrorMatches, `running compute function for vertex "50" failed: something went wrong`)
}

type intMsg struct {
	value int
}

func (m intMsg) Type() string { return "intMsg" }

type localRelayer struct {
	relayErr error
	to       *bsp.Graph
}

func (r localRelayer) Relay(destID string, msg queue.Message) error {
	if r.relayErr != nil {
		return r.relayErr
	}

	return r.to.SendMessage(destID, msg)
}

func executeFixedSteps(g *bsp.Graph, numOfSteps int) error {
	exec := bsp.NewExecutor(g, bsp.ExecutorCallbacks{})

	return exec.RunSteps(context.TODO(), numOfSteps)
}
