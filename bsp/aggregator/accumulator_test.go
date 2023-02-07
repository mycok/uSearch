package aggregator

import (
	"math"
	"math/rand"
	"testing"

	check "gopkg.in/check.v1"
)

type aggregator interface {
	Set(interface{})
	Get() interface{}
	Aggregate(interface{})
	Delta() interface{}
}

var _ = check.Suite(new(accumulatorTestSuite))

type accumulatorTestSuite struct{}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *accumulatorTestSuite) TestFloat64Accumulator(c *check.C) {
	var expected float64
	numOfValues := 100
	values := make([]interface{}, numOfValues)

	for i := 0; i < numOfValues; i++ {
		next := rand.Float64()
		values[i] = next
		expected += next
	}

	aggregated := testConcurrentAccumulatorAggregation(
		new(Float64Accumulator), values,
	).(float64)

	absDelta := math.Abs(expected - aggregated)

	// 1e–6 is the same as 0.000001 (one millionth).
	c.Assert(
		absDelta < 1e-6, check.Equals,
		true,
		check.Commentf("expected to get %f; got %f; |delta| %f > 1e-6", expected, aggregated, absDelta),
	)
}

func (s *accumulatorTestSuite) TestInt64Accumulator(c *check.C) {
	var expected int
	numOfValues := 100
	values := make([]interface{}, numOfValues)

	for i := 0; i < numOfValues; i++ {
		next := rand.Int()
		values[i] = next
		expected += next
	}

	aggregated := testConcurrentAccumulatorAggregation(
		new(IntAccumulator), values,
	).(int)

	c.Assert(expected, check.Equals, aggregated)
}

func testConcurrentAccumulatorAggregation(a aggregator, values []interface{}) interface{} {
	startChan := make(chan struct{})
	syncChan := make(chan struct{})
	doneChan := make(chan struct{})

	for i := 0; i < len(values); i++ {
		go func(index int) {
			startChan <- struct{}{}
			<-syncChan
			a.Aggregate(values[index])
			doneChan <- struct{}{}
		}(i)
	}

	for i := 0; i < len(values); i++ {
		<-startChan
	}

	close(syncChan)

	for i := 0; i < len(values); i++ {
		<-doneChan
	}

	return a.Get()
}
