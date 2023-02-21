package pagerank

import (
	"math"

	"github.com/mycok/uSearch/bsp"
	"github.com/mycok/uSearch/bsp/queue"
)

// Static and compile-time check to ensure ScoreMessage implements
// the Message interface.
var _ queue.Message = (*ScoreMessage)(nil)

// ScoreMessage distributes PageRank scores to neighbors.
type ScoreMessage struct {
	Score float64
}

// Type returns the type of this message.
func (m ScoreMessage) Type() string { return "score" }

// makeComputeFunc returns a ComputeFunc that uses the PageRank algorithm to
// calculate and assign ranks to vertices using the provided dampingFactor.
func makeComputeFunc(dampingFactor float64) bsp.ComputeFunc {
	return func(g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator) error {
		step := g.SuperStep()
		pageCountAggregator := g.Aggregator("page_count")

		// At step 0, we use add one to the aggregator for each vertex in the
		// graph. the updated aggregator may be used to determine the number of
		// vertices in the graph.
		if step == 0 {
			pageCountAggregator.Aggregate(1)

			return nil
		}

		// Runs after the first step.
		var (
			newScore  float64
			pageCount = float64(pageCountAggregator.Get().(int))
		)

		switch step {
		case 1:
			// At step 1, we evenly distribute the PageRank score [1] across all
			// vertices. Since the sum of all scores should be equal to 1, each
			// vertex is assigned an initial score of 1 / pageCount.
			newScore = 1.0 / pageCount
		default:
			// Process incoming messages and calculate new score.
			newScore = (1.0 - dampingFactor) / pageCount
			for msgIt.Next() {
				score := msgIt.Message().(ScoreMessage).Score
				newScore += dampingFactor * score
			}

			// Add accumulated residual page ranks from any dead-ends
			// encountered during the previous step.
			resAgg := g.Aggregator(residualInputAccName(step))
			newScore += dampingFactor * resAgg.Get().(float64)
		}

		absDelta := math.Abs(v.Value().(float64) - newScore)
		g.Aggregator("SAD").Aggregate(absDelta)

		v.SetValue(newScore)

		// If this is a dead-end (no outgoing links) we treat this link
		// as if it was being connected to all links in the graph.
		// Since we cannot broadcast a message to all vertices we will
		// add the per-vertex residual score to an accumulator and
		// integrate it into the scores calculated over the next round.
		numOutLinks := float64(len(v.Edges()))
		if numOutLinks == 0.0 {
			g.Aggregator(residualOutputAccName(step)).Aggregate(newScore / pageCount)

			return nil
		}

		// Otherwise, evenly distribute this vertex / nodes's score to all its
		// neighbors.
		return g.BroadcastToNeighbors(v, ScoreMessage{Score: newScore / numOutLinks})
	}
}

// residualOutputAccName returns the name of the accumulator where the
// residual PageRank scores for the specified superStep are to be written to.
func residualOutputAccName(superStep int) string {
	if superStep%2 == 0 {
		return "residual_0"
	}
	return "residual_1"
}

// residualInputAccName returns the name of the accumulator where the
// residual PageRank scores for the specified superStep are to be read from.
func residualInputAccName(superStep int) string {
	if (superStep+1)%2 == 0 {
		return "residual_0"
	}
	return "residual_1"
}
