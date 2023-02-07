package memory

import (
	"testing"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/linkgraph/graph/graphtest"
)

// Initialize and register an instance of the InMemoryGraphTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(inMemoryGraphTestSuite))

// Test registers the [check] library with the go testing library and enables
// the running of the test suite using the go testing library.
func Test(t *testing.T) {
	check.TestingT(t)
}

// InMemoryGraphTestSuite embeds and runs the BaseSuite tests methods.
type inMemoryGraphTestSuite struct {
	graphtest.BaseSuite
}

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the requirements necessary for
// running that specific test. ie database reset.
func (s *inMemoryGraphTestSuite) SetUpTest(c *check.C) {
	s.SetGraph(NewInMemoryGraph())
}
