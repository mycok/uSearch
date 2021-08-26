package memory

import (
	"testing"

	"github.com/mycok/uSearch/internal/graphlink/graph/graphtests"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the InMemoryGraphTestSuite to be
// executed by check testing package
var _ = check.Suite(new(InMemoryDBTestSuite))

// InMemoryGraphTestSuite encapsulates the BaseSuite type tests methods
type InMemoryDBTestSuite struct {
	graphtests.BaseSuite
}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *InMemoryDBTestSuite) SetUpTest(c *check.C) {
	s.SetGraph(NewInMemoryGraph())
}
