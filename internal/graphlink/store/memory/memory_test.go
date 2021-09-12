package memory

import (
	"testing"

	"github.com/mycok/uSearch/internal/graphlink/graph/graphtests"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the InMemoryGraphTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(InMemoryDBTestSuite))

func Test(t *testing.T) {
	check.TestingT(t)
}

// InMemoryGraphTestSuite embeds the BaseSuite type tests methods.
type InMemoryDBTestSuite struct {
	graphtests.BaseSuite
}

// SetUpTest runs before each test in the test suite. it's
// responsible for setting up the requirements necessary for
// running that specific test. ie database reset.
func (s *InMemoryDBTestSuite) SetUpTest(c *check.C) {
	s.SetGraph(NewInMemoryGraph())
}
