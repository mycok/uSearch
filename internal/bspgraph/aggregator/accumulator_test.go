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

var _ = check.Suite(new(AccumulatorTestSuite))

type AccumulatorTestSuite struct{}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *AccumulatorTestSuite) TestFloat64Accumulator(c *check.C) {
	numOfValues := 100
	values := make([]interface{}, numOfValues)
	var expected float64
	for i := 0; i < numOfValues; i++ {
		next := rand.Float64()
		values[i] = next
		expected += next
	}

	recieved := s.testConcurrentAccess(new(Float64Accumulator), values).(float64)
	absDelta := math.Abs(expected - recieved)

	c.Assert(
		absDelta < 1e-6, check.Equals,
		true,
		check.Commentf("expected to get %f; got %f; |delta| %f > 1e-6", expected, recieved, absDelta),
	)
}

func (s *AccumulatorTestSuite) TestIntAccumulator(c *check.C) {
	numOfValues := 100
	values := make([]interface{}, numOfValues)
	var expected int
	for i := 0; i < numOfValues; i++ {
		next := rand.Int()
		values[i] = next
		expected += next
	}

	recieved := s.testConcurrentAccess(new(IntAccumulator), values).(int)
	c.Assert(expected, check.Equals, recieved)
}

func (s *AccumulatorTestSuite) testConcurrentAccess(a aggregator, values []interface{}) interface{} {
	startedCh := make(chan struct{})
	syncCh := make(chan struct{})
	doneCh := make(chan struct{})
	// Spin up a specific number of workers.
	for i := 0; i < len(values); i++ {
		go func(i int) {
			startedCh <- struct{}{}
			<-syncCh
			a.Aggregate(values[i])
			doneCh <- struct{}{}
		}(i)
	}

	// Wait for all go-routines to start, then start reading off the startedCh to allow other workers
	// to continue execution.
	for i := 0; i < len(values); i++ {
		<-startedCh
	}

	// Allow each go-routine to update the accumulator by closing the syncCh.
	close(syncCh)

	// Wait for all go-routines to exit by reading from the doneCh.
	for i := 0; i < len(values); i++ {
		<-doneCh
	}

	return a.Get()
}
